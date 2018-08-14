// Copyright 2018 Tetrate Labs
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"sync"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/api/core/v1"
)

type (
	Store interface {
		sdk.Handler

		Insert(service *v1.Service)
		Delete(service *v1.Service)
	}

	store struct {
		// While any given store should only be accessed by a single goroutine, we need to read out the data from a
		// separate goroutine, so we need to guard against current updates while performing that read. There should never
		// be racing writers to a single store, though.
		m sync.RWMutex
		s map[string]*v1.Service
	}

	MultiClusterStore struct {
		m sync.RWMutex
		s map[string]store
	}
)

var _ Store = &store{}

func newStore() store { return store{m: sync.RWMutex{}, s: make(map[string]*v1.Service)} }

func NewMultiClusterStore() *MultiClusterStore {
	return &MultiClusterStore{
		m: sync.RWMutex{},
		s: make(map[string]store),
	}
}

func (m *MultiClusterStore) NewCluster(name string) Store {
	m.m.Lock()
	defer m.m.Unlock()

	if store, exists := m.s[name]; exists {
		return &store
	}
	s := newStore()
	m.s[name] = s
	return &s
}

// Names returns all of the service names in the multi cluster store, indexed by cluster.
// i.e.
// 		for cluster, services := range mcs.Names() {
// 			for service := range services {
// 				//...
// 			}
// 		}
func (m *MultiClusterStore) Names() map[string]map[string]struct{} {
	// get a roughly-right-sized alloc by taking a racy read of the map's length, then hold the lock to copy the data
	out := make(map[string]map[string]struct{}, len(m.s))
	m.m.RLock()
	for cluster, store := range m.s {
		store.m.RLock()
		c := make(map[string]struct{}, len(store.s))
		for svc := range store.s {
			c[svc] = struct{}{}
		}
		store.m.RUnlock()
		out[cluster] = c
	}
	m.m.RUnlock()
	return out
}

func (s *store) Insert(svc *v1.Service) {
	s.m.Lock()
	s.s[key(svc)] = svc
	s.m.Unlock()
}

func (s *store) Delete(service *v1.Service) {
	s.m.Lock()
	delete(s.s, key(service))
	s.m.Unlock()
}

func (s *store) Handle(ctx context.Context, event sdk.Event) error {
	switch cr := event.Object.(type) {
	case *v1.Service:
		if event.Deleted {
			s.Delete(cr)
		}
		s.Insert(cr)
	}
	return nil
}

func key(s *v1.Service) string {
	return s.Name + "." + s.Namespace
}
