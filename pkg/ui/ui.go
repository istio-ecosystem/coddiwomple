package ui

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"sort"

	"github.com/tetratelabs/mcc/pkg/service"
)

func RegisterHandlers(mcs *service.MultiClusterStore, mux *http.ServeMux) {
	h := handler{mcs}

	mux.HandleFunc("/", h.serveServiceList)
	mux.HandleFunc("/getconfig", h.genConfig)

}

type handler struct {
	mcs *service.MultiClusterStore
}

func (h handler) serveServiceList(w http.ResponseWriter, req *http.Request) {
	snapshot := h.mcs.Names()
	clusters := make([]string, 0, len(snapshot))
	svcMap := make(map[string]svc)
	// To render, we want to list all clusters by service; the MultiClusterStore gives us
	// the services by cluster. So we'll walk over the set collecting the unique cluster names and
	// constructing a set of services with their list of clusters attached.
	for cluster, services := range snapshot {
		if cluster == "" {
			cluster = "default"
		}
		clusters = append(clusters, cluster)
		for s := range services {
			if svc, exists := svcMap[s]; exists {
				svc.Clusters[cluster] = true
				continue
			}
			svcMap[s] = svc{
				Name:     s,
				Clusters: map[string]bool{cluster: true},
			}
		}
	}

	svcs := make([]svc, 0, len(svcMap))
	for _, s := range svcMap {
		svcs = append(svcs, s)
	}

	// map iteration order is random, keep things in sorted order for consistency
	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i].Name < svcs[j].Name
	})
	sort.Strings(clusters)

	err := tmpl.Execute(w, map[string]interface{}{
		"ClusterNames": clusters,
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
		<title>MCC UI</title>
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
							XHR("/getconfig", name, function(data) {
								console.log("callback called on element: %s with data '%s'", name, data)
								var holder = document.getElementById(name+"-config");
								holder.textContent = data;
								holder.style.display = "";
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
			<tr>
				<td colspan="{{ add (len $clusterNames) 2 }}" display="none"><div id="{{ .Name }}-config"></div></td>
			</tr>{{end}}
		</table>
	</body>
</html>
`))

func (h handler) genConfig(w http.ResponseWriter, req *http.Request) {
	in, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	fmt.Printf("called with payload %s\n", in)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("failed to read req body with: %v", err)
	}
	fmt.Fprintf(w, "got request with body: %s", in)
}
