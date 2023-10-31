# EgressGatewayPolicyCluster

## Motivation

Complementing the lack of cluster-level policy

## Goal

* Support cluster-level policy
* Support for prioritization

## Design

* Cluster-level policy with additional `namespaceSelector` field. When empty, it applies to the entire cluster. If it is not empty, it applies to the eligible NSs.
* Due to the new cluster-level policy, when two policies, `appliedTo` and `destSubnet`, at the cluster level and namespace level are consistent, but `egressGatewayName` or `egressIP` are inconsistent, there will be a problem of which one of the two policies will take effect in the end. So a new field for priority, `priority`, was introduced to solve this problem. The range is 1-65536, the smaller the value, the higher the priority. Users can set the priority themselves. If not set, the default priority of EgressGatewayPolicy is 1000, and the default priority of EgressGatewayPolicyCluster is 32768. If the priority is the same, it will be randomized.

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

1. New field, prioritization of policies

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

1. strategy prioritization
2. namespace filters

Otherwise, both are consistent
