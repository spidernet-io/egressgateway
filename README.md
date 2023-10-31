# egressgateway

[![Auto Nightly CI](https://github.com/spidernet-io/egressgateway/actions/workflows/auto-nightly-ci.yaml/badge.svg)](https://github.com/spidernet-io/egressgateway/actions/workflows/auto-nightly-ci.yaml)
[![Auto Release Version](https://github.com/spidernet-io/egressgateway/actions/workflows/auto-release.yaml/badge.svg)](https://github.com/spidernet-io/egressgateway/actions/workflows/auto-release.yaml)
[![codecov](https://codecov.io/gh/spidernet-io/egressgateway/branch/main/graph/badge.svg?token=8CCT4CIIPx)](https://codecov.io/gh/spidernet-io/egressgateway)
[![Go Report Card](https://goreportcard.com/badge/github.com/spidernet-io/egressgateway)](https://goreportcard.com/report/github.com/spidernet-io/egressgateway)
[![CodeFactor](https://www.codefactor.io/repository/github/spidernet-io/egressgateway/badge)](https://www.codefactor.io/repository/github/spidernet-io/egressgateway)
![badge](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/bzsuni/cc6d42eb27d8ee4c3d19c936eff2c478/raw/egressgatewaye2e.json)
[![OpenSSF Best Practices](https://bestpractices.coreinfrastructure.org/projects/7410/badge)](https://bestpractices.coreinfrastructure.org/projects/7410)

## About

EgressGateway is a network management tool designed for Kubernetes clusters, with a primary focus on managing the egress traffic of Pods to external networks. It addresses challenges related to inter-cluster communication, egress policy control, and high availability. Additionally, it offers support for various network solutions and custom resource definitions (CRDs), enabling users to configure and manage egress policies with flexibility.

## Why EgressGateway

### Support a range of features and advantages

* Address IPv4 and IPv6 dual-stack connectivity issues, ensuring seamless communication across different protocol stacks.
  
* Resolve high availability concerns for Egress nodes, ensuring network connectivity remains unaffected by single-point failures.

* Provide finer-grained policy control, allowing flexible filtering of Pods' Egress policies, including Destination CIDR.

* Support application-level control, allowing EgressGateway to filter Egress applications (Pods) for precise management of specific application outbound traffic.

* Compatible with lower kernel versions, making EgressGateway suitable for various Kubernetes deployment environments.

* Support multiple Egress gateway instances, capable of handling communication between multiple network partitions or clusters.

* Offer namespace-level Egress IP support.

* Enable automatic detection of Egress gateway policies for cluster traffic, simplifying traffic management and configuration.

* Provide namespace default Egress instances.

### Compatible with the following network solutions

* Calico
* Flannel
* Weave
* Spiderpool

You can follow the [Get Started](https://spidernet-io.github.io/egressgateway/usage/Install) to set up your own playground!

## Architecture

<img src="./docs/proposal/03-egress-ip/arch.png" width="100%"></img>

The architecture consists of two parts: the control plane and the data plane. The control plane comprises four control loops, while the data plane consists of three control loops. The control plane is deployed using the Deployment method, supporting high availability with multiple replicas, and the data plane is deployed using DaemonSet.

## To start using EgressGateway

Please refer to the [development documentation](https://spidernet-io.github.io/egressgateway/v0.2/reference/EgressTunnel/).
Please refer to the [installation guide](https://spidernet-io.github.io/egressgateway/v0.2/usage/Install/).

## Join the EgressGateway Community
We welcome contributions in any kind. If you have any questions about contributions, please consult the [contribution documentation](https://github.com/spidernet-io/egressgateway/blob/main/docs/develop/Contribute.en.md).

## License

EgressGateway is licensed under the Apache License, Version 2.0. See [LICENSE](https://github.com/spidernet-io/spiderpool/blob/main/LICENSE) for the full license text.
