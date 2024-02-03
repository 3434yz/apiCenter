package registry

import (
	"apiCenter/model"
	"context"
	"github.com/go-kratos/kratos/pkg/conf/paladin"
	"strings"
	"sync"
)

type scheduler struct {
	schedulers map[string]*model.Scheduler
	mutex      sync.RWMutex
	r          *Registry
}

func newScheduler(r *Registry) *scheduler {
	return &scheduler{
		schedulers: make(map[string]*model.Scheduler),
		r:          r,
	}
}

func (s *scheduler) Load() {
	for _, key := range paladin.Keys() {
		if !strings.HasSuffix(key, ".json") {
			continue
		}
		v := paladin.Get(key)
		content, err := v.String()
		if err != nil {
			return
		}
		sch := new(model.Scheduler)
		if err := sch.Set(content); err != nil {
			continue
		}
		s.schedulers[appsKey(sch.AppID, sch.Env)] = sch
	}
}

func (s *scheduler) Reload() {
	event := paladin.WatchEvent(context.Background())
	for {
		e := <-event
		if strings.HasSuffix(e.Key, ".json") {
			continue
		}
		sch := new(model.Scheduler)
		if err := sch.Set(e.Value); err != nil {
			continue
		}
		s.mutex.Lock()
		key := appsKey(sch.AppID, sch.Env)
		s.r.alock.Lock()
		if a, ok := s.r.appm[key]; ok {
			a.UpdateLatest(0)
		}
		s.r.alock.Unlock()
		s.schedulers[key] = sch
		s.mutex.Unlock()
		s.r.broadcast(sch.Env, sch.AppID)
	}
}

func (s *scheduler) Get(appid, env string) *model.Scheduler {
	s.mutex.RLock()
	sch := s.schedulers[appsKey(appid, env)]
	s.mutex.RUnlock()
	return sch
}
