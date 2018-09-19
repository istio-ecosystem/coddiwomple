# mcc
Multi Cluster Config Generation

To install:
```bash
go get github.com/tetratelabs/mcc/cmd/mcc

mcc --service-file ./services.json --cluster-file ./clusters.json 
```

Where `clusters.json` must is a JSON array of clusters, where each cluster is `{"name": string, "address": string}`; the address must be a DNS name.

For example:
```json
[
  {
    "name": "a",
    "address": "a.com"
  },
  {
    "name": "b",
    "address": "b.com"
  },
  {
    "name": "c",
    "address": "c.com"
  }
]
```

`service.json` is a JSON array of service objects where each service is:
```json
[
  {
    "name": string,
    "dns_prefixes": string[],
    "ports": {
      "name": string,
      "protocol": string, // MUST BE one of HTTP|HTTPS|GRPC|HTTP2|MONGO|TCP.
      "service_port": uint32,
      "backend_port": uint32,

    },
    // each key should be the name of a cluster in the cluster-file.
    // Each value is the (fully qualified) name of the service in that cluster.
    "backends": {[key: string]: string},
    "address": string,
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

## TODO

- `mcc create-service-json --kubeconfig-path ./path/to/kube/config --kubeconfig-context cluster-name --out ./services.json`
  - Command that connects to a kube cluster, creates GlobalService configs for all of them, merging them if they already exist in the output file.
- `mcc create-cluster-json --name cluster-name --address cluster.adress.com --out ./clusters.json`
  - Similar to above, but for clusters
- Support IP addresses in cluster address field

  
# Development

To build:
```bash
dep ensure
make
```

to re-build easily
```bash
make clean && make
```

Of course, `go build github.com/tetratelabs/mcc/cmd/mcc` also works. 