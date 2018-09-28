package ui

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
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"log"

	"github.com/tetratelabs/mcc/pkg/datamodel"
	"github.com/tetratelabs/mcc/pkg/datamodel/mem"
	"github.com/tetratelabs/mcc/pkg/routing"
)

func RegisterHandlers(dm datamodel.DataModel, infra datamodel.Infrastructure, clusterNames []string, mux *http.ServeMux) {
	h := handler{dm, infra, clusterNames}

	mux.HandleFunc("/", h.serveServiceList)
	// returns array of configs, each is the content of a <pre> block
	mux.HandleFunc("/getconfig", h.genConfig)

}

type handler struct {
	dm       datamodel.DataModel
	infra    datamodel.Infrastructure
	clusters []string
}

func (h handler) serveServiceList(w http.ResponseWriter, req *http.Request) {
	gss := h.dm.ListGlobalServices()
	svcs := make([]svc, 0, len(gss))
	for svcName, gs := range gss {
		localClusterMap := make(map[string]bool, len(gs.Backends))
		for name := range gs.Backends {
			localClusterMap[name] = true
		}
		svcs = append(svcs, svc{
			Name:     svcName,
			Clusters: localClusterMap,
		})
	}

	// map iteration order is random, keep things in sorted order for consistency
	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i].Name < svcs[j].Name
	})

	err := tmpl.Execute(w, map[string]interface{}{
		"ClusterNames": h.clusters,
		"Services":     svcs,
	})
	if err != nil {
		fmt.Fprintf(w, "failed with %v", err)
	}
}

type svc struct {
	Name     string
	Clusters map[string]bool
}

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},
}).Parse(`{{ $clusterNames := .ClusterNames }}<html lang="en">
	<head>
		<title>Coddiwomple UI</title>
		<meta name="description" content="Your services listed by cluster">
		<script>
			window.onload = function() {
				// URL is the URL to POST data to;
				function XHR(url, data, callback) {
					var req = new XMLHttpRequest();
					req.onreadystatechange = function(data) {
						if (req.readyState == req.DONE) {
							if (req.status != 200) {
								console.log("req to ", url, " failed with status code ", req.status);
							}
							callback(req.responseText);
						}
					}
					console.log("POST'ing data %s", data)
					req.open("POST", url);
					req.send(JSON.stringify(data));
				}

				function attachEventsToLinks() {
					document.querySelectorAll('a').forEach(function(element) {
						element.addEventListener("click", function() {
							name = element.getAttribute("data-service-name");
							XHR("/getconfig", name, function(raw) {
								data = JSON.parse(raw)
								console.log("callback called on element: %s", name)
								console.log(data)
								var row = document.getElementById(name+"-config");
								row.innerHTML = '';
								row.insertCell(0); // we need an empty cell at the beginning and end of the table
								data.forEach(function(contents) {
									cell = row.insertCell(-1);
									pre = document.createElement("pre");
									pre.innerHTML = contents;
									cell.appendChild(pre);
								})
								row.insertCell(-1);
							})
						})
					});
				}
				attachEventsToLinks();
			}
		</script>
	</head>
	<body>
		<table>
			<tr>
				<th rowspan="2">Service</th>
				<th colspan="{{ len .ClusterNames }}">Clusters</th>
				<th rowspan="2">Generate</th>
			</tr>
			<tr>{{ range .ClusterNames }}<th>{{.}}</th>{{ end }}</tr>
			</tr>{{ range $s := .Services }}
			<tr>
				<td>{{ .Name }}</td>{{ range $name := $clusterNames }}
				{{ if (index $s.Clusters $name) }}<td>X</td>{{ else }}<td />{{ end }}{{ end }}
				<td><a data-service-name="{{ .Name }}" href="#">Generate Config</a></td>
			</tr>
			<tr id="{{ .Name }}-config"></tr>{{end}}
		</table>
	</body>
</html>
`))

func (h handler) genConfig(w http.ResponseWriter, req *http.Request) {
	svcBytes, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()

	svcKey := strings.Trim(string(svcBytes), `"`)
	log.Printf("called with payload %q\n", svcKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to read req body with: %v", err)
		return
	}

	parts := strings.Split(svcKey, ".")
	if len(parts) < 1 {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to extract service name and namespace from service: %q", svcKey)
		return
	}
	name := parts[0]

	svc, err := h.dm.GetGlobalService(name)
	if err == mem.ErrNotFound {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("service %s not found\n", svcKey)
		fmt.Fprintf(w, "service %s not found", svcKey)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("error retrieving service %s: %v\n", svcKey, err)
		fmt.Fprintf(w, "service %q not found", svcKey)
		return
	}

	perClusterConfig, err := routing.BuildGlobalServiceConfigs(svc, h.clusters, h.infra)

	inOrderOutput := make([]string, len(h.clusters))
	for i, name := range h.clusters {
		out := &bytes.Buffer{}
		for _, cfg := range perClusterConfig[name] {
			fmt.Fprintf(out, "%s---\n", string(cfg.Yaml))
		}
		inOrderOutput[i] = out.String()
	}

	payload, err := json.Marshal(inOrderOutput)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed marshal text into JSON with: %v", err)
		fmt.Fprintf(w, "failed to marshal text into JSON: %v", err)
		return
	}
	if _, err := w.Write(payload); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to write out JSON to response: %v", err)
		fmt.Fprintf(w, "failed to write out JSON to response: %v", err)
		return
	}
}
