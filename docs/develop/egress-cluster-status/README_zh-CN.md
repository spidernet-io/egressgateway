# EgressClusterStatus

## 简介
为了简化 Egress 策略的配置，引入 Egress Ignore CIDR 功能，允许以手动和自动的方式获取集群的 CIDR。当 EgressGatewayPolicy 的 `destSubnet` 字段为空时，数据面将会自动匹配 EgressClusterStatus CR 中的 CIDR 之外的流量，并将其转发到 Egress 网关。

## CRD

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressClusterStatus
metadata:
   name: "default"  # 1
status:
  egressIgnoreCIDR:
    nodeIP:
      ipv4:
      - "10.6.0.1"
      ipv6:
      - "fd00::1"
    clusterIP:
      ipv4:
      - "10.6.0.1"
      ipv6:
      - "fd00::1"
    podCIDR:
      ipv4:
      - "10.6.0.0/24"
      ipv6:
      - "fd00::1/122"
```

1. 名称为 `default`，由系统维护只能创建一个;
2. 根据 `egressIgnoreCIDR.autoDetect` 配置检测出的集群 CIDR 或 IP。

## 代码设计

### 初始化

### Controller

### agent

## 其他

### 配置文件

修改配置文件，增加如下配置：

```yaml
feature:
  egressIgnoreCIDR:
    autoDetect:
      podCIDR: ""     # 1
      clusterIP: true # 2
      nodeIP: true    # 3
    custom:
      - "10.6.1.0/24"
```

1. 支持设置为支持 kube-ovn, calico, k8s 等；
2. 支持设置为 Service CIDR 自动检测；
3. 支持设置为 Node IP 自动检测，当新加一个节点时，自动将节点的的所有 IP 更新到 EgressClusterStatus。