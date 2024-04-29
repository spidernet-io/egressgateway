The EgressPolicy CRD is used to specify the Pods and its destination CIDRs for which an Egress strategy should be applied, along with the corresponding IP addresses to be used for Egress.

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressPolicy
metadata:
  namespace: "default"
  name: "policy-test"
spec:
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
  priority: 100
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

| Field       | Description                                                   | Schema            | Validation | Values | Default |
|-------------|---------------------------------------------------------------|-------------------|------------|--------|---------|
| podSelector | Use Egress Policy on Pods Matched by Selector                 | map[string]string | optional   |        |         |
| podSubnet   | Use Egress Policy on Pods Matched by Subnet (Not Implemented) | []string          | optional   | CIDR   |         |
