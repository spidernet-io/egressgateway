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
    namespaceSelector:   # (1)
      matchLabels:
        app: "shopping"
  destSubnet:
    - "10.6.1.92/32"
    - "fd00::92/128"
```

## Definition

### Metadata

| Field     | Description                                | Schema | Validation |
|-----------|--------------------------------------------|--------|------------|
| namespace | The namespace of the EgressPolicy resource | string | required   |
| name      | The name of the EgressPolicy resource      | string | required   |

### Spec

| Field             | Description                                                                                                                                                                                                                                                    | Schema                  | Validation | Values        | Default |
|-------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------|------------|---------------|---------|
| egressGatewayName | Reference to the EgressGateway to use                                                                                                                                                                                                                          | string                  | required   |               |         |
| egressIP          | Configuration for the egress IP settings                                                                                                                                                                                                                       | [egressIP](#egressIP)   | optional   |               |         |
| appliedTo         | Selector for the Pods to which the EgressPolicy should be applied                                                                                                                                                                                              | [appliedTo](#appliedTo) | required   |               |         |
| destSubnet        | When accessing the subnets in this list, use the Egress IP. If `feature.clusterCIDR.autoDetect` was enabled during installation and `destSubnet` is not configured, then access to external networks outside the cluster will automatically use the Egress IP. | []string                | optional   | CIDR notation |         |
| priority          | Priority of the policy                                                                                                                                                                                                                                         | integer                 | optional   |               |         |

#### egressIP

| Field     | Description                                                                                               | Schema   | Validation | Values      | Default |
|-----------|-----------------------------------------------------------------------------------------------------------|----------|------------|-------------|---------|
| ipv4      | Specific IPv4 address to use if defined                                                                   | string   | optional   | valid IPv4  |         |
| ipv6      | Specific IPv6 address to use if defined                                                                   | string   | optional   | valid IPv6  |         |
| useNodeIP | Flag to indicate if the Node IP should be used as the Egress IP when no specific IP address is defined    | bool     | optional   | true/false  | false   |

#### appliedTo

| Field             | Description                                                                                                                                                                                                                         | Schema            | Validation | Values | Default |
|-------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------|------------|--------|---------|
| podSelector       | Use Egress Policy on Pods Matched by Selector                                                                                                                                                                                       | map[string]string | optional   |        |         |
| podSubnet         | Use Egress Policy on Pods Matched by Subnet (Not Implemented)                                                                                                                                                                       | []string          | optional   | CIDR   |         |
| namespaceSelector | The `namespaceSelector` uses a selector to select the list of matching namespaces. Within the selected namespace scope, use the `podSelector` to select the matching Pods, and then apply the Egress policy to these selected Pods. |                   |            |        |         |
