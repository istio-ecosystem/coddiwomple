package mem

import (
	"errors"

	"github.com/tetratelabs/mcc/pkg/datamodel"
)

type (
	// infra is an implementation of datamodel.infra which stores the set of clusters in memory.
	infra map[string]string

	// dm is an implementation of datamodel.dm which stores the set of services in memory.
	dm map[string]*datamodel.GlobalService
)

var (
	ErrNotFound = errors.New("service not found")

	_ datamodel.Infrastructure = infra{}
	_ datamodel.DataModel      = dm{}
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

func DataModel() datamodel.DataModel {
	return dm(make(map[string]*datamodel.GlobalService))
}

func (d dm) CreateGlobalService(g *datamodel.GlobalService) error {
	d[g.Name] = g
	return nil
}

func (d dm) GetGlobalService(name string) (*datamodel.GlobalService, error) {
	v, found := d[name]
	if !found {
		return nil, ErrNotFound
	}
	return v, nil
}

func (d dm) UpdateGlobalService(g *datamodel.GlobalService) error {
	d[g.Name] = g
	return nil
}

func (d dm) DeleteGlobalService(name string) (*datamodel.GlobalService, error) {
	v, found := d[name]
	if !found {
		return nil, ErrNotFound
	}
	delete(d, name)
	return v, nil
}

func (d dm) ListGlobalServices() map[string]*datamodel.GlobalService {
	return d
}
