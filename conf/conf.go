package conf

import (
	"github.com/BurntSushi/toml"
	"github.com/go-kratos/kratos/pkg/conf/env"
	"github.com/go-kratos/kratos/pkg/conf/paladin"
	log "github.com/go-kratos/kratos/pkg/log"
	http "github.com/go-kratos/kratos/pkg/net/http/blademaster"
)

var (
	configKey = "discovery.toml"

	Conf = &Config{}
)

type Env struct {
	Region    string
	Zone      string
	Host      string
	DeployEnv string
}

type Config struct {
	Nodes         []string
	Zones         map[string][]string
	HttpServer    *http.ServerConfig
	HttpClient    *http.ClientConfig
	Env           *Env
	Log           *log.Config
	Scheduler     []byte
	EnableProtect bool
}

func (c *Config) fix() (err error) {
	if c.Env == nil {
		c.Env = new(Env)
	}
	if c.Env.Region == "" {
		c.Env.Region = env.Region
	}
	if c.Env.Zone == "" {
		c.Env.Zone = env.Zone
	}
	if c.Env.Host == "" {
		c.Env.Host = env.Hostname
	}
	if c.Env.DeployEnv == "" {
		c.Env.DeployEnv = env.DeployEnv
	}
	return
}

func Init() (err error) {
	if err = paladin.Init(); err != nil {
		return
	}
	return paladin.Watch(configKey, Conf)
}

func (c *Config) Set(content string) (err error) {
	var tmpConf *Config
	if _, err = toml.Decode(content, &tmpConf); err != nil {
		log.Error("decode config fail %v", err)
		return
	}
	if err = tmpConf.fix(); err != nil {
		return
	}
	*Conf = *tmpConf
	return nil
}
