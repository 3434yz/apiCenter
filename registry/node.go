package registry

import (
	"apiCenter/conf"
	"apiCenter/model"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/pkg/ecode"
	"github.com/go-kratos/kratos/pkg/log"
	http "github.com/go-kratos/kratos/pkg/net/http/blademaster"
	"net/url"
	"strconv"
	"strings"
)

const (
	_registerURL = "/discovery/register"
	_cancelURL   = "/discovery/cancel"
	_renewURL    = "/discovery/renew"
	_setURL      = "/discovery/set"
)

type Node struct {
	c      *conf.Config
	client *http.Client

	pRegisterURL string
	registerURL  string
	renewURL     string
	cancelURL    string
	setURL       string

	zone      string
	addr      string
	status    model.NodeStatus
	otherZone bool
}

func newNode(c *conf.Config, addr string) (n *Node) {
	n = &Node{
		c:           c,
		client:      http.NewClient(c.HTTPClient),
		registerURL: fmt.Sprintf("http://%s%s", addr, _registerURL),
		renewURL:    fmt.Sprintf("http://%s%s", addr, _renewURL),
		cancelURL:   fmt.Sprintf("http://%s%s", addr, _cancelURL),
		setURL:      fmt.Sprintf("http://%s%s", addr, _setURL),

		addr:   addr,
		status: model.NodeStatusLost,
	}
	return
}

func (n *Node) call(c context.Context, action model.Action, i *model.Instance, uri string, data any) (err error) {
	params := url.Values{}
	params.Set("region", i.Region)
	params.Set("zone", i.Zone)
	params.Set("env", i.Env)
	params.Set("appid", i.AppID)
	params.Set("hostname", i.Hostname)
	params.Set("from_zone", "true")
	if n.otherZone {
		params.Set("replication", "false")
	} else {
		params.Set("replication", "true")
	}
	switch action {
	case model.Register:
		params.Set("addrs", strings.Join(i.Addrs, ","))
		params.Set("version", i.Version)
		meta, _ := json.Marshal(i.Metadata)
		params.Set("metadata", string(meta))
		params.Set("reg_timestamp", strconv.FormatInt(i.RegTimestamp, 10))
		params.Set("dirty_timestamp", strconv.FormatInt(i.DirtyTimestamp, 10))
		params.Set("latest_timestamp", strconv.FormatInt(i.LatestTimestamp, 10))
	case model.Renew:
		params.Set("dirty_timestamp", strconv.FormatInt(i.DirtyTimestamp, 10))
	case model.Cancel:
		params.Set("latest_timestamp", strconv.FormatInt(i.LatestTimestamp, 10))
	}

	var res struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err = n.client.Post(c, uri, "", params, &res); err != nil {
		log.Error("node be called(%s) instance(%v) error(%v)", uri, i, err)
		return
	}
	return
}

func (n *Node) setCall(c context.Context, arg *model.ArgSet, uri string) (err error) {
	params := url.Values{}
	params.Set("region", arg.Region)
	params.Set("zone", arg.Zone)
	params.Set("env", arg.Env)
	params.Set("appid", arg.AppID)
	params.Set("set_timestamp", strconv.FormatInt(arg.SetTimestamp, 10))
	params.Set("from_zone", "true")
	if n.otherZone {
		params.Set("replication", "false")
	} else {
		params.Set("replication", "true")
	}
	if len(arg.Hostname) != 0 {
		for _, name := range arg.Hostname {
			params.Add("hostname", name)
		}
	}
	if len(arg.Status) != 0 {
		for _, status := range arg.Status {
			params.Add("status", strconv.FormatInt(status, 10))
		}
	}
	if len(arg.Metadata) != 0 {
		for _, metadata := range arg.Metadata {
			params.Add("metadata", metadata)
		}
	}
	var res struct {
		Code int `json:"code"`
	}
	if err = n.client.Post(c, uri, "", params, &res); err != nil {
		log.Error("node be setCalled(%s) appid(%s) env (%s) error(%v)", uri, arg.AppID, arg.Env, err)
		return
	}
	if res.Code != 0 {
		log.Error("node be setCalled(%s) appid(%s) env (%s) responce code(%v)", uri, arg.AppID, arg.Env, res.Code)
		err = ecode.Int(res.Code)
	}
	return
}

func (n *Node) Register(c context.Context, i *model.Instance) (err error) {
	err = n.call(c, model.Register, i, n.registerURL, nil)
	if err != nil {
		log.Warn("node be called(%s) register instance(%v) error(%v)", n.registerURL, i, err)
	}
	return
}

func (n *Node) Cancel(c context.Context, i *model.Instance) (err error) {
	err = n.call(c, model.Cancel, i, n.cancelURL, nil)
	if err != nil {
		log.Warn("node be called(%s) instance(%v) already canceled", n.cancelURL, i)
	}
	return
}

func (n *Node) Renew(c context.Context, i *model.Instance) (err error) {
	var res *model.Instance
	err = n.call(c, model.Renew, i, n.renewURL, &res)
	if err == ecode.ServerErr {
		log.Warn("node be called(%s) instance(%v) error(%v)", n.renewURL, i, err)
		n.status = model.NodeStatusLost
		return
	}
	n.status = model.NodeStatusUP
	if err == ecode.NothingFound {
		log.Warn("node be called(%s) instance(%v) error(%v)", n.renewURL, i, err)
		err = n.call(c, model.Register, i, n.registerURL, nil)
		return
	}
	if err == ecode.Conflict && res != nil {
		err = n.call(c, model.Register, res, n.pRegisterURL, nil)
	}
	return
}

func (n *Node) Set(ctx context.Context, set *model.ArgSet) (err error) {
	return n.setCall(ctx, set, n.setURL)
}
