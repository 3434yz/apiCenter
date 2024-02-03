package model

import "encoding/json"

// NodeStatus Status of instances
type NodeStatus int

const (
	// NodeStatusUP Ready to receive register
	NodeStatusUP NodeStatus = iota
	// NodeStatusLost lost with each other
	NodeStatusLost
)

const (
	// AppID is discvoery id
	AppID = "infra.discovery"
)

type Node struct {
	Addr   string     `json:"addr,omitempty"`
	Status NodeStatus `json:"status,omitempty"`
	Zone   string     `json:"zone,omitempty"`
}

type Scheduler struct {
	AppID   string                   `json:"app_id,omitempty"`
	Env     string                   `json:"env,omitempty"`
	Clients map[string]*ZoneStrategy `json:"clients,omitempty"`
	Remark  string                   `json:"remark,omitempty"`
}

func (s *Scheduler) Set(content string) (err error) {
	return json.Unmarshal([]byte(content), &s)
}

type ZoneStrategy struct {
	Zones map[string]*Strategy `json:"zones,omitempty"`
}

type Strategy struct {
	Weight int64 `json:"weight,omitempty"`
}
