EgressPolicy CRD 用于指定哪些 Pod 访问哪些目标 CIDR 时走 Egress 策略，以及 Egress 所使用的 IP 地址。租户级资源。

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
    allocatorPolicy: default  
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
status:
  eip:                        
    ipv4: 172.18.1.2
    ipv6: fc00:f853:ccd::9
  node: egressgateway-worker  
```

## 定义

### metadata

| 字段        | 描述                   | 数据类型 | 验证 |
|-----------|----------------------|------|----|
| namespace | EgressPolicy 资源的命名空间 | 字符串  | 必填 |
| name      | EgressPolicy 资源的名称   | 字符串  | 必填 |

### spec

| 字段                | 描述                                                                                                      | 数据类型                    | 验证 | 可选值      | 默认值 |
|-------------------|---------------------------------------------------------------------------------------------------------|-------------------------|----|----------|-----|
| egressGatewayName | 使用的 EgressGateway 的引用                                                                                   | 字符串                     | 必填 |          |     |
| egressIP          | 出口 IP 设置的配置                                                                                             | [egressIP](#egressIP)   | 可选 |          |     |
| appliedTo         | 应将 EgressPolicy 应用于哪些 Pods 的选择器                                                                         | [appliedTo](#appliedTo) | 必填 |          |     |
| destSubnet        | 访问该列表的子网时使用 Egress IP，如果安装时开启了 `feature.clusterCIDR.autoDetect`，destSubnet 没设置时，则访问集群外网络自动使用 Egress IP。 | 字符串数组                   | 可选 | CIDR 表示法 |     |
| priority          | 策略的优先级                                                                                                  | 整数                      | 可选 |          |     |

#### egressIP

| 字段        | 描述                                    | 数据类型   | 验证 | 可选值        | 默认值   |
|-----------|---------------------------------------|--------|----|------------|-------|
| ipv4      | 如果定义，则使用特定的 IPv4 地址                   | string | 可选 | 有效的 IPv4   |       |
| ipv6      | 如果定义，则使用特定的 IPv6 地址                   | string | 可选 | 有效的 IPv6   |       |
| useNodeIP | 当没有定义特定的 IP 地址时，是否使用节点 IP 作为出口 IP 的标志 | bool   | 可选 | true/false | false |

#### appliedTo

| 字段          | 描述                                | 数据类型              | 验证 | 可选值  | 默认值 |
|-------------|-----------------------------------|-------------------|----|------|-----|
| podSelector | 通过 Selector 匹配实施 Egress 策略 Pod    | map[string]string | 可选 |      |     |
| podSubnet   | 通过 Subnet 匹配实施 Egress 策略 Pod（未实现） | []string          | 可选 | CIDR |     |
