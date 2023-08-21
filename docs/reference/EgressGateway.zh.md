EgressGateway CRD 用于选择一组节点作为集群的 Egress 节点，并为该节点组配置 Egress IP 池。Egress IP 可在此组节点内浮动。集群级资源。

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

1. 设置 EgressGateway 可使用的 Egress IP 池的范围；
2. Egress IPv4 池，支持三种方法：单个 IP `10.6.0.1`、范围 `10.6.0.1-10.6.0.10` 和 CIDR `10.6.0.1/26`；
3. Egress IPv6 池，如果启用了双栈要求，则 IPv4 和 IPv6 的数量必须一致，格式与 IPv4 相同；
4. 要使用的默认 IPv4 EIP。如果 EgressPolicy 没有指定 EIP，且 EIP 分配策略为 `default`，则分配给该 EgressPolicy 的 EIP 将是 `ipv4DefaultEIP`；
5. 要使用的默认 IPv6 EIP，规则与 `ipv6DefaultEIP` 相同；
6. 设置 Egress 节点的匹配条件和策略；
7. 通过 Selector 选择一组节点作为 Egress 节点，Egress IP 可在此范围内浮动；
8. EgressGateway 选择 Egress 节点的策略，目前仅支持平均选择；
9. 节点选择器选择的 Egress 节点，以及节点上有效的 Egress IP，以及使用该 Egress IP 的 EgressPolicy；
10. Egress 节点的名称；
11. Egress 节点对应的 EgressTunnel 对象的状态；
12. 此 Egress 节点上有效的 EIP 信息；
13. Egress IPv4，如果 EgressPolicy 和 EgressClusterPolicy 使用节点 IP，则此字段为空；
14. Egress IPv6，在双栈情况下，IPv4 和 IPv6 一一对应；
15. 哪些策略使用此节点上的有效 Egress IP；
16. 使用 Egress IP 的策略名称；
17. 使用 Egress IP 的策略的命名空间。

