EgressPolicy CRD 用于指定哪些 Pod 访问哪些目标 CIDR 时走 Egress 策略，以及 Egress 所使用的 IP 地址。租户级资源。

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

1. 选择 EgressPolicy 引用的 EgressGateway：
2. Egress IP 表示 EgressPolicy 所使用的 EgressIP 设置：
    * 若在创建时定义了 `ipv4` 或 `ipv6` 地址，则从 EgressGateway 的 `.ippools` 中分配一个 IP 地址，若在 policy1 中，申请使用了 IP 地址 `10.6.1.21` 和 `fd00:1` ，然后创建 policy2 中，申请使用了 IP 地址 `10.6.1.21` 和 `fd00:2`，则会报错，此时 policy2 会分配失败；
    * 若未定义 `ipv4` 或 `ipv6` 地址，且 `useNodeIP` 为 true 时，则使用所引用 EgressGateway 的匹配中的 Node 的 IP 作为 Egress 地址；
    * 若未在创建时定义 `ipv4` 或 `ipv6` 地址，且 `useNodeIP` 为 `false` 时；
        * 则自动从 EgressGateway 的 `.ranges` 中分配一个 IP 地址（开启 IPv6 时，请求分配一个 IPv4 和 一个 IPv6 地址）。
    * `egressGatewayName` 不能为空。
3. 支持使用节点 IP 作为 Egress IP（只允许选择一种）；
4. 选择需要应用 EgressPolicy 的 Pod；
   a. 以 Label 的方式进行选择
   b. 直接指定 Pod 的网段 （a 和 b 不能同时使用）
5. 指定访问 Egress 的目标地址，若未指定目标地址，则生效的策略位目标地址非集群内 CIDR 时，全部转发到 Egress 节点；
6. 策略的优先级。
