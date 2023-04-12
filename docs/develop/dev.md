# Develop

## Quick Start

1. `make build_local_image`

2. `make e2e_init`

3. `make e2e_run`

4. check proscope, browser visits http://nodeIP:4040

### Go Package (Structure) Design

```bash
.
├── api
│   └── v1
│       ├── client
│       │   ├── healthy
│       │   └── http_server_api_client.go
│       ├── openapi.yaml
│       └── server
│           ├── configure_http_server_api.go
│           ├── doc.go
│           ├── embedded_spec.go
│           ├── restapi
│           └── server.go
├── charts
│   ├── Chart.yaml
│   ├── crds
│   │   ├── egressgateway.spidernet.io_egressgatewaypolicies.yaml
│   │   ├── egressgateway.spidernet.io_egressgateways.yaml
│   │   └── egressgateway.spidernet.io_egressnodes.yaml
│   ├── LICENSE
│   ├── README.md
│   ├── templates
│   │   ├── configmap.yaml
│   │   ├── daemonset.yaml
│   │   ├── deployment.yaml
│   │   ├── grafanaDashboard.yaml
│   │   ├── _helpers.tpl
│   │   ├── pdb.yaml
│   │   ├── prometheusrule.yaml
│   │   ├── roleBinding.yaml
│   │   ├── role.yaml
│   │   ├── serviceaccount.yaml
│   │   ├── servicemonitor.yaml
│   │   ├── service.yaml
│   │   └── tls.yaml
│   └── values.yaml
├── cmd
│   ├── agent
│   │   ├── cmd
│   │   │   └── root.go
│   │   └── main.go
│   ├── controller
│   │   ├── cmd
│   │   │   └── root.go
│   │   └── main.go
│   └── nettools
│       ├── client
│       │   └── main.go
│       ├── initFlag.go
│       └── server
│           └── main.go
├── codecov.yml
├── CODEOWNERS
├── docs
│   ├── develop
│   │   └── dev.md
│   ├── mkdocs.yml
│   ├── proposal
│   │   ├── 01-egress-gateway
│   │   │   ├── Agent\ Reconcile\ Flow.png
│   │   │   ├── Controller\ Reconcile\ Flow.png
│   │   │   ├── Egress\ Gateway\ Datapath.png
│   │   │   └── Egress\ Gateway.png
│   │   └── 02-egress-node
│   │       └── EgressNode-zh_CN.md
│   ├── README.md
│   └── usage
│       └── Install.md
├── go.mod
├── go.sum
├── images
│   ├── agent
│   │   └── Dockerfile
│   ├── agent-base
│   │   ├── build-gops.sh
│   │   ├── configure-iptables-wrapper.sh
│   │   ├── Dockerfile
│   │   ├── install-others.sh
│   │   ├── iptables-wrapper
│   │   └── sources.list
│   ├── controller
│   │   └── Dockerfile
│   ├── controller-base
│   │   ├── build-gops.sh
│   │   ├── Dockerfile
│   │   ├── install-others.sh
│   │   └── sources.list
│   ├── nettools
│   │   └── Dockerfile
│   └── nettools-base
│       ├── build-gops.sh
│       ├── configure-iptables-wrapper.sh
│       ├── Dockerfile
│       ├── install-others.sh
│       ├── iptables-wrapper
│       └── sources.list
├── LICENSE
├── Makefile
├── Makefile.defs
├── pkg
│   ├── agent
│   │   ├── agent.go
│   │   ├── metrics
│   │   │   └── metrics.go
│   │   ├── police.go
│   │   ├── police_test.go
│   │   ├── route
│   │   │   └── route.go
│   │   ├── vxlan
│   │   │   ├── parent.go
│   │   │   ├── parent_test.go
│   │   │   ├── vxlan.go
│   │   │   └── vxlan_test.go
│   │   ├── vxlan.go
│   │   └── vxlan_test.go
│   ├── config
│   │   └── config.go
│   ├── controller
│   │   ├── allocator
│   │   │   ├── allocator.go
│   │   │   └── interface.go
│   │   ├── controller.go
│   │   ├── controller_test.go
│   │   ├── egress_gateway.go
│   │   ├── egress_gateway_test.go
│   │   ├── egress_node.go
│   │   ├── metrics
│   │   │   └── metrics.go
│   │   ├── node.go
│   │   └── webhook
│   │       ├── validate.go
│   │       └── validate_test.go
│   ├── ethtool
│   │   ├── ethtool_darwin.go
│   │   └── ethtool_linux.go
│   ├── ipam
│   │   └── ipam.go
│   ├── ipset
│   │   ├── ipset.go
│   │   ├── testing
│   │   │   └── ipset.go
│   │   └── types.go
│   ├── iptables
│   │   ├── actions.go
│   │   ├── binary.go
│   │   ├── cmdshim
│   │   │   └── cmd_shim.go
│   │   ├── interface.go
│   │   ├── lock.go
│   │   ├── match_builder.go
│   │   ├── metrics.go
│   │   ├── restore_buffer.go
│   │   ├── rules.go
│   │   ├── table.go
│   │   ├── testutils
│   │   │   └── test.go
│   │   └── version.go
│   ├── k8s
│   │   ├── apis
│   │   │   └── egressgateway.spidernet.io
│   │   └── client
│   │       ├── clientset
│   │       ├── informers
│   │       └── listers
│   ├── lock
│   │   ├── lock_debug.go
│   │   ├── lock_debug_test.go
│   │   ├── lock_fast.go
│   │   ├── lock_fast_test.go
│   │   ├── lock.go
│   │   └── lock_suite_test.go
│   ├── logger
│   │   ├── logger.go
│   │   └── logsink.go
│   ├── profiling
│   │   └── manager.go
│   ├── schema
│   │   └── schema.go
│   ├── types
│   │   └── interface.go
│   └── utils
│       ├── flat.go
│       ├── map.go
│       ├── node.go
│       └── set
│           ├── interface.go
│           └── set.go
├── README.md
├── test
│   ├── doc
│   │   ├── egressgateway.md
│   │   ├── egressnode.md
│   │   ├── egresspolicy.md
│   │   └── stress.md
│   ├── e2e
│   │   ├── common
│   │   │   ├── constant.go
│   │   │   ├── deployment.go
│   │   │   ├── egressconfigmap.go
│   │   │   ├── egressgateway.go
│   │   │   ├── egressnode.go
│   │   │   ├── egresspolicy.go
│   │   │   ├── err.go
│   │   │   ├── ip.go
│   │   │   ├── iptables.go
│   │   │   └── node.go
│   │   ├── egressgateway
│   │   │   ├── egressgateway_suite_test.go
│   │   │   └── egressgateway_test.go
│   │   ├── egressnode
│   │   │   ├── egressnode_suite_test.go
│   │   │   └── egressnode_test.go
│   │   ├── egresspolicy
│   │   │   ├── egresspolicy_suite_test.go
│   │   │   └── egresspolicy_test.go
│   │   ├── err
│   │   │   └── err.go
│   │   ├── example
│   │   │   ├── example_suite_test.go
│   │   │   └── example_test.go
│   │   └── tools
│   │       └── tools.go
│   ├── kindconfig
│   │   └── global-kind.yaml
│   ├── Makefile
│   ├── scripts
│   │   ├── check-nettools-server.sh
│   │   ├── debugCluster.sh
│   │   ├── getPerformanceData.sh
│   │   ├── installCalico.sh
│   │   └── installE2eTools.sh
│   ├── test.go
│   └── yaml
│       ├── calico.yaml
│       ├── grafanadashboards.yaml
│       ├── monitoring.coreos.com_podmonitors.yaml
│       ├── monitoring.coreos.com_probes.yaml
│       ├── monitoring.coreos.com_prometheusrules.yaml
│       ├── monitoring.coreos.com_servicemonitors.yaml
│       ├── testclient.yaml
│       ├── testpod.yaml
│       └── testserver.yaml
├── tools
│   ├── copyright-header.txt
│   ├── golang
│   │   ├── codeCoverage.sh
│   │   ├── crdControllerGen.sh
│   │   ├── crdSdkGen.sh
│   │   ├── e2ecover.sh
│   │   ├── ginkgo.sh
│   │   └── goSwagger.sh
│   ├── images
│   │   ├── get-image-digest.sh
│   │   └── update-golang-image.sh
│   ├── scripts
│   │   └── changelog.sh
│   └── tools.go
├── VERSION
└── vendor
```