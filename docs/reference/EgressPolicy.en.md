The EgressPolicy CRD is used to specify the Pods and its destination CIDRs for which an Egress strategy should be applied, along with the corresponding IP addresses to be used for Egress.

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressPolicy
metadata:
  namespace: "default"
  name: "policy-test"
spec:
  egressGatewayName: "eg1"  # (1)
  egressIP:                 # (2)
    ipv4: ""                            
    ipv6: ""
    useNodeIP: false        # (3)
  appliedTo:                
    podSelector:            # (4) 
      matchLabels:    
        app: "shopping"
    podSubnet:              # (5)
    - "172.29.16.0/24"
    - 'fd00:1/126'
  destSubnet:               # (6)
    - "10.6.1.92/32"
    - "fd00::92/128"
  priority: 100             # (7)
```

1. Select the EgressGateway referenced by the EgressPolicy.
2. Egress IP represents the EgressIP settings used by the EgressPolicy:
    * If `ipv4` or `ipv6` addresses are defined when creating, an IP address will be allocated from the EgressGateway's `.ippools`. If policy1 requests `10.6.1.21` and `fd00:1` and then policy2 requests `10.6.1.21` and `fd00:2`, an error will occur, causing policy2 allocation to fail.
    * If `ipv4` or `ipv6` addresses are not defined and `useNodeIP` is true, the Egress address will be the Node IP of the referenced EgressGateway.
    * If `ipv4` or `ipv6` addresses are not defined when creating and `useNodeIP` is `false`, an IP address will be automatically allocated from the EgressGateway's `.ranges` (when IPv6 is enabled, both an IPv4 and IPv6 address will be requested).
    * `egressGatewayName` must not be empty.
3. Support using the Node IP as the Egress IP (only one option can be chosen).
4. Select the Pods to which the EgressPolicy should be applied by using Label.
5. Select the Pods to which the EgressPolicy should be applied by specifying the Pod subnet directly (options 4 and 5 cannot be used simultaneously)
6. When specifying the destination addresses for Egress access, if no specific destination address is provided, the following policy will be enforced: requests with destination addresses outside of the cluster's internal CIDR range will be forwarded to the Egress node.
7. Priority of the policy.