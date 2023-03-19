package discovery

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bilibili/discovery/model"
	"github.com/bilibili/discovery/registry"
)

var (
	_fetchAllURL = "http://%s/discovery/fetch/all"
)

// Protected return if service in init protect mode.
// if service in init protect mode,only support write,
// read operator isn't supported.
func (d *Discovery) Protected() bool {
	return d.protected
}

// 从其他节点同步数据到本节点
func (d *Discovery) syncUp() {
	nodes := d.nodes.Load().(*registry.Nodes)
	for _, node := range nodes.AllNodes() {
		if nodes.Myself(node.Addr) {
			continue
		}
		uri := fmt.Sprintf(_fetchAllURL, node.Addr)
		var res struct {
			Code int                          `json:"code"`
			Data map[string][]*model.Instance `json:"data"`
		}
		if err := d.client.Get(context.TODO(), uri, "", nil, &res); err != nil {
			log.Error("d.client.Get(%v) error(%v)", uri, err)
			continue
		}
		if res.Code != 0 {
			log.Error("service syncup from(%s) failed ", uri)
			continue
		}
		// sync success from other node,exit protected mode
		d.protected = false
		for _, is := range res.Data {
			for _, ins := range is {
				_ = d.registry.Register(ins, ins.LatestTimestamp)
			}
		}
		// 不返回时确保所有的实例把其他实例注册到自身.
		nodes.UP()
	}
}

func (d *Discovery) regSelf() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now().UnixNano()
	ins := &model.Instance{}
}
