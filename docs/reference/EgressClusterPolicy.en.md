The EgressClusterPolicy CRD is used to define cluster-level Egress policy rules, similar to the [EgressPolicy](EgressPolicy.en.md) CRD, but with the added `spec.appliedTo.namespaceSelector` attribute.

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
  appliedTo:
    podSelector:
      matchLabels:
        app: "shopping"
    podSubnet:
      - "172.29.16.0/24"
      - 'fd00:1/126'
    namespaceSelector:   # 1
      matchLabels:
        app: "shopping"
  destSubnet:
    - "10.6.1.92/32"
    - "fd00::92/128"
```

1. The `namespaceSelector` uses a selector to select the list of matching namespaces. Within the selected namespace scope, use the `podSelector` to select the matching Pods, and then apply the Egress policy to these selected Pods.
