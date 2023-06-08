# EgressClusterPolicy

## 简介

EgressClusterGatewayPolicy CRD 用于定义集群级 Egress 策略规则。其用法与 EgressGatewayPolicy CRD 相比多了 `spec.appliedTo.namespaceSelector` 属性。

## CRD
```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressClusterGatewayPolicy
metadata:
  name: "policy-test"
spec:
  priority: 100             
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

1. namespaceSelector：该属性使用 selector 选择匹配租户列表，再使用 `podSelector` 选择租户范围下匹配中的 Pod，然后对选择中的 Pod 应用 Egress 策略。

## 代码设计

### 初始化

### Controller

### Agent

## 其他