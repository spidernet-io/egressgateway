EgressClusterPolicy CRD 用于定义集群级 Egress 策略规则，与 [EgressPolicy](EgressPolicy.zh.md) CRD 类似，但增加了 `spec.appliedTo.namespaceSelector` 字段，其他字段与 EgressPolicy 一致。

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressClusterPolicy
metadata:
  name: "policy-test"
spec:
  priority: 100
  egressGatewayName: "eg1"
  egressIP:
    ipv4: ""
    ipv6: ""
    useNodeIP: false
    allocatorPolicy: default
  appliedTo:
    podSelector:
      matchLabels:
        app: "shopping"
    podSubnet:
      - "172.29.16.0/24"
      - 'fd00:1/126'
    namespaceSelector:   # (1)
      matchLabels:
        app: "shopping"
  destSubnet:
    - "10.6.1.92/32"
    - "fd00::92/128"
status:
  eip:
    ipv4: 172.18.1.2
    ipv6: fc00:f853:ccd::9
  node: egressgateway-worker
```

1. `namespaceSelector` 使用 selector 选择匹配的命名空间列表。在选定的命名空间范围内，使用 `podSelector` 选择匹配的 Pod，然后对这些选中的 Pod 应用 Egress 策略。
