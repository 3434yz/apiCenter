package discovery

import (
	"apiCenter/conf"
	"apiCenter/registry"
	"context"
	http "github.com/go-kratos/kratos/pkg/net/http/blademaster"
	"sync/atomic"
	"time"
)

type Discovery struct {
	c         *conf.Config
	protected bool
	client    *http.Client
	registry  *registry.Registry
	nodes     atomic.Value
}

func New(c *conf.Config) (d *Discovery, cancel context.CancelFunc) {
	d = &Discovery{
		protected: c.EnableProtect,
		c:         c,
		client:    http.NewClient(c.HTTPClient),
		registry:  registry.NewRegistry(c),
	}
	d.nodes.Store(registry.NewNodes(c))
	d.syncUp()
	cancel = d.regSelf()
	go d.nodesproc()
	go d.exitProtect()
	return
}

func (d *Discovery) exitProtect() {
	time.Sleep(time.Second * 60)
	d.protected = false
}
