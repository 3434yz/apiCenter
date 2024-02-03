package registry

import (
	"apiCenter/conf"
	"fmt"
	"github.com/go-kratos/kratos/pkg/ecode"
	"github.com/go-kratos/kratos/pkg/log"
	"math/rand"
	"sync"
	"time"

	"apiCenter/model"
)

const (
	_evictThreshold = int64(90 * time.Second)
	_evictCeiling   = int64(3600 * time.Second)
)

type Registry struct {
	appm  map[string]*model.Apps
	alock sync.RWMutex

	conns     map[string]*hosts
	cLock     sync.RWMutex
	scheduler *scheduler
	gd        *Guard
}

type hosts struct {
	hclock sync.RWMutex
	hosts  map[string]*conn
}

type conn struct {
	ch         chan map[string]*model.InstanceInfo
	arg        *model.ArgPolls
	latestTime int64
	count      int
}

// newConn new consumer chan.
func newConn(ch chan map[string]*model.InstanceInfo, latestTime int64, arg *model.ArgPolls) *conn {
	return &conn{
		ch:         ch,
		arg:        arg,
		latestTime: latestTime,
		count:      1,
	}
}

func (r *Registry) newApps(appid, env string) (a *model.Apps, ok bool) {
	key := appsKey(appid, env)
	r.alock.Lock()
	defer r.alock.Unlock()
	a, ok = r.appm[key]
	if !ok {
		a = model.NewApps()
		r.appm[key] = a
	}
	return
}

func (r *Registry) apps(appid, env, zone string) (as []*model.App, a *model.Apps, ok bool) {
	key := appsKey(appid, env)
	r.alock.RLock()
	defer r.alock.RUnlock()
	a, ok = r.appm[key]
	if ok {
		as = a.App(zone)
	}
	return
}

func NewRegistry(conf *conf.Config) (r *Registry) {
	r = &Registry{
		appm:  make(map[string]*model.Apps),
		conns: make(map[string]*hosts),
		gd:    new(Guard),
	}
	r.scheduler = newScheduler(r)
	r.scheduler.Load()
	go r.scheduler.Reload()
	go r.proc()
	return
}

func (r *Registry) newApp(i *model.Instance) (a *model.App) {
	as, _ := r.newApps(i.AppID, i.Env)
	a, _ = as.NewApp(i.Zone, i.AppID, i.LatestTimestamp)
	return
}

func appsKey(appid, env string) string {
	return fmt.Sprintf("%s-%s", appid, env)
}

func pollKey(appid, env string) string {
	return fmt.Sprintf("%s-%s", env, appid)
}

func (r *Registry) Register(ins *model.Instance, latestTime int64) (err error) {
	app := r.newApp(ins)
	i, ok := app.NewInstance(ins, latestTime)
	if ok {
		r.gd.incrExp()
	}
	r.broadcast(i.Env, i.AppID)
	return
}

func (r *Registry) Renew(arg *model.ArgRenew) (i *model.Instance, ok bool) {
	as, _, _ := r.apps(arg.AppID, arg.Env, arg.Zone)
	if len(as) == 0 {
		return
	}
	if i, ok = as[0].Renew(arg.Hostname); !ok {
		return
	}
	r.gd.incrFac()
	return
}

func (r *Registry) Cancel(arg *model.ArgCancel) (i *model.Instance, ok bool) {
	if i, ok = r.cancel(arg.Zone, arg.Env, arg.AppID, arg.Hostname, arg.LatestTimestamp); !ok {
		return
	}
	r.gd.decrExp()
	return
}

func (r *Registry) cancel(zone, env, appid, hostname string, lts int64) (i *model.Instance, ok bool) {
	var l int
	a, as, _ := r.apps(appid, env, zone)
	if len(a) == 0 {
		return
	}
	if i, l, ok = a[0].Cancel(hostname, lts); !ok {
		return
	}
	as.UpdateLatest(lts)
	if l == 0 {
		if a[0].Len() == 0 {
			as.Del(zone)
		}
	}
	if len(as.App("")) == 0 {
		r.alock.Lock()
		delete(r.appm, appsKey(appid, env))
		r.alock.Unlock()
	}
	r.broadcast(env, appid)
	return
}

// FetchAll 根据AppID来拉取实例
func (r *Registry) FetchAll() (im map[string][]*model.Instance) {
	ass := r.allApps()
	im = make(map[string][]*model.Instance)
	for _, apps := range ass {
		for _, app := range apps.App("") {
			im[app.AppID] = append(im[app.AppID], app.Instances()...)
		}
	}
	return
}

// Fetch 拉取所有符合条件的实例
func (r *Registry) Fetch(zone, env, appid string, latestTime int64, status uint32) (info *model.InstanceInfo, err error) {
	key := appsKey(appid, env)
	r.alock.RLock()
	apps, ok := r.appm[key]
	r.alock.RUnlock()
	if !ok {
		err = ecode.NothingFound
		return
	}
	info, err = apps.InstanceInfo(zone, latestTime, status)
	if err != nil {
		return
	}
	sch := r.scheduler.Get(appid, env)
	if sch != nil {
		info.Scheduler = new(model.Scheduler)
		info.Scheduler.Clients = sch.Clients
	}
	return
}

func (r *Registry) Set(arg *model.ArgSet) (ok bool) {
	as, _, _ := r.apps(arg.AppID, arg.Env, arg.Zone)
	if len(as) == 0 {
		return
	}
	if ok = as[0].Set(arg); !ok {
		return
	}
	r.broadcast(arg.Env, arg.AppID)
	return
}

