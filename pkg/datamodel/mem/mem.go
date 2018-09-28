// Copyright 2018 Tetrate, Inc. All rights reserved.
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

package mem

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"istio.io/istio/pilot/pkg/serviceregistry/kube"
	"k8s.io/api/core/v1"

	"github.com/tetratelabs/mcc/pkg/datamodel"
)

type (
	// infra is an implementation of datamodel.infra which stores the set of clusters in memory.
	infra map[string]string

	// DataModel is an implementation of datamodel.DataModel which stores the set of services in memory.
	DataModel struct {
		m    sync.RWMutex
		svcs map[string]*datamodel.GlobalService
	}
)

var (
	ErrNotFound = errors.New("service not found")

	_ datamodel.Infrastructure = infra{}
	_ datamodel.DataModel      = &DataModel{}
)

func Infrastructure(clusters map[string]string) datamodel.Infrastructure {
	return infra(clusters)
}

func (i infra) GetIngressGatewayAddress(clusterName string) (string, error) {
	v, found := i[clusterName]
	if !found {
		return "", ErrNotFound
	}
	return v, nil
}

func NewDataModel() *DataModel {
	return &DataModel{sync.RWMutex{}, make(map[string]*datamodel.GlobalService)}
}

func (d *DataModel) CreateGlobalService(g *datamodel.GlobalService) error {
	return d.UpdateGlobalService(g)
}

func (d *DataModel) GetGlobalService(name string) (*datamodel.GlobalService, error) {
	d.m.RLock()
	v, found := d.svcs[name]
	d.m.RUnlock()

	if !found {
		return nil, ErrNotFound
	}
	return v, nil
}

func (d *DataModel) UpdateGlobalService(g *datamodel.GlobalService) error {
	d.m.Lock()
	d.svcs[g.Name] = g
	d.m.Unlock()
	return nil
}

func (d *DataModel) DeleteGlobalService(name string) (*datamodel.GlobalService, error) {
	d.m.Lock()
	defer d.m.Unlock()

	v, found := d.svcs[name]
	if !found {
		return nil, ErrNotFound
	}
	delete(d.svcs, name)
	return v, nil
}

func (d *DataModel) ListGlobalServices() map[string]*datamodel.GlobalService {
	// take a racy length read to avoid alloc while holding lock
	out := make(map[string]*datamodel.GlobalService, len(d.svcs))
	d.m.RLock()
	for k, v := range d.svcs {
		out[k] = v
	}
	d.m.RUnlock()
	return out
}

func (d *DataModel) Handler(cluster string) sdk.Handler {
	return perClusterWatcher{
		name: cluster,
		dm:   d,
	}
}

type perClusterWatcher struct {
	name string // name of the cluster we're watching
	dm   datamodel.DataModel
}

func (p perClusterWatcher) Handle(ctx context.Context, event sdk.Event) error {
	switch cr := event.Object.(type) {
	case *v1.Service:
		gs := p.GetOrCreateGlobalService(cr)
		if event.Deleted {
			_, err := p.dm.DeleteGlobalService(gs.Name)
			return err
		}
		return p.dm.UpdateGlobalService(gs)
	}
	return nil
}

func (p perClusterWatcher) GetOrCreateGlobalService(s *v1.Service) *datamodel.GlobalService {
	gs, err := p.dm.GetGlobalService(s.Name)
	if err == ErrNotFound {
		ports := make([]datamodel.Port, 0, len(s.Spec.Ports))
		for _, p := range s.Spec.Ports {
			protocol := kube.ConvertProtocol(p.Name, p.Protocol)
			ports = append(ports, datamodel.Port{
				ServicePort: uint32(p.Port),
				Protocol:    string(protocol),
				BackendPort: uint32(p.TargetPort.IntVal),
				Name:        p.Name,
			})
		}

		svc := &datamodel.GlobalService{
			Name: s.Name,
			DNSPrefixes: []string{
				s.Name,
				s.Name + "." + s.Namespace,
			},
			Ports:        ports,
			Backends:     map[string]string{p.name: serviceName(s)},
			Unregistered: false,
		}

		if s.Spec.Type == v1.ServiceTypeClusterIP {
			svc.Address = net.ParseIP(s.Spec.ClusterIP)
		} else if s.Spec.Type == v1.ServiceTypeLoadBalancer {
			svc.Address = net.ParseIP(s.Spec.LoadBalancerIP)
		}
		return svc
	}

	// service with the same name already exists; merge them. We assume ports and DNS prefixes already match.
	// TODO: do we need to do more checking to ensure the services really match (e.g. not assume DNS, ports match)?
	if _, exists := gs.Backends[p.name]; !exists {
		gs.Backends[p.name] = serviceName(s)
	}
	return gs
}

func serviceName(s *v1.Service) string {
	if s.ClusterName != "" {
		return fmt.Sprintf("%s.%s.%s", s.Name, s.Namespace, s.ClusterName)
	} else {
		return fmt.Sprintf("%s.%s.svc.cluster.local", s.Name, s.Namespace)
	}
}
