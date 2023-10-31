# EgressGatewayPolicyCluster

## 动机

补充没有集群级别 policy 的功能

## 目标

* 支持集群级别的 policy
* 支持优先级

## 设计
* 集群级别的 policy，多了 `namespaceSelector` 字段。为空时，则作用于整个集群。不为空时，则作用于符合条件的 NS。
* 由于新增了集群级别的 policy，当集群级别与 namespace 级别的两个 policy， `appliedTo`、`destSubnet` 一致，但 `egressGatewayName`  或`egressIP` 不一致时，两个策略谁最终生效就将成为一个问题。所以引入一个优先级的新字段 `priority` 来解决该问题。范围为 1-65536，数值越小，优先级越高。用户可以自行设置优先级。如果没设置时，EgressGatewayPolicy 默认优先级为 1000，EgressGatewayPolicyCluster 默认优先级为 32768.如果优先级一致时，则随机排序。


## EgressPolicy CRD
```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressPolicy
metadata:
  namespace: "default"
  name: "policy-test"
spec:
  priority: 100             # 1
  egressGatewayName: "eg1"
  egressIP:
    ipv4: ""                            
    ipv6: ""
    useNodeIP: false
  appliedTo:
    podSelector:
      matchLabels:    
        app: "shopping"
    podSubnet:
    - "172.29.16.0/24"
    - 'fd00:1/126'
  destSubnet:
    - "10.6.1.92/32"
    - "fd00::92/128"
```

1. 新增字段，策略的优先级

## EgressClusterPolicy CRD
```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressClusterPolicy
metadata:
  namespace: "default"
  name: "policy-test"
spec:
  priority: 100             # 1
  egressGatewayName: "eg1"
  egressIP:
    ipv4: ""                            
    ipv6: ""
    useNodeIP: false
  appliedTo:
    podSelector:
      matchLabels:    
        app: "shopping"
    podSubnet:
    - "172.29.16.0/24"
    - 'fd00:1/126'
    namespaceSelector:      # 1
      matchLabels:    
        app: "shopping"
  destSubnet:
    - "10.6.1.92/32"
    - "fd00::92/128"
```

1. 策略的优先级
2. namespace 筛选器


其他方面，两者一致