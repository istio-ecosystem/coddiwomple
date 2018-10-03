# cw
Coddiwomple, a Multi-Cluster-Config Generator

## Installation
```bash
$ go get -u github.com/istio-ecosystem/cw
$ cd $GOPATH/github.com/istio-ecosystem/cw
$ dep ensure
$ make
```

## Quick Start - UI Mode
The easiest way to start using `cw` is through the built-in UI.

```bash
cw ui --cluster-file ./clusters.json 
```

Where `clusters.json` must is a JSON array of clusters, where each cluster is `{"name": string, "address": string, "kubeconfig_path": string, "kubeconfig_context": string}`.
The address must be the DNS name or IP address of the istio-ingressgateway.
The given context in the kubeconfig file must have credentials to connect to the cluster; we list the Kubernetes `Services` which are running.

For example:
```json
[
  {
    "name": "a",
    "address": "a.com",
    "kubeconfig_path": "./kubeconfig-a",
    "kubecondig_context": "default"
  },
  {
    "name": "b",
    "address": "b.com",
    "kubeconfig_path": "./kubeconfig-b",
    "kubecondig_context": "default"
  },
  {
    "name": "c",
    "address": "c.com",
    "kubeconfig_path": "./kubeconfig-c",
    "kubecondig_context": "default"
  }
]
```

## CLI Mode
Coddiwomple has a CLI mode which is designed to be called from scripts and gives more control over the input and generated output.

```bash
cw gen --cluster-file ./clusters.json --service-file ./services.json
```

Where `clusters.json` must is a JSON array of clusters, as above.

In CLI mode, no connection is made to the clusters on your behalf, so Services must be explicitly listed as well.
This allows you to list only the Services for which you'd like Istio multi-mesh config generated.

Thus, `service.json` is a JSON array of service objects where each service is:
```json
[
  {
    "name": string,
    "dns_prefixes": string[], // names by which the service can be addressed. Coddiwomple will emit configuration which will also make <prefix>.global availabe
    "ports": {
      "name": string, // Name associated with the port
      "protocol": string, // MUST BE one of HTTP|HTTPS|GRPC|HTTP2|MONGO|TCP.
      "service_port": uint32, // This is the port clients call in to, i.e. the port to intercept in the mesh, and to open at the gateway
      "backend_port": uint32, // The port exposed by the backend (k8s) Service.
    },
    // Each key should be the name of a cluster in the cluster-file.
    // Each value is the (fully qualified) name of the service in that cluster.
    "backends": {[key: string]: string},
    "address": string, // [optional, rarely used] hard-coded IP address of the service for TCP services
  }
]
```

For exmaple:

```json
[
    {
        "name": "foo",
        "dns_prefixes": [
            "foo",
            "foo.default"
        ],
        "ports": [
            {
                "name": "http",
                "service_port": 80,
                "protocol": "HTTP",
                "backend_port": 80
            }
        ],
        "backends": {
            "a": "foo.default.svc.cluster.local"
        }
    },
    {
        "name": "bar",
        "dns_prefixes": [
            "bar",
            "bar.default"
        ],
        "ports": [
            {
                "name": "http",
                "service_port": 80,
                "protocol": "HTTP",
                "backend_port": 80
            }
        ],
        "backends": {
            "b": "bar.default.svc.cluster.local"
        }
    },
    {
        "name": "car",
        "dns_prefixes": [
            "car",
            "car.default"
        ],
        "ports": [
            {
                "name": "tcp",
                "service_port": 81,
                "protocol": "TCP",
                "backend_port": 81
            }
        ],
        "backends": {
            "c": "car.default.svc.cluster.local"
        },
        "address": "1.2.3.4"
    }
]
```

## Example - Split Bookinfo
The `testdata/split-bookinfo` directory contains the Kubernetes YAMLs for Bookinfo, split in two to distribute the services across two clusters, A, and B thus:

* Cluster A
  * `productpage` and ingress `Gateway`, `VirtualService`
  * `ratings`
  * `details`
* Cluster B
  * `reviews`
  * `details`

Ergo there is a call-chain from `productpage` (cluster A) -> `reviews` (cluster B) -> `ratings` (cluster A).

The directory also contains an example `services.json` detailing Bookinfo; only the cluster names need be changed.

Note: In order for Bookinfo to use Coddiwomple's Istio configuration to call across meshes, the Bookinfo code needs to be modified to use names of the form `details.global`.
Thus, these YAMLs deploy custom Bookinfo images with this change made.
The source for this modified Bookinfo hasn't been published yet, but is trivial to recreate.
