package model

import (
	"encoding/json"
	"time"

	log "github.com/go-kratos/kratos/pkg/log"
)

const (
	InstanceStatusUp     = uint32(1)
	InstanceStatusWating = uint32(1)
)

func (i *Instance) filter(status uint32) bool {
	return status&i.Status > 0
}

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
	Region   string            `json:"region"`
	Zone     string            `json:"zone"`
	Env      string            `json:"env"`
	AppID    string            `json:"appid"`
	Hostname string            `json:"hostname"`
	Addrs    []string          `json:"addrs"`
	Version  string            `json:"version"`
	Metadata map[string]string `json:"metadata"`

	// Status enum instance status
	Status uint32 `json:"status"`

	// timestamp
	RegTimestamp   int64 `json:"reg_timestamp"`
	UpTimestamp    int64 `json:"up_timestamp"` // 最后一次切换为up的时间戳
	RenewTimestamp int64 `json:"renew_timestamp"`
	DirtyTimestamp int64 `json:"dirty_timestamp"`

	LatestTimestamp int64 `json:"latest_timestamp"`
}

// NewInstance new a instance.
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
