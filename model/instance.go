package model

import (
	"encoding/json"
	"github.com/go-kratos/kratos/pkg/ecode"
	"github.com/go-kratos/kratos/pkg/log"
	"sync"
	"time"
)

const (
	// InstanceStatusUP Ready to receive traffic
	InstanceStatusUP = uint32(1)
	// InstancestatusWating Intentionally shutdown for traffic
	InstancestatusWating = uint32(1) << 1
)

// Action Replicate type of node
type Action int

const (
	// Register Replicate the add action to all nodes
	Register Action = iota
	// Renew Replicate the heartbeat action to all nodes
	Renew
	// Cancel Replicate the cancel action to all nodes
	Cancel
	// Weight Replicate the Weight action to all nodes
	Weight
	// Delete Replicate the Delete action to all nodes
	Delete
	// Status Replicate the Status action to all nodes
	Status
)

type Instance struct {
	Region   string            `json:"region,omitempty"`
	Zone     string            `json:"zone,omitempty"`
	Env      string            `json:"env,omitempty"`
	AppID    string            `json:"app_id,omitempty"`
	Hostname string            `json:"hostname,omitempty"`
	Addrs    []string          `json:"addrs,omitempty"`
	Version  string            `json:"version,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`

	Status uint32 `json:"status,omitempty"`

	RegTimestamp   int64 `json:"reg_timestamp,omitempty"`
	UpTimestamp    int64 `json:"up_timestamp,omitempty"` // NOTE: It is the latest timestamp that status becomes UP.
	RenewTimestamp int64 `json:"renew_timestamp,omitempty"`
	DirtyTimestamp int64 `json:"dirty_timestamp,omitempty"`

	LatestTimestamp int64 `json:"latest_timestamp,omitempty"`
}

func NewInstance(arg *ArgRegister) (i *Instance) {
	now := time.Now().UnixNano()
	i = &Instance{
		Region:          arg.Region,
		Zone:            arg.Zone,
		Env:             arg.Env,
		AppID:           arg.AppID,
		Hostname:        arg.Hostname,
		Addrs:           arg.Addrs,
		Version:         arg.Version,
		Status:          arg.Status,
		RegTimestamp:    now,
		UpTimestamp:     now,
		LatestTimestamp: now,
		RenewTimestamp:  now,
		DirtyTimestamp:  now,
	}
	if arg.Metadata != "" {
		if err := json.Unmarshal([]byte(arg.Metadata), &i.Metadata); err != nil {
			log.Error("json unmarshal metadata err %v", err)
		}
	}
	return
}

// deep copy a new instance from old one
func copyInstance(oi *Instance) (ni *Instance) {
	ni = new(Instance)
	*ni = *oi
	ni.Addrs = make([]string, len(oi.Addrs))
	for i, add := range oi.Addrs {
		ni.Addrs[i] = add
	}
	ni.Metadata = make(map[string]string)
	for k, v := range oi.Metadata {
		ni.Metadata[k] = v
	}
	return
}

type InstanceInfo struct {
	Instances       map[string][]*Instance
	Scheduler       *Scheduler
	LatestTimestamp int64
}

func (i *Instance) filter(status uint32) bool {
	return status&i.Status > 0
}

type App struct {
	AppID     string
	Zone      string
	instances map[string]*Instance

	lock            sync.RWMutex
	latestTimestamp int64
}

func NewApp(appid, zone string) (a *App) {
	a = &App{
		AppID:     appid,
		Zone:      zone,
		instances: make(map[string]*Instance),
	}
	return
}

func (a *App) Instances() (is []*Instance) {
	a.lock.RUnlock()
	defer a.lock.RUnlock()
	is = make([]*Instance, 0, len(a.instances))
	for _, i := range a.instances {
		ni := new(Instance)
		*ni = *i
		is = append(is, ni)
	}
	return
}

// NewInstance 注册一个实例
func (a *App) NewInstance(ni *Instance, latestTime int64) (i *Instance, ok bool) {
	i = new(Instance)
	a.lock.Lock()
	defer a.lock.Unlock()
	oi, ok := a.instances[ni.Hostname]
	if ok {
		ni.UpTimestamp = oi.UpTimestamp
		if ni.LatestTimestamp < oi.LatestTimestamp {
			log.Warn("register exist(%v) dirty timestamp over than caller(%v)", oi, ni)
			ni = oi
		}
	}
	a.instances[ni.Hostname] = ni
	a.updateLatest(latestTime)
	*i = *ni
	ok = !ok
	return
}

