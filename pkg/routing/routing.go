package routing

// Copyright (c) Tetrate, Inc 2018. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/tetratelabs/mcc/pkg/datamodel"

	"github.com/ghodss/yaml"
	multierror "github.com/hashicorp/go-multierror"
	istioapi "istio.io/api/networking/v1alpha3"
	istiocrd "istio.io/istio/pilot/pkg/config/kube/crd"
	istioconfig "istio.io/istio/pilot/pkg/model"
)

// IstioConfigDescriptor describes an Istio networking configuration object
// along with other metadata useful for processing
type IstioConfigDescriptor struct {
	// Name of the resource
	Name string
	// Hosts associated with the resource (applies to Gateway, DestinationRules, VirtualServices)
	Hosts []string
	// Config is the configuration associated with the object (proto, plus config meta)
	Config *istioconfig.Config
	// Yaml is the CRD in yaml form
	Yaml []byte
	// Cluster where this config needs to be present
	Cluster string
}

// GenerateConfigs generates configuration for every cluster, service pair in the DataModel.
// It returns a map of (service name -> (cluster name -> configs))
func GenerateConfigs(dm datamodel.DataModel, infra datamodel.Infrastructure, clusters []string) ([]string, map[string]map[string][]*IstioConfigDescriptor, error) {
	var errs error
	svcs := dm.ListGlobalServices()
	names := make([]string, 0, len(svcs))
	out := make(map[string]map[string][]*IstioConfigDescriptor, len(svcs))
	for name, svc := range svcs {
		cfgs, err := BuildGlobalServiceConfigs(svc, clusters, infra)
		if err != nil {
			errs = multierror.Append(errs, errors.Wrap(err, "could not construct configs"))
			continue
		}
		names = append(names, name)
		out[name] = cfgs
	}
	sort.Strings(names)
	return names, out, errs
}

// DefaultDomainSuffix is the shared DNS suffix for all such global services.
// TODO: make this configurable
const DefaultDomainSuffix = "global"

func BuildGlobalServiceConfigs(globalService *datamodel.GlobalService, clusters []string, infrastructure datamodel.Infrastructure) (map[string][]*IstioConfigDescriptor, error) {
	// In an attempt to handle updates, we try to remove the service
	// generate ingress gateway per cluster
	gateways, err := buildIstioGatewayForGlobalService(globalService)
	if err != nil {
		return nil, err
	}

	// Generate a virtual service for each backend cluster/service
	virtualServices, err := buildVirtualServiceForGlobalService(globalService, gateways)
	if err != nil {
		return nil, err
	}

	serviceEntry, err := buildServiceEntryForGlobalService(globalService, infrastructure)
	if err != nil {
		return nil, err
	}

	configsToApply := make(map[string][]*IstioConfigDescriptor)
	// Since we return error above, at this stage, gateways and virtualservices will have same set of clusters
	for cluster := range globalService.Backends {
		configsToApply[cluster] = []*IstioConfigDescriptor{gateways[cluster], virtualServices[cluster]}
	}

	for _, c := range clusters {
		configsToApply[c] = append(configsToApply[c], serviceEntry)
	}

	return configsToApply, nil
}

func removeGlobalService(globalService *datamodel.GlobalService, clusters []string) (map[string][]*IstioConfigDescriptor, error) {

	gateways, err := removeIstioGatewayForGlobalService(globalService)
	if err != nil {
		return nil, err
	}

	virtualServices, err := removeVirtualServiceForGlobalService(globalService)
	if err != nil {
		return nil, err
	}

	serviceEntry, err := removeServiceEntryForGlobalService(globalService)
	if err != nil {
		return nil, err
	}

	configsToDelete := make(map[string][]*IstioConfigDescriptor)
	// Since we return error above, at this stage, gateways and virtualservices will have same set of clusters
	for cluster := range globalService.Backends {
		configsToDelete[cluster] = []*IstioConfigDescriptor{gateways[cluster], virtualServices[cluster]}
	}

	for _, c := range clusters {
		configsToDelete[c] = append(configsToDelete[c], serviceEntry)
	}

	return configsToDelete, nil
}

func buildIstioGatewayForGlobalService(globalService *datamodel.GlobalService) (map[string]*IstioConfigDescriptor, error) {
	// 1. generate ingress gateway
	gateway := &istioapi.Gateway{
		Servers:  make([]*istioapi.Server, 0), // We need a server for each port in global service
		Selector: map[string]string{"istio": "ingressgateway"},
	}

	hosts := make([]string, 0)
	for _, dnsPrefix := range globalService.DNSPrefixes {
		hosts = append(hosts, fmt.Sprintf("%s.%s", dnsPrefix, DefaultDomainSuffix))
	}
	gatewayName := fmt.Sprintf("cw-%s-gateway", globalService.Name)

	for _, p := range globalService.Ports {
		server := &istioapi.Server{
			Port: &istioapi.Port{
				Number:   p.ServicePort,
				Protocol: p.Protocol,
				Name:     p.Name,
			},
			Hosts: hosts,
			// TODO TLS
		}
		gateway.Servers = append(gateway.Servers, server)
	}

	crd := &istioconfig.Config{
		ConfigMeta: istioconfig.ConfigMeta{
			Type:      istioconfig.Gateway.Type,
			Group:     istioconfig.Gateway.Group,
			Version:   istioconfig.Gateway.Version,
			Name:      gatewayName,
			Namespace: "cw",
			Domain:    "svc.cluster.local",
		},
		Spec: gateway,
	}

	yaml, err := protoConfigToYAML(istioconfig.Gateway, crd)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*IstioConfigDescriptor)
	for cluster := range globalService.Backends {
		out[cluster] = &IstioConfigDescriptor{
			Name:    gatewayName,
			Hosts:   hosts,
			Config:  crd,
			Yaml:    yaml,
			Cluster: cluster,
		}
	}

	return out, nil
}

