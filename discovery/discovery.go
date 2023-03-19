package discovery

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"apiCenter/conf"
	"apiCenter/registry"
)

type Discovery struct {
	c         *conf.Config
	protected bool
	client    http.Client
	registry  *registry.Registry
	nodes     atomic.Value
}

func New(c *conf.Config) (d *Discovery, cancel context.CancelFunc) {
	d = &Discovery{
		c:         c,
		protected: c.EnableProtect,
		client:    http.NewClient(c.HttpClient),
		registry:  registry.NewRegistry(c),
	}
	d.nodes.Store(c.Nodes)
	d.syncUp()
	cancel = d.regSelf()
	go d.nodesproc()
	go d.exitProtect()
	return
}

func (d *Discovery) exitProtect() {
	// 受保护时间内只允许写不允许读
	time.Sleep(time.Second * 60)
	d.protected = false
}
