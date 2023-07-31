## Quick Start

1. build local CI image 

        make build_local_image

2. setup cluster

        # setup the cluster
        make e2e_init

        # for china developer, use china image registry, use HTTP_PROXY to pull chart 
        make e2e_init -e E2E_CHINA_IMAGE_REGISTRY=true -e HTTP_PROXY=http://10.0.0.1:7890

        # show the cluster
        export KUBECONFIG=$(pwd)/test/runtime/kubeconfig_egressgateway.config
        kubectl get node

3. run the E2E test 

        make e2e_run

4. check proscope, browser visits http://nodeIP:4040

### Go Package (Structure) Design

```bash
.
├── api
├── charts
├── cmd
├── docs
├── images
├── output
├── pkg
│   ├── agent
│   ├── coalescing
│   ├── config
│   ├── constant
│   ├── controller
│   ├── egressgateway
│   ├── ethtool
│   ├── ipset
│   ├── iptables
│   ├── k8s
│   ├── layer2
│   ├── lock
│   ├── logger
│   ├── markallocator
│   ├── profiling
│   ├── schema
│   ├── types
│   └── utils
├── test
│   ├── doc
│   ├── e2e
│   ├── kindconfig
│   ├── scripts
│   └── yaml
├── tools
│   ├── golang
│   ├── images
│   └── scripts
└── vendor
```