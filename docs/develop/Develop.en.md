## Quick Start

1. `make build_local_image`
2. `make e2e_init`
3. `make e2e_run`
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