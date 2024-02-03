package model

type ArgRegister struct {
	Region          string   `json:"region,omitempty"`
	Zone            string   `json:"zone,omitempty"`
	Env             string   `json:"env,omitempty"`
	AppID           string   `json:"app_id,omitempty"`
	Hostname        string   `json:"hostname,omitempty"`
	Status          uint32   `json:"status,omitempty"`
	Addrs           []string `json:"addrs,omitempty"`
	Version         string   `json:"version,omitempty"`
	Metadata        string   `json:"metadata,omitempty"`
	Replication     bool     `json:"replication,omitempty"`
	LatestTimestamp int64    `json:"latest_timestamp,omitempty"`
	DirtyTimestamp  int64    `json:"dirty_timestamp,omitempty"`
	FromZone        bool     `json:"from_zone,omitempty"`
}

type ArgRenew struct {
	Zone           string `form:"zone" validate:"required"`
	Env            string `form:"env" validate:"required"`
	AppID          string `form:"appid" validate:"required"`
	Hostname       string `form:"hostname" validate:"required"`
	Replication    bool   `form:"replication"`
	DirtyTimestamp int64  `form:"dirty_timestamp"`
	FromZone       bool   `form:"from_zone"`
}

type ArgCancel struct {
	Zone            string `form:"zone" validate:"required"`
	Env             string `form:"env" validate:"required"`
	AppID           string `form:"appid" validate:"required"`
	Hostname        string `form:"hostname" validate:"required"`
	FromZone        bool   `form:"from_zone"`
	Replication     bool   `form:"replication"`
	LatestTimestamp int64  `form:"latest_timestamp"`
}

// ArgFetch define fetch param.
type ArgFetch struct {
	Zone   string `form:"zone"`
	Env    string `form:"env" validate:"required"`
	AppID  string `form:"appid" validate:"required"`
	Status uint32 `form:"status" validate:"required"`
}

// ArgFetchs define fetchs arg.
type ArgFetchs struct {
	Zone   string   `form:"zone"`
	Env    string   `form:"env" validate:"required"`
	AppID  []string `form:"appid" validate:"gt=0"`
	Status uint32   `form:"status" validate:"required"`
}

type ArgPolls struct {
	Zone            string   `json:"zone,omitempty"`
	Env             string   `json:"env,omitempty"`
	AppID           []string `json:"app_id,omitempty"`
	Hostname        string   `json:"hostname,omitempty"`
	LatestTimestamp []int64  `json:"latest_timestamp,omitempty"`
}

type ArgPoll struct {
	Zone            string `json:"zone,omitempty"`
	Env             string `json:"env,omitempty"`
	AppID           string `json:"app_id,omitempty"`
	Hostname        string `json:"hostname,omitempty"`
	LatestTimestamp int64  `json:"latest_timestamp,omitempty"`
}

type ArgSet struct {
	Region       string   `json:"region,omitempty"`
	Zone         string   `json:"zone,omitempty"`
	Env          string   `json:"env,omitempty"`
	AppID        string   `json:"app_id,omitempty"`
	Hostname     []string `json:"hostname,omitempty"`
	Status       []int64  `json:"status,omitempty"`
	Metadata     []string `json:"metadata,omitempty"`
	Replication  bool     `json:"replication,omitempty"`
	FromZone     bool     `json:"from_zone,omitempty"`
	SetTimestamp int64    `json:"set_timestamp,omitempty"`
}
