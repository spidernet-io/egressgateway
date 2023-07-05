The EgressPolicy CRD is used to specify which Pods access which target CIDRs using Egress policies, as well as the IP addresses used by Egress. Namespaced resource.

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressPolicy
metadata:
  namespace: "default"
  name: "policy-test"
spec:
  egressGatewayName: "eg1"  # 1
  egressIP:                 # 2
    ipv4: ""                            
    ipv6: ""
    useNodeIP: false        # 3
  appliedTo:                # 4
    podSelector:            # 4-a 
      matchLabels:    
        app: "shopping"
    podSubnet:              # 4-b
    - "172.29.16.0/24"
    - 'fd00:1/126'
  destSubnet:               # 5
    - "10.6.1.92/32"
    - "fd00::92/128"
  priority: 100             # 6
```

1. Select the EgressGateway referenced by the EgressPolicy.
2. Egress IP represents the EgressIP settings used by the EgressPolicy:
    * If `ipv4` or `ipv6` addresses are defined when creating, an IP address will be allocated from the EgressGateway's `.ippools`. If policy1 requests `10.6.1.21` and `fd00:1` and then policy2 requests `10.6.1.21` and `fd00:2`, an error will occur, causing policy2 allocation to fail.
    * If `ipv4` or `ipv6` addresses are not defined and `useNodeIP` is true, the Egress address will be the Node IP of the referenced EgressGateway.
    * If `ipv4` or `ipv6` addresses are not defined when creating and `useNodeIP` is `false`, an IP address will be automatically allocated from the EgressGateway's `.ranges` (when IPv6 is enabled, both an IPv4 and IPv6 address will be requested).
    * `egressGatewayName` must not be empty.
3. Supports using the Node IP as the Egress IP (only one option can be chosen).
4. Select the Pods to which the EgressPolicy should be applied.
   a. Select by using Label
   b. Specify the Pod subnet directly (options a and b cannot be used simultaneously)
5. Specify the target addresses for accessing Egress. If no target addresses are specified, the effective policy will forward all traffic to Egress nodes when the destination is outside the cluster CIDR.
6. Priority of the policy.