// Generate a virtual service for each backend cluster/service
func buildVirtualServiceForGlobalService(globalService *datamodel.GlobalService,
	gateways map[string]*IstioConfigDescriptor) (map[string]*IstioConfigDescriptor, error) {

	out := make(map[string]*IstioConfigDescriptor)
	var errs error

	for cluster, backendHost := range globalService.Backends {
		virtualService := &istioapi.VirtualService{
			Hosts:    gateways[cluster].Hosts,
			Gateways: []string{gateways[cluster].Name},
			Http:     []*istioapi.HTTPRoute{},
			// TODO: TLS/TCP
		}
		// Generate a HTTP route for all http ports
		for _, p := range globalService.Ports {
			// TODO: Support other protocols once all pieces are in place
			if istioconfig.ParseProtocol(p.Protocol).IsHTTP() {
				httpRoute := &istioapi.HTTPRoute{
					Match: []*istioapi.HTTPMatchRequest{{Port: p.ServicePort}},
					Route: []*istioapi.DestinationWeight{
						{
							Destination: &istioapi.Destination{
								Host: backendHost,
								Port: &istioapi.PortSelector{Port: &istioapi.PortSelector_Number{Number: p.BackendPort}},
							},
							Weight: 100,
						},
					},
				}
				virtualService.Http = append(virtualService.Http, httpRoute)
			}
		}

		virtualServiceCRD := &istioconfig.Config{
			ConfigMeta: istioconfig.ConfigMeta{
				Type:      istioconfig.VirtualService.Type,
				Group:     istioconfig.VirtualService.Group,
				Version:   istioconfig.VirtualService.Version,
				Name:      fmt.Sprintf("cw-%s-virtualservice", globalService.Name),
				Namespace: "cw",
				Domain:    "svc.cluster.local", // TODO: We need to know this from the local cluster
			},
			Spec: virtualService,
		}

		virtualServiceYAML, err := protoConfigToYAML(istioconfig.VirtualService, virtualServiceCRD)
		if err != nil {
			errs = multierror.Append(errs, err)
			// Skip the entire virtual service
			continue
		}

		out[cluster] = &IstioConfigDescriptor{
			Name:    virtualServiceCRD.Name,
			Hosts:   gateways[cluster].Hosts,
			Config:  virtualServiceCRD,
			Yaml:    virtualServiceYAML,
			Cluster: cluster,
		}
	}

	return out, errs
}

func buildServiceEntryForGlobalService(globalService *datamodel.GlobalService, infrastructure datamodel.Infrastructure) (*IstioConfigDescriptor, error) {
	var errs error
	hosts := make([]string, 0)
	for _, dnsPrefix := range globalService.DNSPrefixes {
		hosts = append(hosts, fmt.Sprintf("%s.%s", dnsPrefix, DefaultDomainSuffix))
	}

	serviceEntry := &istioapi.ServiceEntry{
		Hosts:      hosts,
		Location:   istioapi.ServiceEntry_MESH_EXTERNAL,
		Resolution: istioapi.ServiceEntry_DNS,
	}

	if len(globalService.Address) > 0 {
		serviceEntry.Addresses = []string{globalService.Address.String()}
	}

	endpointPortMap := make(map[string]uint32)

	for _, p := range globalService.Ports {
		serviceEntry.Ports = append(serviceEntry.Ports, &istioapi.Port{
			Number:   p.ServicePort,
			Protocol: p.Protocol,
			Name:     p.Name,
		})
		endpointPortMap[p.Name] = p.ServicePort
	}

	backendClusters := make([]string, 0)
	// Add one endpoint for every backend cluster. Traffic will be load balanced across these endpoints
	for cluster := range globalService.Backends {
		backendClusters = append(backendClusters, cluster)
	}
	sort.SliceStable(backendClusters, func(i, j int) bool {
		return backendClusters[i] < backendClusters[j]
	})

	for _, cluster := range backendClusters {
		gatewayAddress, err := infrastructure.GetIngressGatewayAddress(cluster)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		serviceEntry.Endpoints = append(serviceEntry.Endpoints, &istioapi.ServiceEntry_Endpoint{
			Address: gatewayAddress,
			Ports:   endpointPortMap,
			Labels:  map[string]string{"cluster": cluster},
		})
	}

	// Return error if there are no endpoints for the service entry
	if len(serviceEntry.Endpoints) == 0 {
		return nil, errs
	}

	serviceEntryCRD := &istioconfig.Config{
		ConfigMeta: istioconfig.ConfigMeta{
			Type:      istioconfig.ServiceEntry.Type,
			Group:     istioconfig.ServiceEntry.Group,
			Version:   istioconfig.ServiceEntry.Version,
			Name:      fmt.Sprintf("cw-%s-serviceentry", globalService.Name),
			Namespace: "cw",
			Domain:    "svc.cluster.local", // TODO: We need to know this from the local cluster
		},
		Spec: serviceEntry,
	}

	serviceEntryYAML, err := protoConfigToYAML(istioconfig.ServiceEntry, serviceEntryCRD)
	if err != nil {
		errs = multierror.Append(errs, err)
		// Skip the entire service entry
		return nil, errs
	}

	return &IstioConfigDescriptor{
		Name:    serviceEntryCRD.Name,
		Hosts:   hosts,
		Config:  serviceEntryCRD,
		Yaml:    serviceEntryYAML,
		Cluster: "",
	}, errs
}