func (a *App) Renew(hostname string) (i *Instance, ok bool) {
	i = new(Instance)
	a.lock.Lock()
	defer a.lock.Unlock()
	oi, ok := a.instances[hostname]
	if !ok {
		return
	}
	oi.RenewTimestamp = time.Now().UnixNano()
	i = copyInstance(oi)
	return
}

func (a *App) updateLatest(latestTime int64) {
	if latestTime < a.latestTimestamp {
		latestTime = a.latestTimestamp + 1
	}
	a.latestTimestamp = latestTime
}

func (a *App) Cancel(hostname string, latestTime int64) (i *Instance, l int, ok bool) {
	i = new(Instance)
	a.lock.Lock()
	defer a.lock.Unlock()
	oi, ok := a.instances[hostname]
	if !ok {
		return
	}
	delete(a.instances, hostname)
	l = len(a.instances)
	oi.LatestTimestamp = latestTime
	a.updateLatest(latestTime)
	*i = *oi
	return
}

func (a *App) Len() (l int) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	l = len(a.instances)
	return
}

func (a *App) Set(changes *ArgSet) (ok bool) {
	a.lock.Lock()
	defer a.lock.Unlock()
	var (
		dst     *Instance
		setTime = changes.SetTimestamp
	)

	for i, hostname := range changes.Hostname {
		if dst, ok = a.instances[hostname]; !ok {
			return
		}
		if len(changes.Status) > 0 {
			if uint32(changes.Status[i]) != InstanceStatusUP && uint32(changes.Status[i]) != InstancestatusWating {
				log.Error("SetWeight change status(%d) is error", changes.Status[i])
				ok = false
			}
			dst.Status = uint32(changes.Status[i])
			if dst.Status == InstanceStatusUP {
				dst.UpTimestamp = setTime
			}
		}
		if len(changes.Metadata) > 0 {
			if err := json.Unmarshal([]byte(changes.Metadata[i]), &dst.Metadata); err != nil {
				log.Error("set change metadata err %s", changes.Metadata[i])
				ok = false
				return
			}
		}
		dst.LatestTimestamp = setTime
		dst.DirtyTimestamp = setTime
	}
	a.updateLatest(setTime)
	return
}

type Apps struct {
	apps            map[string]*App
	lock            sync.RWMutex
	latestTimestamp int64
}

func NewApps() *Apps {
	return &Apps{
		apps: make(map[string]*App),
	}
}

func (p *Apps) NewApp(zone, appid string, lts int64) (a *App, new bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	a, ok := p.apps[zone]
	if !ok {
		a = NewApp(appid, zone)
		p.apps[zone] = a
	}
	if lts <= p.latestTimestamp {
		lts = p.latestTimestamp + 1
	}
	p.latestTimestamp = lts
	new = !ok
	return
}

func (p *Apps) App(zone string) (as []*App) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if zone != "" {
		a, ok := p.apps[zone]
		if !ok {
			return
		}
		as = []*App{a}
	} else {
		for _, app := range p.apps {
			as = append(as, app)
		}
	}
	return
}

func (p *Apps) UpdateLatest(lts int64) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if lts <= p.latestTimestamp {
		lts = p.latestTimestamp + 1
	}
	p.latestTimestamp = lts
}

func (p *Apps) Del(zone string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.apps, zone)
}

func (p *Apps) InstanceInfo(zone string, lts int64, status uint32) (ci *InstanceInfo, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if lts >= p.latestTimestamp {
		err = ecode.NotModified
		return
	}
	ci = &InstanceInfo{
		LatestTimestamp: p.latestTimestamp,
		Instances:       map[string][]*Instance{},
	}
	var ok bool
	for z, app := range p.apps {
		if z == zone || zone == "" {
			ok = true
			instances := make([]*Instance, 0)
			for _, i := range app.instances {
				if i.filter(status) {
					ni := copyInstance(i)
					instances = append(instances, ni)
				}
			}
			ci.Instances[zone] = instances
		}
	}
	if !ok {
		err = ecode.NothingFound
	} else if len(ci.Instances) == 0 {
		err = ecode.NotModified
	}
	return
}
