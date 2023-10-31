EgressGateway 项目为 Kubernetes 提供 Egress 能力。

<img src="./proposal/01-egress-gateway/Egress-Gateway.png" width="76%"></img>

从2021年开始，我们收到了以下反馈。

有两个集群 A 和 B。集群 A 基于 VMWare 并主要运行数据库负载，集群 B 是一个 Kubernetes 集群。集群 B 中的某些应用需要访问集群 A 中的数据库，而网络管理员希望通过出口网关管理集群的 Pod。

## 特性

* 解决 IPv4/IPv6 双栈连接问题
* 解决 Egress 节点的高可用性问题
* 允许过滤 Pod 的 Egress 策略（_目标 CIDR_）
* 允许过滤 Egress 应用（_Pod_）
* 可用于较低内核版本
* 支持多个出口网关实例
* 支持租户级别的 Egress IP
* 支持自动检测集群流量的 Egress 网关策略
* 支持命名空间默认 Egress 实例

### 兼容性

* Calico
* Flannel
* Weave
* Spiderpool

### CRDs

* EgressTunnel
* EgressGateway
* EgressPolicy
* EgressClusterPolicy
* EgressEndpointSlice
* EgressClusterEndpointSlice
* EgressClusterInfo

你可以跟随[起步](https://spidernet-io.github.io/egressgateway/zh/usage/Install)指南搭建你自己的测试环境～

## Develop

<img src="./proposal/03-egress-ip/arch.png" width="100%"></img>

参考[开发](develop/Develop.en.md)文档。

## License

EgressGateway 基于 Apache License，Version 2.0。详细参考 [LICENSE](https://github.com/spidernet-io/spiderpool/blob/main/LICENSE) 查看完整 LICENSE 内容。