func (r *Registry) Polls(arg *model.ArgPolls) (ch chan map[string]*model.InstanceInfo, notify bool, miss []string, err error) {
	var ins = make(map[string]*model.InstanceInfo)
	if len(arg.AppID) != len(arg.LatestTimestamp) {
		arg.LatestTimestamp = make([]int64, len(arg.AppID))
	}
	for i := range arg.AppID {
		var ii *model.InstanceInfo
		ii, err = r.Fetch(arg.Zone, arg.Env, arg.AppID[i], arg.LatestTimestamp[i], model.InstanceStatusUP)
		if err == ecode.NothingFound {
			miss = append(miss, arg.AppID[i])
			continue
		}
		if err == nil {
			ins[arg.AppID[i]] = ii
			notify = true
		}
	}
	// 有更新，返回数据
	if notify {
		ch = make(chan map[string]*model.InstanceInfo, 5)
		ch <- ins
		return
	} else {
		for i := range arg.AppID {
			k := pollKey(arg.AppID[i], arg.Env)
			r.cLock.Lock()
			if _, ok := r.conns[k]; !ok {
				r.conns[k] = &hosts{hosts: make(map[string]*conn)}
			}
			hosts := r.conns[k]
			r.cLock.Unlock()

			// 找到conn进行监听
			hosts.hclock.Lock()
			conn, ok := hosts.hosts[arg.Hostname]
			if !ok {
				if ch == nil {
					ch = make(chan map[string]*model.InstanceInfo, 5)
				}
				conn = newConn(ch, arg.LatestTimestamp[i], arg)
				log.Info("Polls from(%s) new connection(%d)", arg.Hostname, conn.count)
			} else {
				conn.count++
				if ch == nil {
					ch = conn.ch
				}
				log.Info("Polls from(%s) reuse connection(%d)", arg.Hostname, conn.count)
			}
			hosts.hosts[arg.Hostname] = conn
			hosts.hclock.Unlock()
		}
	}
	return
}

func (r *Registry) broadcast(env, appid string) {
	key := pollKey(appid, env)
	r.cLock.Lock()
	conns, ok := r.conns[key]
	if !ok {
		r.cLock.Unlock()
		return
	}
	delete(r.conns, key)
	r.cLock.Unlock()
	conns.hclock.RLock()
	defer conns.hclock.RUnlock()
	for _, conn := range conns.hosts {
		ii, err := r.Fetch(conn.arg.Zone, env, appid, 0, model.InstanceStatusUP)
		if err != nil {
			continue
		}
		for i := 0; i < conn.count; i++ {
			select {
			case conn.ch <- map[string]*model.InstanceInfo{appid: ii}:
				log.Info("broadcast to(%s) success(%d)", conn.arg.Hostname, i+1)
			case <-time.After(500 * time.Millisecond):
				log.Info("broadcast to(%s) failed(%d) maybe chan full", conn.arg.Hostname, i+1)
			}
		}
	}
}

func (r *Registry) allApps() (ass []*model.Apps) {
	r.alock.RLock()
	defer r.alock.RUnlock()
	ass = make([]*model.Apps, 0, len(r.appm))
	for _, apps := range r.appm {
		ass = append(ass, apps)
	}
	return
}

func (r *Registry) resetExp() {
	cnt := int64(0)
	for _, p := range r.allApps() {
		for _, a := range p.App("") {
			cnt += int64(a.Len())
		}
	}
	r.gd.setExp(cnt)
}

func (r *Registry) proc() {
	tk := time.Tick(1 * time.Minute)
	tk2 := time.Tick(15 * time.Minute)
	for {
		select {
		case <-tk:
			r.gd.updateFac()
			r.evict()
		case <-tk2:
			r.resetExp()
		}
	}
}

func (r *Registry) evict() {
	protect := r.gd.ok()

	r.alock.Lock()
	defer r.alock.Unlock()

	var eis []*model.Instance
	var registrySize int

	all := r.allApps()
	for _, apps := range all {
		for _, app := range apps.App("") {
			registrySize += app.Len()
			is := app.Instances()
			for _, i := range is {
				delta := time.Now().UnixNano() - i.RenewTimestamp
				if (!protect && delta > _evictThreshold) || delta > _evictCeiling {
					eis = append(eis, i)
				}
			}
		}
	}

	eCnt := len(eis)
	evictionLimit := registrySize - int(float64(registrySize)*_percentThreshold)
	if eCnt > evictionLimit {
		eCnt = evictionLimit
	}
	if eCnt == 0 {
		return
	}
	for i := 0; i < eCnt; i++ {
		// Pick a random item (Knuth shuffle algorithm)
		next := i + rand.Intn(len(eis)-i)
		eis[i], eis[next] = eis[next], eis[i]
		ei := eis[i]
		r.cancel(ei.Zone, ei.Env, ei.AppID, ei.Hostname, time.Now().UnixNano())
	}
}

func (r *Registry) DelConns(arg *model.ArgPolls) {
	for i := range arg.AppID {
		key := pollKey(arg.AppID[i], arg.Env)
		r.cLock.Lock()
		hosts, ok := r.conns[key]
		if !ok {
			r.cLock.Unlock()
			continue
		}
		r.cLock.Unlock()
		hosts.hclock.Lock()
		if conn, ok := hosts.hosts[arg.Hostname]; ok {
			if conn.count > 1 {
				conn.count--
			} else {
				delete(hosts.hosts, arg.Hostname)
			}
		}
		hosts.hclock.Unlock()
	}
}
