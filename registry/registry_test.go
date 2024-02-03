package registry

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"apiCenter/conf"
	"apiCenter/model"

	"github.com/go-kratos/kratos/pkg/conf/paladin"
	. "github.com/smartystreets/goconvey/convey"
)

var reg = &model.ArgRegister{AppID: "main.arch.test", Hostname: "reg", Zone: "sh0001", Env: "pre", Status: model.InstanceStatusUP}
var regH1 = &model.ArgRegister{AppID: "main.arch.test", Hostname: "regH1", Zone: "sh0001", Env: "pre", Status: 1}

var reg2 = &model.ArgRegister{AppID: "main.arch.test2", Hostname: "reg2", Zone: "sh0001", Env: "pre", Status: 1}

var arg = &model.ArgRenew{Zone: "sh0001", Env: "pre", AppID: "main.arch.test", Hostname: "reg"}
var cancel = &model.ArgCancel{Zone: "sh0001", Env: "pre", AppID: "main.arch.test", Hostname: "reg"}
var cancel2 = &model.ArgCancel{Zone: "sh0001", Env: "pre", AppID: "main.arch.test", Hostname: "regH1"}

func TestMain(m *testing.M) {
	flag.Set("conf", "./")
	flag.Parse()
	paladin.Init()
	m.Run()
	os.Exit(0)
}

func TestRegister(t *testing.T) {
	i := model.NewInstance(reg)
	register(t, i)
}

func TestDiscovery(t *testing.T) {
	i1 := model.NewInstance(reg)
	i2 := model.NewInstance(regH1)
	r := register(t, i1, i2)
	Convey("test discovery", t, func() {
		fetchArg := &model.ArgFetch{Zone: "sh0001", Env: "pre", AppID: "main.arch.test", Status: 3}
		info, err := r.Fetch(fetchArg.Zone, fetchArg.Env, fetchArg.AppID, 0, fetchArg.Status)
		So(err, ShouldBeNil)
		So(len(info.Instances["sh0001"]), ShouldEqual, 2)

		pollArg := &model.ArgPolls{Zone: "sh0001", Env: "pre", AppID: []string{"main.arch.test"}, Hostname: "test"}
		ch, _, _, err := r.Polls(pollArg)
		So(err, ShouldBeNil)
		apps := <-ch
		So(len(apps["main.arch.test"].Instances["sh0001"]), ShouldEqual, 2)
		pollArg.LatestTimestamp[0] = apps["main.arch.test"].LatestTimestamp
		fmt.Println(apps["main.arch.test"])

		r.Cancel(cancel)
		ch, _, _, err = r.Polls(pollArg)
		So(err, ShouldBeNil)
		apps = <-ch
		So(len(apps["main.arch.test"].Instances), ShouldEqual, 1)
		pollArg.LatestTimestamp[0] = apps["main.arch.test"].LatestTimestamp
		r.Cancel(cancel2)
	})
}

func TestRenew(t *testing.T) {
	src := model.NewInstance(reg)
	r := register(t, src)
	Convey("test renew", t, func() {
		i, ok := r.Renew(arg)
		So(ok, ShouldBeTrue)
		So(i, ShouldResemble, src)
	})
}

func register(t *testing.T, is ...*model.Instance) (r *Registry) {
	Convey("test register", t, func() {
		r = NewRegistry(&conf.Config{})
		var num int
		for _, i := range is {
			err := r.Register(i, 0)
			So(err, ShouldBeNil)
			if i.AppID == "main.arch.test" {
				num++
			}
		}
		fetchArg := &model.ArgFetch{Zone: "sh0001", Env: "pre", AppID: "main.arch.test", Status: 3}
		instancesInfo, err := r.Fetch(fetchArg.Zone, fetchArg.Env, fetchArg.AppID, 0, fetchArg.Status)
		So(err, ShouldBeNil)
		So(len(instancesInfo.Instances["sh0001"]), ShouldEqual, num)
	})
	return r
}
