package conf

import (
	"flag"
	"testing"

	"github.com/go-kratos/kratos/pkg/log"
)

func Test_ConfigInit(t *testing.T) {
	flag.Parse()
	if err := Init(); err != nil {
		log.Error("conf.Init() error(%v)", err)
		panic(err)
	}
}
