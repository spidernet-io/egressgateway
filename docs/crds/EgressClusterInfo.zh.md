## 简介

为了简化 Egress 策略的配置，引入 Egress Ignore CIDR 功能，允许自动获取集群的 CIDR。当 EgressGatewayPolicy 的 `destSubnet` 字段为空时，数据面将会自动匹配 EgressClusterStatus CR 中的 CIDR 之外的流量，并将其转发到 Egress 网关。

## CRD

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressClusterInfo
metadata:
  name: "default"    # 1
spec: {}
status:
  egressIgnoreCIDR:  # 2
    clusterIP:       # 3
      ipv4:
      - "172.41.0.0/16"
      ipv6:
      - "fd41::/108"
    nodeIP:
      ipv4:
      - "172.18.0.3"
      - "172.18.0.4"
      - "172.18.0.2"
      ipv6:
      - "fc00:f853:ccd:e793::3"
      - "fc00:f853:ccd:e793::4"
      - "fc00:f853:ccd:e793::2"
    podCIDR:
      ipv4:
      - "172.40.0.0/16"
      ipv6:
      - "fd40::/48"
```

1. 名称默认为 `default`，由系统维护，只能创建一个，不可被修改。
2. `egressIgnoreCIDR` 定义 egressGateway 要忽略的 cidr。
3. `clusterIP` 集群默认的 service-cluster-ip-range。是否开启，由 egressgateway 配置文件默认的 `egressIgnoreCIDR.autoDetect.clusterIP` 指定。
4. `nodeIP` 集群节点的 IP（只取 node yaml `status.address` 中的 IP，多卡情况下，其他网卡 IP 被视作集群外 IP 处理）集合。是否开启，由 egressgateway 配置文件默认的 `egressIgnoreCIDR.autoDetect.nodeIP` 指定。
5. `podCIDR` 集群的 cni 使用的 cidr。由 egressgateway 配置文件默认的 `egressIgnoreCIDR.autoDetect.podCIDR` 指定。

## 代码设计

### 初始化

### Controller
#### Node Event

- Create：node 创建时，将 node 的 ip 自动添加到 egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP` 中。
- Update：node ip 有更新时，将 node 的 ip 自动更新到 egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP` 中。
- Delete：node 被删除时，将 node 的 ip 从 egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP` 中删除。

#### Calico IPPool Event

当 egressgateway 配置文件的 `egressIgnoreCIDR.autoDetect.podCIDR` 为 "calico" 时，监听 Calico 的 IPPool Event。
- Create：calico ippool 创建时，将 ippool cidr 自动添加到 egressclusterinfos CR `status.egressIgnoreCIDR.podCIDR` 中。
- Update：calico ippool 有更新时，将 ippool cidr 自动更新到 egressclusterinfos CR `status.egressIgnoreCIDR.podCIDR` 中。
- Delete：calico ippool 被删除时，将 ippool cidr 从 egressclusterinfos CR `status.egressIgnoreCIDR.podCIDR` 中删除。

### Agent

无

## 其他

### 配置文件

修改配置文件，增加如下配置：

```yaml
feature:
  egressIgnoreCIDR:
    autoDetect:
      podCIDR: ""      # 1
      clusterIP: true  # 2
      nodeIP: true     # 3
    custom:
      - "10.6.1.0/24"
```

1. `podCIDR`，目前支持 `calico`、`k8s`。默认为 `k8s`。
2. `clusterIP`，支持设置为 Service CIDR 自动检测。
3. `nodeIP`，支持设置为 Node IP 自动检测。