The EgressGateway CRD is used to select a group of nodes as the Egress nodes of the cluster and configure the Egress IP pool for this group of nodes. The Egress IP can float within this range. Cluster scope resource.

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressGateway
metadata:
  name: "eg1"
spec:
  ippools:                      # 1
    ipv4:                       # 2
      - "10.6.1.55"
      - "10.6.1.60-10.6.1.65"
      - "10.6.1.70/28"
    ipv6:                       # 3
      - ""
    ipv4DefaultEIP: ""          # 4
    ipv6DefaultEIP: ""          # 5
  nodeSelector:                 # 6
    selector:                   # 7
      matchLabels:
        egress: "true"
    policy: "doing"             # 8
status:                         
  nodeList:                     # 9
    - name: "node1"             # 10
      status: "Ready"           # 11
      epis:                     # 12
        - ipv4: "10.6.1.55"     # 13
          ipv6: "fd00::55"      # 14
          policies:             # 15
            - name: "app"         # 16
              namespace: "default"  # 17
```

1. Set the range of egress IP pool that EgressGateway can use;
2. Egress IPv4 pool, supporting three methods: single IP `10.6.0.1`, range `10.6.0.1-10.6.0.10`, and CIDR `10.6.0.1/26`;
3. Egress IPv6 pool, if dual-stack requirements are enabled, the number of IPv4 and IPv6 must be consistent, and the format is the same as IPv4;
4. The default IPv4 EIP to use. If the EgressPolicy does not specify EIP and the EIP assignment policy is `default`, the EIP assigned to this EgressPolicy will be `ipv4DefaultEIP`;
5. The default IPv6 EIP to use, the rules are the same as `ipv6DefaultEIP`;
6. Set the matching conditions and policy for egress nodes;
7. Select a group of nodes as egress gateway nodes through Selector, and egress IP can float within this range;
8. The policy for EgressGateway to select Egress nodes, currently only supports average selection;
9. The egress nodes selected by node selector, as well as the effective egress IP on the node, and the EgressPolicy that uses this egress IP;
10. The name of the Egress node;
11. The status of the Egress node;
12. The effective EIP information on this gateway node;
13. Egress IPv4, if EgressPolicy and EgressClusterPolicy use node IP, this field is empty;
14. Egress IPv6, in the dual-stack situation, IPv4 and IPv6 are one-to-one corresponding;
15. Which policies are using the effective Egress IP on this node;
16. Name of the Policy using the Egress IP;
17. Namespace of the Policy using the Egress IP.

