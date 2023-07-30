# 自建集群安装 egressGateway

## 介绍

本文将演示在一个自建集群上快速安装 egressGateway

## 要求

1. 已经具备一个自建好的 kubernetes 集群，至少有 2 个节点。当前，egressGateway 支持的 CNI 只包含 calico

2. 集群准备好 helm 工具

## 安装准备

* 对于 CNI 是 calico 的集群，请执行如下命令，该命令确保 egressGateway 的 iptables 规则不会被 calico 规则覆盖，否则 egressGateway 将不能工作。

```shell
kubectl patch FelixConfiguration default --patch '{"spec": {"chainInsertMode": "Append"}}'
```

> `spec.chainInsertMode` 的意义可参考 [calico 文档](https://projectcalico.docs.tigera.io/reference/resources/felixconfig)：

## 安装 egressGateway

### 添加 egressGateway 仓库

```shell
helm repo add egressgateway https://spidernet-io.github.io/egressgateway/
helm repo update
```

### 安装 egressgateway

可使用如下命令快速安装 egressgateway

```shell
helm install egressgateway  egressgateway/egressgateway -n kube-system \
	--set feature.tunnelIpv4Subnet="192.200.0.1/16" \
	--set feature.tunnelIpv6Subnet="fd01::21/112"
```

> 安装命令中，需要提供用于 egressgateway 隧道节点的 ipv4 和 ipv6 网段，要求该网段和集群内的其他地址不冲突
> 如果不希望使用 ipv6 ，可使用选项 --set feature.enableIPv6=false 关闭

确认所有的 egressgateway pod 运行正常

```shell

```

## 创建 EgressGateway 实例

EgressGateway 定义了一组节点作为集群的出口网关，集群内的 egress 流量将会通过这组节点转发而出集群。
因此，我们需要预先定义一组 EgressGateway，例子如下

```yaml
cat <<EOF | kuebctl apply -f -
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressGateway
metadata:
  name: default
spec:
  ippools:
    ipv4:
    - "10.6.1.60-10.6.1.66"
  nodeSelector:
    selector:
      matchLabels:
        egressgateway: true
EOF
``` 

> * 集群的 egress 流量的源 IP 地址，可使用网关节点的 IP，也可使用独立的 VIP。 
> 如上 yaml 例子中，spec.ippools.ipv4 定义了一组 egress 的 出口 VIP 地址，需要根据具体环境的实际情况调整，
> 其中，`spec.ippools.ipv4` 的 CIDR 应该是与网关节点上的出口网卡（一般情况下是默认路由的网卡）的子网相同，否则，极有可能导致 egress 访问不通。
> 
> * 通过 EgressGateway 的 spec.nodeSelector 来 select 一组节点作为出口网关，它支持 select 多个节点来实现高可用。

给出口网关节点打上 label，可以给多个 node 打上 label，作为生成环境，建议 2 个节点，作为 POC 环境， 建议 1 个节点即可

```shell
kubectl label node NodeName egressgateway=true
```

查看状态
```shell


```

## 创建一个测试 Pod 应用

创建一个测试 Pod，以模拟需要出口 Egress IP 的应用程序。

```shell
kubectl create deployment client --image nginx
```

## 创建 EgressPolicy CR 对象

EgressPolicy 实例用于定义哪些 POD 的出口流量要经过 egressGateway node 转发，以及其它的配置细节。
可创建如下例子，当匹配的 pod 访问任意集群外部的地址（任意不是 node ip、CNI pod CIDR、clusterIP 的地址）时，都会被 egressGateway node 转发

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressPolicy
metadata:
  name: test-client
  namespace: default
spec:
  appliedTo:
    podSelector:
      matchLabels:
        app: client
```

## 测试结果

我们可以看到 mock-app 访问外部服务时对端看到的 IP 是 EgressGateway 的 IP 地址。

```shell
kubectl exec -it mock-app bash
$ curl 10.6.1.92:8080
Remote IP: 10.6.1.60
```
