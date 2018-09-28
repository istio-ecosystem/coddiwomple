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

package cmd

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/tetratelabs/mcc/pkg/datamodel"
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
		Example: "cw gen ",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			var infra datamodel.Infrastructure
			var clusters []string
			//if clustersFile != "" {
			clusters, _, infra, err = clustersFromFile(clustersFile)
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
			"E.g. `kubectl apply -f <(cw gen --cluster cluster-name) --context cluster-name`")
	cmd.PersistentFlags().StringVar(&service, "service", "",
		"Print configuration only for the provided service; must match a service name in the service-file. "+
			"E.g. `kubectl apply -f <(cw gen --service foo --cluster cluster-name) --context cluster-name`")
	//cmd.PersistentFlags().StringSliceVarP(&clusters, "clusters", "c", []string{},
	//	"comma separated list of name:address pairs where the address is a DNS name. // TODO support IPs")
	cmd.PersistentFlags().StringVar(&clustersFile, "cluster-file", "./clusters.json",
		`Path to a file with a JSON array of clusters, where a cluster is an object like '{"name": "ClusterName", "address": "dns.address.of.cluster"}'`)
	cmd.PersistentFlags().StringVar(&servicesFile, "service-file", "./services.json",
		`Path to a file with a JSON array of GlobalServices, see datamodel.GlobalService for the JSON schema.`)

	return cmd
}
