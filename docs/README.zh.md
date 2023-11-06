# EgressGateway

EgressGateway 是用于 Kubernetes 集群的出口网关策略解决方案，专注于管理 Pods 对外部网络的出口流量，解决多集群通信、出口策略控制和高可用性问题，同时支持多种网络解决方案和自定义资源定义 (CRDs)，使用户能够更灵活地配置和管理出口策略。

## 架构

![Architecture](./architecture.png)

## 为什么选择 EgressGateway

### 提供了一系列功能和优势

* 解决 IPv4/IPv6 双栈连接问题，确保网络通信在不同协议栈下的无缝连接。
* 解决 Egress 节点的高可用性问题，确保网络连通性不受单点故障的干扰。
* 允许更精细的策略控制，可以通过 EgressGateway 灵活地过滤 Pods 的 Egress 策略，包括 Destination CIDR。
* 允许过滤 Egress 应用（Pod），能够更精确地管理特定应用的出口流量。
* 支持多个出口网关实例，能够处理多个网络分区或集群之间的通信。
* 支持租户级别的 Egress IP。
* 支持自动检测集群流量的 Egress 网关策略。
* 支持命名空间默认 Egress 实例。
* 可用于较低内核版本，适用于各种 Kubernetes 部署环境。

### 兼容以下网络解决方案

* [Calico](https://github.com/projectcalico/calico)
* [Flannel](https://github.com/flannel-io/flannel)
* [Weave](https://github.com/weaveworks/weave)
* [Spiderpool](https://github.com/spidernet-io/spiderpool)

## 开始使用 EgressGateway

参考[安装指南](usage/Install.zh.md)

## 社区

我们欢迎任何形式的贡献。如果您有任何有关贡献方面的疑问，请参阅[贡献指南](develop/Contribute.en.md)。

## License

EgressGateway 基于 Apache License，Version 2.0。详细参考 [LICENSE](https://github.com/spidernet-io/spiderpool/blob/main/LICENSE) 查看完整 LICENSE 内容。