func removeIstioGatewayForGlobalService(globalService *datamodel.GlobalService) (map[string]*IstioConfigDescriptor, error) {
	gatewayName := fmt.Sprintf("cw-%s-gateway", globalService.Name)

	crd := &istioconfig.Config{
		ConfigMeta: istioconfig.ConfigMeta{
			Type:      istioconfig.Gateway.Type,
			Group:     istioconfig.Gateway.Group,
			Version:   istioconfig.Gateway.Version,
			Name:      gatewayName,
			Namespace: "cw",
			Domain:    "svc.cluster.local",
		},
		Spec: &istioapi.Gateway{},
	}

	yaml, err := protoConfigToYAML(istioconfig.Gateway, crd)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*IstioConfigDescriptor)
	for cluster := range globalService.Backends {
		out[cluster] = &IstioConfigDescriptor{
			Name:    gatewayName,
			Config:  crd,
			Yaml:    yaml,
			Cluster: cluster,
		}
	}

	return out, nil
}

// Generate a virtual service for each backend cluster/service
func removeVirtualServiceForGlobalService(globalService *datamodel.GlobalService) (map[string]*IstioConfigDescriptor, error) {
	out := make(map[string]*IstioConfigDescriptor)
	var errs error

	for cluster := range globalService.Backends {
		virtualServiceCRD := &istioconfig.Config{
			ConfigMeta: istioconfig.ConfigMeta{
				Type:      istioconfig.VirtualService.Type,
				Group:     istioconfig.VirtualService.Group,
				Version:   istioconfig.VirtualService.Version,
				Name:      fmt.Sprintf("cw-%s-virtualservice", globalService.Name),
				Namespace: "cw",
				Domain:    "svc.cluster.local", // TODO: We need to know this from the local cluster
			},
			Spec: &istioapi.VirtualService{},
		}

		virtualServiceYAML, err := protoConfigToYAML(istioconfig.VirtualService, virtualServiceCRD)
		if err != nil {
			errs = multierror.Append(errs, err)
			// Skip the entire virtual service
			continue
		}

		out[cluster] = &IstioConfigDescriptor{
			Name:    virtualServiceCRD.Name,
			Config:  virtualServiceCRD,
			Yaml:    virtualServiceYAML,
			Cluster: cluster,
		}
	}

	return out, errs
}

func removeServiceEntryForGlobalService(globalService *datamodel.GlobalService) (*IstioConfigDescriptor, error) {
	serviceEntryCRD := &istioconfig.Config{
		ConfigMeta: istioconfig.ConfigMeta{
			Type:      istioconfig.ServiceEntry.Type,
			Group:     istioconfig.ServiceEntry.Group,
			Version:   istioconfig.ServiceEntry.Version,
			Name:      fmt.Sprintf("cw-%s-serviceentry", globalService.Name),
			Namespace: "cw",
			Domain:    "svc.cluster.local", // TODO: We need to know this from the local cluster
		},
		Spec: &istioapi.ServiceEntry{},
	}

	serviceEntryYAML, err := protoConfigToYAML(istioconfig.ServiceEntry, serviceEntryCRD)
	if err != nil {
		return nil, err
	}

	return &IstioConfigDescriptor{
		Name:    serviceEntryCRD.Name,
		Config:  serviceEntryCRD,
		Yaml:    serviceEntryYAML,
		Cluster: "",
	}, nil
}

func protoConfigToYAML(schema istioconfig.ProtoSchema, istioConfigObject *istioconfig.Config) ([]byte, error) {
	kubeObject, err := istiocrd.ConvertConfig(schema, *istioConfigObject)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert Istio %s object to K8S CRD: %s", schema.Type, err)
	}

	yamlPayload, err := yaml.Marshal(kubeObject)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert Istio %s kube CRD to YAML: %s", schema.Type, err)
	}
	return yamlPayload, nil
}

func concatenateYAMLs(objs []*IstioConfigDescriptor) []byte {
	var concat bytes.Buffer
	for _, obj := range objs {
		concat.Write(obj.Yaml)
		concat.WriteString("---\n")
	}
	return concat.Bytes()
}
