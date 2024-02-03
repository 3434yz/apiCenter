package registry

import (
	"context"
	"fmt"
	"math/rand"

	"apiCenter/conf"
	"apiCenter/model"

	"github.com/go-kratos/kratos/pkg/sync/errgroup"
)

type Nodes struct {
	nodes    []*Node
	zones    map[string][]*Node // order zone
	selfAddr string
}

func NewNodes(c *conf.Config) *Nodes {
	nodes := make([]*Node, 0, len(c.Nodes))
	for _, addr := range c.Nodes {
		n := newNode(c, addr)
		n.zone = c.Env.Zone
		n.pRegisterURL = fmt.Sprintf("http://%s%s", c.HTTPServer.Addr, _registerURL)
		nodes = append(nodes, n)
	}
	zones := make(map[string][]*Node)
	for name, addrs := range c.Zones {
		zns := make([]*Node, 0, len(addrs))
		for _, addr := range addrs {
			n := newNode(c, addr)
			n.otherZone = true
			n.zone = name
			n.pRegisterURL = fmt.Sprintf("http://%s%s", c.HTTPServer.Addr, _registerURL)
			zns = append(zns, n)
		}
		zones[name] = zns
	}
	return &Nodes{
		nodes:    nodes,
		zones:    zones,
		selfAddr: c.HTTPServer.Addr,
	}
}

func (ns *Nodes) Replicate(c context.Context, action model.Action, i *model.Instance, otherZone bool) (err error) {
	if len(ns.nodes) == 0 {
		return
	}
	eg := errgroup.WithContext(c)
	for _, node := range ns.nodes {
		if !ns.Myself(node.addr) {
			ns.action(eg, action, node, i)
		}
	}
	if !otherZone {
		for _, zns := range ns.zones {
			if n := len(zns); n > 0 {
				ns.action(eg, action, zns[rand.Intn(n)], i)
			}
		}
	}
	err = eg.Wait()
	return
}

func (ns *Nodes) ReplicateSet(c context.Context, argSet *model.ArgSet, otherZone bool) (err error) {
	if len(ns.nodes) == 0 {
		return
	}
	eg := errgroup.WithContext(c)
	for _, n := range ns.nodes {
		if !ns.Myself(n.addr) {
			node := n
			eg.Go(func(c context.Context) error {
				return node.Set(c, argSet)
			})
		}
	}
	if !otherZone {
		for _, zns := range ns.zones {
			if n := len(zns); n > 0 {
				node := zns[rand.Intn(n)]
				eg.Go(func(c context.Context) error {
					return node.Set(c, argSet)
				})
			}
		}
	}
	err = eg.Wait()
	return
}

func (ns *Nodes) action(eg *errgroup.Group, action model.Action, n *Node, i *model.Instance) {
	switch action {
	case model.Register:
		eg.Go(func(ctx context.Context) error {
			return n.Register(ctx, i)
		})
	case model.Renew:
		eg.Go(func(ctx context.Context) error {
			return n.Renew(ctx, i)
		})
	case model.Cancel:
		eg.Go(func(ctx context.Context) error {
			return n.Cancel(ctx, i)
		})
	}
}

func (ns *Nodes) Nodes() (nsi []*model.Node) {
	nsi = make([]*model.Node, 0, len(ns.nodes))
	for _, nd := range ns.nodes {
		if nd.otherZone {
			continue
		}
		node := &model.Node{
			Addr:   nd.addr,
			Status: nd.status,
			Zone:   nd.zone,
		}
		nsi = append(nsi, node)
	}
	return
}

func (ns *Nodes) AllNodes() (nsi []*model.Node) {
	nsi = make([]*model.Node, 0, len(ns.nodes))
	for _, nd := range ns.nodes {
		node := &model.Node{
			Addr:   nd.addr,
			Status: nd.status,
			Zone:   nd.zone,
		}
		nsi = append(nsi, node)
	}
	for _, zns := range ns.zones {
		if n := len(zns); n > 0 {
			nd := zns[rand.Intn(n)]
			node := &model.Node{
				Addr:   nd.addr,
				Status: nd.status,
				Zone:   nd.zone,
			}
			nsi = append(nsi, node)
		}
	}
	return
}

func (ns *Nodes) Myself(addr string) bool {
	return ns.selfAddr == addr
}

func (ns *Nodes) Up() {
	for _, nd := range ns.nodes {
		if ns.Myself(nd.addr) {
			nd.status = model.NodeStatusUP
		}
	}
}
