package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/tetratelabs/mcc/pkg/datamodel"
	"github.com/tetratelabs/mcc/pkg/datamodel/mem"
	"github.com/tetratelabs/mcc/pkg/routing"
)

func configGenCmd() *cobra.Command {

	var (
		cluster string
		service string
		//clusters     []string
		clustersFile string
		servicesFile string
	)

	cmd := &cobra.Command{
		Use:     "gen",
		Short:   "Generates Istio config for each cluster for the target service.",
		Example: "mcc gen ",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			var infra datamodel.Infrastructure
			var clusters []string
			//if clustersFile != "" {
			clusters, infra, err = clustersFromFile(clustersFile)
			//} else {
			//	clusters, infra, err = clustersFlagToInfra(clusters)
			//}
			if err != nil {
				return errors.Wrap(err, "invalid clusters:")
			}

			var dm datamodel.DataModel
			dm, err = serviceFromFile(servicesFile)
			if err != nil {
				return errors.Wrapf(err, "could not read services from %q", servicesFile)
			}

			svcs, cfgs, err := routing.GenerateConfigs(dm, infra, clusters)
			if err != nil {
				return errors.Wrap(err, "could not construct config from clusters and services")
			}

			// TODO: flag for output to file, etc.
			out := os.Stdout

			for _, svc := range svcs {
				if service != "" && svc != service {
					continue
				}

				fmt.Fprintf(out, "################################################################################\n")
				fmt.Fprintf(out, "# Configs for Service %q\n", svc)
				fmt.Fprintf(out, "################################################################################\n")
				for cl, cfg := range cfgs[svc] {
					// filter output by --cluster flag
					if cluster != "" && cl != cluster {
						continue
					}

					fmt.Fprintf(out, "####################\n")
					fmt.Fprintf(out, "# Configs for Cluster %q\n", cl)
					fmt.Fprintf(out, "####################\n")
					for _, c := range cfg {
						fmt.Fprint(out, "---\n")
						fmt.Fprint(out, string(c.Yaml))
						fmt.Fprint(out, "\n")
					}
				}
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&cluster, "cluster", "",
		"Print configuration only for the provided cluster; must match a cluster name in the cluster-file. "+
			"E.g. `kubectl apply -f <(mcc gen --cluster cluster-name) --context cluster-name`")
	cmd.PersistentFlags().StringVar(&service, "service", "",
		"Print configuration only for the provided service; must match a service name in the service-file. "+
			"E.g. `kubectl apply -f <(mcc gen --service foo --cluster cluster-name) --context cluster-name`")
	//cmd.PersistentFlags().StringSliceVarP(&clusters, "clusters", "c", []string{},
	//	"comma separated list of name:address pairs where the address is a DNS name. // TODO support IPs")
	cmd.PersistentFlags().StringVar(&clustersFile, "cluster-file", "./clusters.json",
		`Path to a file with a JSON array of clusters, where a cluster is an object like '{"name": "ClusterName", "address": "dns.address.of.cluster"}'`)

	cmd.PersistentFlags().StringVar(&servicesFile, "service-file", "./services.json",
		`Path to a file with a JSON array of GlobalServices, see datamodel.GlobalService for the JSON schema.`)

	return cmd
}

type services []datamodel.GlobalService

func serviceFromFile(path string) (datamodel.DataModel, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open file %q", path)
	}
	gss := services{}
	if err := json.Unmarshal(contents, &gss); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal file as json")
	}

	dm := mem.DataModel()
	for _, gs := range gss {
		svc := gs
		dm.CreateGlobalService(&svc)
	}
	return dm, nil
}

type clusters []cluster
type cluster struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func clustersFromFile(path string) ([]string, datamodel.Infrastructure, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return []string{}, nil, errors.Wrapf(err, "could not open file %q", path)
	}
	c := clusters{}
	if err := json.Unmarshal(contents, &c); err != nil {
		return []string{}, nil, errors.Wrap(err, "could not unmarshal file as json")
	}

	names := make([]string, len(c))
	cls := make(map[string]string, len(c))
	for i, cl := range c {
		names[i] = cl.Name
		cls[cl.Name] = cl.Address
	}
	sort.Strings(names)
	return names, mem.Infrastructure(cls), nil
}

func clustersFlagToInfra(clusters []string) ([]string, datamodel.Infrastructure, error) {
	cls := make(map[string]string, len(clusters))
	names := make([]string, 0, len(clusters))
	var errs error
	for i, c := range clusters {
		parts := strings.Split(c, ":")
		if len(parts) != 2 {
			errs = multierror.Append(errs, fmt.Errorf("expected `name:address` pairs but got %q", c))
			continue
		}
		cls[parts[0]] = parts[1]
		names[i] = parts[0]
	}
	sort.Strings(names)
	return names, mem.Infrastructure(cls), errs
}
