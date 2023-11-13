EgressPolicy CRD 用于指定哪些 Pod 访问哪些目标 CIDR 时走 Egress 策略，以及 Egress 所使用的 IP 地址。租户级资源。

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressPolicy
metadata:
  namespace: "default"
  name: "policy-test"
spec:
  egressGatewayName: "eg1"    # (1)
  egressIP:                   # (2)
    ipv4: ""                            
    ipv6: ""
    useNodeIP: false          # (3)
    allocatorPolicy: default  # (4)
  appliedTo:                
    podSelector:              # (5) 
      matchLabels:    
        app: "shopping"
    podSubnet:                # (6)
    - "172.29.16.0/24"
    - 'fd00:1/126'
  destSubnet:                 # (7)
    - "10.6.1.92/32"
    - "fd00::92/128"
  priority: 100               # (8)
status:
  eip:                        # (9)
    ipv4: 172.18.1.2
    ipv6: fc00:f853:ccd::9
  node: egressgateway-worker  # (10)
```

1. 选择 EgressPolicy 引用的 EgressGateway：
2. Egress IP 表示 EgressPolicy 所使用的 EgressIP 设置：
    * 若在创建时定义了 `ipv4` 或 `ipv6` 地址，则从 EgressGateway 的 `.ippools` 中分配一个 IP 地址，若在 policy1 中，申请使用了 IP 地址 `10.6.1.21` 和 `fd00:1` ，然后创建 policy2 中，申请使用了 IP 地址 `10.6.1.21` 和 `fd00:2`，则会报错，此时 policy2 会分配失败，因为已分配的 `ipv4` 与 `ipv6` 地址会一一绑定，再次使用时，需要同时使用。如果只指定一者，会自动使用对应的另一者；
    * 若未定义 `ipv4` 或 `ipv6` 地址，且 `useNodeIP` 为 true 时，则使用所引用的 EgressGateway 匹配的 Node IP 作为 Egress 地址；
    * `egressGatewayName` 不能为空。
3. 支持使用节点 IP 作为 Egress IP（只允许选择一种）；
4. 默认为 `default` 模式，若未在创建时定义 `ipv4` 或 `ipv6` 地址，且 `useNodeIP` 为 `false` 时；
    * 为 `default` 时，则使用 EgressGateway 的 `.ippools.ipv4DefaultEIP/ipv6DefaultEIP` 值作为 EIP
    * 为 `rr` 时，则从 EgressGateway 的 `.ippools` 中随机分配一个未使用的 IP 地址（开启 IPv6 时，请求分配一个 IPv4 和 一个 IPv6 地址）。如果所有 IP 地址都被使用时，则 EIP 分配失败。
5. 以 Label 的方式选择需要应用 EgressPolicy 的 Pod；
6. 通过直接指定 Pod 的网段选择需要应用 EgressPolicy 的 Pod（4 和 5 不能同时使用）
7. 指定访问 Egress 的目标地址，若未指定目标地址，则以下策略将生效：对于那些目标地址不属于集群内部 CIDR 的请求，将全部转发到 Egress 节点。
8. 策略的优先级（未实现，保留字段）。
9. 该 EgressPolicy 所分配到的 EgressIP。
10. 该 EgressPolicy 的 EgressIP 所在的节点，同时也是该 EgressPolicy 的网关节点。
