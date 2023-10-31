# egressgateway

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
