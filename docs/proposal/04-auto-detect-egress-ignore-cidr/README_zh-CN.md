# Egress ignore CIDR

## 动机

为了简化 Egress 策略的配置，引入 Egress Ignore CIDR 功能，允许以手动和自动的方式获取集群的 CIDR。当 EgressGatewayPolicy 的 `destSubnet` 字段为空时，数据面将会自动匹配 EgressClusterStatus CR 中的 CIDR 之外的流量，并将其转发到 Egress 网关。

## 目标

* 优化 EgressGatewayPolicy 使用体验

## 设计

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

### EgressClusterStatus CRD

集群级 CRD。

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

### 数据面策略

#### ipset 

数据面在处理 `destSubnet` 为空的的 EgressGatewayPolicy 时，将 iptables 匹配策略改为 `--match-set !egress-ingore-cidr dst`。

```shell
iptables -A EGRESSGATEWAY-MARK-REQUEST -t mangle -m conntrack --ctdir ORIGINAL \
-m set --match-set !egress-ingore-cidr dst  \
-m set --match-set $IPSET_RULE_SRC_NAME src  \
-j MARK --set-mark $NODE_MARK -m comment --comment "rule uuid: mark request packet"
```

### 代码设计

#### Controller

新增一个控制循环，根据 `egressIgnoreCIDR.autoDetect` 配置来 Watch 集群的相关资源，更新自动检测的 CIDR 到 EgressClusterStatus CR 的 `status.egressIgnoreCIDR` 中。

#### Agent

* 在 Policy 的控制循环中，处理 EgressClusterStatus 更新到名为 `egress-ingore-cidr` 的 ipset 中；
* 对于 `destSubnet` 字段为空时的 EgressGatewayPolicy 策略，使用 `egress-ingore-cidr` 的 ipset 匹配流量。
