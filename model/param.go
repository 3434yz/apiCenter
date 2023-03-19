package model

// ArgRegister define register param.
type ArgRegister struct {
	Region          string   `form:"region"`
	Zone            string   `form:"zone" validate:"required"`
	Env             string   `form:"env" validate:"required"`
	AppID           string   `form:"appid" validate:"required"`
	Hostname        string   `form:"hostname" validate:"required"`
	Status          uint32   `form:"status" validate:"required"`
	Addrs           []string `form:"addrs" validate:"gt=0"`
	Version         string   `form:"version"`
	Metadata        string   `form:"metadata"`
	Replication     bool     `form:"replication"`
	LatestTimestamp int64    `form:"latest_timestamp"`
	DirtyTimestamp  int64    `form:"dirty_timestamp"`
	FromZone        bool     `form:"from_zone"`
}
