package datamodel

import "net"

//go:generate mockgen -source=model.go -destination=mock/mock_datamodel.go

// DataModel is the standard interface that all concrete DataModel types will adhere to.
// Objects can be stored in any datastore (in mem, etcd, rdbms, etc.)
type DataModel interface {
	CreateGlobalService(g *GlobalService) error
	GetGlobalService(name string) *GlobalService
	UpdateGlobalService(g *GlobalService) error
	DeleteGlobalService(name string) (*GlobalService, error)
	ListGlobalServices() map[string]*GlobalService
}

// Port describes the properties of a specific port of a service.
type Port struct {
	// ServicePort is a valid non-negative integer port number.
	ServicePort uint32

	// Protocol exposed on the port.
	// MUST BE one of HTTP|HTTPS|GRPC|HTTP2|MONGO|TCP.
	Protocol string

	// BackendPort is the corresponding port exposed by the backend services.
	BackendPort uint32

	// Name associated with the port
	Name string
}

// GlobalService is a service exposed from a cluster. All traffic will
// arrive at the ingress gateway of the cluster.
type GlobalService struct {
	// Name is a globally unique name to refer to this service in other API
	// calls. The same global service can be exposed from multiple clusters
	// in cases where the customer wants a global load balancing across
	// clusters.
	Name string

	// DNSPrefixes for hosts used by the service.  The full DNS name will be
	// constructed based on the pre-configured DNS suffix. For example,
	// foo.ns1 will become foo.ns1.svc.cluster.global if svc.cluster.global
	// is the DNS suffix.
	DNSPrefixes []string

	// Ports exposed by the service.
	Ports []Port

	// Backend services in different clusters
	Backends map[string]string

	// Address is the VIP assigned to this service
	Address net.IP

	// Unregistered is set by the server to indicate that
	// the service will be removed in the future after cleaning up
	// the associated configurations from the respective clusters
	Unregistered bool
}

// Infrastructure abstracts the system that has information about
// the actual location of the gateways, their addresses, handles to
// the underlying clusters connected to this manager, etc.
type Infrastructure interface {
	// GetIngressGatewayAddress returns the address of the ingress gateway
	// of a cluster, that is accessible from other clusters.
	GetIngressGatewayAddress(clusterName string) (string, error)
}
