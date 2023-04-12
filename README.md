# egressgateway

[![Auto Nightly CI](https://github.com/spidernet-io/egressgateway/actions/workflows/auto-nightly-ci.yaml/badge.svg)](https://github.com/spidernet-io/egressgateway/actions/workflows/auto-nightly-ci.yaml)
[![Auto Release Version](https://github.com/spidernet-io/egressgateway/actions/workflows/auto-release.yaml/badge.svg)](https://github.com/spidernet-io/egressgateway/actions/workflows/auto-release.yaml)
[![codecov](https://codecov.io/gh/spidernet-io/egressgateway/branch/main/graph/badge.svg?token=8CCT4CIIPx)](https://codecov.io/gh/spidernet-io/egressgateway)
[![Go Report Card](https://goreportcard.com/badge/github.com/spidernet-io/egressgateway)](https://goreportcard.com/report/github.com/spidernet-io/egressgateway)
[![CodeFactor](https://www.codefactor.io/repository/github/spidernet-io/egressgateway/badge)](https://www.codefactor.io/repository/github/spidernet-io/egressgateway)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=spidernet-io_egressgateway&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=spidernet-io_egressgateway)
![badge](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/bzsuni/cc6d42eb27d8ee4c3d19c936eff2c478/raw/egressgatewaye2e.json)

## Background

<img src="./docs/proposal/01-egress-gateway/Egress Gateway.png" width="76%"></img>

Starting with 2021, we received some feedback as follows.

There are two clusters A and B. Cluster A is VMWare-based and runs mainly Database workloads, and Cluster B is a Kubernetes cluster. Some applications in Cluster B need to access the database in Cluster A, and the network administrator wants the cluster Pods to be managed through an egress gateway.

## Summary

The gateway provides network egress capabilities for Kubernetes clusters.

### features

* Solve IPv4 IPv6 dual-stack connectivity.
* Solve the high availability of Egress Nodes.
* Allow filtering Pods Egress Policy (_Destination CIDR_).
* Allow filtering of egress Applications (_Pods_).
* Can be used in low kernel version.
* Support multiple egress gateways instance.
* Support namespaced egress IP.
* Supports automatic detection of cluster traffic for egress gateways policies.
* Support namespace default egress instances.

### compatibility

* Calico

### CRDs

* EgressNode
* EgressGateway
* EgressGatewayPolicy
* EgressEndpointSlice

You can follow the [Get Started](https://spidernet-io.github.io/egressgateway/usage/install) to set up your own playground!

## Develop

<img src="./docs/proposal/03-egress-ip/arch.png" width="100%"></img>

Refer to [develop](./docs/develop/dev.md).

## License

EgressGateway is licensed under the Apache License, Version 2.0. See [LICENSE](https://github.com/spidernet-io/spiderpool/blob/main/LICENSE) for the full license text.