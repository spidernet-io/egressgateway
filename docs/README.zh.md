# EgressGateway

EgressGateway 是用于 Kubernetes 集群的网络管理工具，专注于管理 Pods 对外部网络的出口流量，解决多集群通信、出口策略控制和高可用性问题，同时支持多种网络解决方案和自定义资源定义 (CRDs)，使用户能够更灵活地配置和管理出口策略。

## 为什么选择 EgressGateway

### 提供了一系列功能和优势

* **解决 IPv4 和 IPv6 双栈连接问题**，确保网络通信在不同协议栈下的无缝连接。
  
* **解决 Egress 节点的高可用性问题**，确保网络连通性不受单点故障的干扰。

* **支持更精细的策略控制**，您可以通过 EgressGateway 灵活地过滤 Pods 的 Egress 策略，包括 Destination CIDR。
  
* **支持应用程序级别的控制**，EgressGateway 允许过滤 Egress 应用程序（Pods），使您能够更精确地管理特定应用的出口流量。
  
* **支持低内核版本**，EgressGateway 可以在低内核版本中使用，适用于各种 Kubernetes 部署环境。
  
* **支持多 Egress 网关实例**，能够处理多个网络分区或集群之间的通信。
  
* **支持命名空间级别的 Egress IP**。
  
* **支持自动检测集群流量的 Egress 网关策略**，简化流量管理和配置。
  
* **支持命名空间默认 Egress 实例**。

### 兼容以下网络解决方案

* Calico
* Flannel
* Weave
* Spiderpool

你可以跟随[安装](https://spidernet-io.github.io/egressgateway/zh/usage/Install)指南搭建你自己的测试环境。

### 架构

<img src="./proposal/03-egress-ip/arch.png" width="100%"></img>

架构包含：控制面和数据面 2 部分组成，控制面由 4 个控制循环组成，数据面由 3 个控制循环组成。控制面以 Deployment 方式部署，支持多副本高可用，数据面以 DaemonSet 的方式部署。

## 开始使用 EgressGateway

参考[开发](develop/Develop.en.md)文档
参考[安装指南](https://spidernet-io.github.io/egressgateway/v0.2/usage/Install/)

## 社区

我们欢迎任何形式的贡献。如果您有任何有关贡献方面的疑问，请参阅[贡献指南](https://github.com/spidernet-io/egressgateway/blob/main/docs/develop/Contribute.en.md)。

## License

EgressGateway 基于 Apache License，Version 2.0。详细参考 [LICENSE](https://github.com/spidernet-io/spiderpool/blob/main/LICENSE) 查看完整 LICENSE 内容。
