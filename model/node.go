package model

const (
	AppID = "infra.discovery"
)

// Scheduler info.
type Scheduler struct {
	AppID   string                   `json:"app_id,omitempty"`
	Env     string                   `json:"env,omitempty"`
	Clients map[string]*ZoneStrategy `json:"clients"` // zone-ratio
	Remark  string                   `json:"remark"`
}
