package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/operator-framework/operator-sdk/pkg/sdk/metrics"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/tetratelabs/mcc/pkg/service"
	"github.com/tetratelabs/mcc/pkg/ui"
)

const (
	resourcePluralName = "v1/services"
	allNamespaces      = ""
	resyncPeriod       = 5
)

var (
	groupVersionKind = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}
)

func uiCmd() (serve *cobra.Command) {
	var (
		port        int
		kubeConfigs []string
	)

	serve = &cobra.Command{
		Use:     "ui",
		Short:   "Starts the mcc UI on localhost",
		Example: "mcc ui --port 123",
		RunE: func(cmd *cobra.Command, args []string) error {
			collector := metrics.New()
			metrics.RegisterCollector(collector)

			multiStore := service.NewMultiClusterStore()
			for kubeConfig, contexts := range processFlags(kubeConfigs) {
				for _, kubeCtx := range contexts {
					client, err := k8sClientFor(kubeConfig, kubeCtx)
					if err != nil {
						return fmt.Errorf("failed to construct k8s client %q with context %q: %v", kubeConfig, kubeCtx, err)
					}
					log.Printf("Watching for %q across all namespaces in cluster %q with resync period %d", resourcePluralName, kubeCtx, resyncPeriod)
					i := sdk.NewInformerWithHandler(resourcePluralName, allNamespaces, client, resyncPeriod, collector, multiStore.NewCluster(kubeCtx))
					go i.Run(context.Background())
				}
			}

			mux := http.NewServeMux()
			ui.RegisterHandlers(multiStore, mux)
			address := fmt.Sprintf(":%d", port)
			log.Printf("starting server on %s", address)
			return http.ListenAndServe(address, mux)
		},
	}

	serve.PersistentFlags().IntVar(&port, "port", 8080, "the port to start the local UI server on")
	serve.PersistentFlags().StringArrayVar(&kubeConfigs,
		"kube-config", nil,
		`kubeconfig location in the form "filepath:contextNameOne,contextNameTwo" where contextNameOne is the name passed to "kubectl --context=contextNameOne". If no contexts are provided the tool will use the default context. This flag can be repeated to set multiple kubeconfigs with multiple contexts each.`)
	return serve
}

func processFlags(kubeconfigFlag []string) map[string][]string {
	// no values provided, so we use the default
	if len(kubeconfigFlag) == 0 {
		return map[string][]string{"~/.kube/config": {""}}
	}

	out := make(map[string][]string, len(kubeconfigFlag))
	for _, cfg := range kubeconfigFlag {
		// two cases:
		// 1) --kube-config=path/to/config
		// 2) --kube-config=path/to/config:contextOne,contextTwo
		parts := strings.Split(cfg, ":")
		contexts := []string{""}
		if len(parts) > 1 {
			contexts = strings.Split(parts[1], ",")
		}
		out[parts[0]] = contexts
	}
	return out
}

func k8sClientFor(path, context string) (dynamic.ResourceInterface, error) {
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: path},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create config with: %v", err)
	}

	k, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client with: %v", err)
	}

	cachedDiscoveryClient := cached.NewMemCacheClient(k.Discovery())
	restMapper := discovery.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient, meta.InterfacesForUnstructured)
	restMapper.Reset()
	config.ContentConfig = dynamic.ContentConfig()
	clientPool := dynamic.NewClientPool(config, restMapper, dynamic.LegacyAPIPathResolverFunc)

	mapping, err := restMapper.RESTMapping(groupVersionKind.GroupKind(), groupVersionKind.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get REST mapping for service with: %v", err)
	}
	client, err := clientPool.ClientForGroupVersionKind(groupVersionKind)
	if err != nil {
		return nil, err
	}

	return client.Resource(&metav1.APIResource{
		Name:       mapping.Resource,
		Namespaced: mapping.Scope == meta.RESTScopeNamespace,
		Kind:       "Service",
	}, allNamespaces), nil
}
