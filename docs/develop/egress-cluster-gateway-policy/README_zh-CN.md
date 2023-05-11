# EgressClusterGatewayPolicy

## 简介

集群级别的 EgressClusterGatewayPolicy

## CRD
```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressClusterGatewayPolicy
metadata:
  namespace: "default"
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

1. namespaceSelector(LabelSelector): 通过标签筛选符合要求的命名空间

其他字段与 EgressGatewayPolicy 一致

## 代码设计

### 初始化

### Controller

### agent

## 其他