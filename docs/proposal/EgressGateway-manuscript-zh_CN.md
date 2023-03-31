## 规则

对所有的规则分为三类，每个节点都要生效的、非网关节点生效、网关节点生效。

由于一个节点可能是某一个 EgressGateway 中某一个 EIP 的网关节点，但也可能是另外一个 或同一个 EgressGateway 中的 EIP 的非网关节点。所以网关节点与非网关节点都是以某一个 EIP 的角度来说的。没有绝对值。

但是规则生效时，如何判断呢？
在内存中维护一个全局的数组中保存所有的 EIPList，每个 EgressNode 记录本节点上生效的 EIP，如果 EIPList 中的 EIP 在本节点上没有，则本节点为该 EIP 的非网关节点。如果在节点上存在，则为该 EIP 的网关节点。由此判断需要生效哪些规则

### 标签

每个节点对应一个，全局唯一的标签。标签由前缀+唯一标识符生成。标签格式如下
NODE_MARK = 0x26 + value + 0000
value 为 16 位，支持的节点总数为 2^16

在下发 policy 规则时所打的标签，取决于该规则的 EIP 所生效的节点

### 在每个节点上都要生效的规则
1、各节点之间，隧道需要打通的规则就就不一一暂开了


2、将 policy 命中的流量，重新打标签。节点第一次变成网关节点时更新，或者节点 join 时做一次，后面不更新
```
    iptables -t mangle -N EGRESSGATEWAY-RESET-MARK
    iptables -t mangle -I FORWARD 1  -j EGRESSGATEWAY-RESET-MARK -m comment --comment "egress gateway: mark egress packet"

    iptables -t mangle -A EGRESSGATEWAY-RESET-MARK -m mark --mark $NODE_MARK/0x26000000 -j MARK --set-mark 0x12000000 -m comment --comment "egress gateway: change mark"
```

3、保持 policy 命中流量的标签。直接创建一次，不需要更新
```
    iptables -t filter -I FORWARD 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"
    iptables -t filter -I OUTPUT 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"
    iptables -t mangle -I POSTROUTING 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"
```

4、policy 命中的源 IP、目的 IP 的 ipset。实时更新
```
    IPSET_RULE_DEST_NAME=egress-dest-uuid

    ipset x $IPSET_RULE_DEST_NAME

    ipset create $IPSET_RULE_DEST_NAME hash:net

    ipset add $IPSET_RULE_DEST_NAME 10.6.105.150/32



    IPSET_RULE_SRC_NAME=egress-src-uuid

    ipset x $IPSET_RULE_SRC_NAME

    ipset create $IPSET_RULE_SRC_NAME hash:net

    ipset add $IPSET_RULE_SRC_NAME 172.29.234.173/32
```

5、聚合 policy 命中流量打标签的链。直接创建一次，不需要更新
```
    iptables -t mangle -N EGRESSGATEWAY-MARK-REQUEST
    iptables -t mangle -I PREROUTING 1  -j EGRESSGATEWAY-MARK-REQUEST -m comment --comment "egress gateway: mark egress packet"
```

6、聚合不需要做 SNAT 规则的链。直接创建一次，不需要更新
```
    # iptables -t nat -N EGRESSGATEWAY-NO-SNAT
    # iptables -t nat -I POSTROUTING 1  -j EGRESSGATEWAY-NO-SNAT -m comment --comment "egress gateway: no snat"

    iptables -t nat -A POSTROUTING 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: no snat"
```

7、聚合需要做 SNAT 规则的链。直接创建一次，不需要更新

```
    iptables -t nat -N EGRESSGATEWAY-SNAT-EIP
    # 需要在不需要 SNAT 的规则后面插入，才能保证该链在最前面
    iptables -t nat -I POSTROUTING 1  -j EGRESSGATEWAY-SNAT-EIP -m comment --comment "egress gateway: snat EIP"

```

### 在非网关节点上要生效的规则

1、policy 命中的流量打标签，保证能从隧道走。其中 NODE_MARK 的值根据 policy 对应的 EIP 所在节点决定。实时更新
```
    iptables -A EGRESSGATEWAY-MARK-REQUEST -t mangle -m conntrack --ctdir ORIGINAL \
    -m set --match-set $IPSET_RULE_DEST_NAME dst  \
    -m set --match-set $IPSET_RULE_SRC_NAME src  \
    -j MARK --set-mark $NODE_MARK   -m comment --comment "rule uuid: mark request packet"
```
    
2、策略路由规则
```
    ip rule add fwmark $NODE_MARK table $TABLE_NUM
```

### 在网关节点上要生效的规则

1、policy 命中的流量。出网关时做 SNAT。实时更新
```
    iptables -t nat -A EGRESSGATEWAY-SNAT-EIP -m set --match-set $IPSET_RULE_SRC_NAME src -m set --match-set $IPSET_RULE_DST_NAME dst -j SNAT --to-source $EIP
```

## CRD

### EgressGateway CRD

为集群级别的，但可以设置可以被所有的命名空间的 policy 使用，也可以被某些命名空间的 policy 使用。可以有创建多个 EgressGateway，通过 laebl Selector 来选择 EIP 生效的节点。而且可以设置支持的 EIP 范围，policy 选择 EgressGateway 时，可以通过注解指定 EIP，但 EIP 得要在 EgressGateway 的 EIP 范围内 。如果不指定 EIP，则在随机使用某一个未生效的。如果该 EgressGateway 的 EIP 都生效，就随机使用任意一个。
一个 EgressGateway 可以被多个 policy 选择，但 policy 只能选一个 EgressGateway。

EIP 在选中的节点上，选择策略为在最少 EIP 节点生效。（待确定）如果 eipRanges 为空，则使用节点的 IP，规则 SNAT 使用 MASQUERADE

```
apiVersion: egressgateway.spidernet.io/v1
kind: EgressGateway
metadata:
  name: "eg1"
spec:
  nodeSelector:
    matchLabels:
      egress: "true"        # 匹配标签为自定义
  useNodeIP: false          # bool 类型，是否使用节点 IP 作为 EIP，但怎么设计还未定
  eipRanges:
    ipv4:  # EIP 的取值范围，为数组类型，支持单个、IP段、CIDR
    - "10.6.167.100"
    - "10.6.167.110-10.6.167.120"
    - "10.6.168.0/24"
    ...
    ipv6:  # EIP 的 IPV6，跟 IPV4 一样
    - xxx(跟 ipv4 一样)
    ...
  isAllNamespace: false     # 是否所有租户可用
  namespaces:       # 为数组，可用的租户
  - ns1
  - ns2
  ...
status:
  nodeList:     # 表示 EIP 生效情况
  - node1:
    status: "ready" # 是否删除
    eips:    # 该节点上本 EgressGateway 生效的 EIP
      - eip: "10.6.167.100"
        policys:       # 如果 EIP 没有 policy 引用，则回收该 EIP，做相关的规则清理
        - "policy1"
        - "policy2"
      - eip: "10.6.167.110"
        policy: 
        - "policy3"
        - "policy4"

```

### EgressGatewayPolicy CRD

为租户级，需要绑定一个符合条件的 EgressGateway（只能选一个）。可以通过 annotation 指定 EIP，有且只能有一个 EIP。创建的同时，会创建一个附属 EgressEndpointSlice 对象，EndPointSlice 对象中聚合了关于该 policy 的信息。当 policy 命中的 Pod 过多时，会创建更多的 EgressEndpointSlice。

```
apiVersion: egressgateway.spidernet.io/v1
kind: EgressGatewayPolicy
metadata:
  name: "egPolicy1"
  namespace: "default"  # 为租户级，属于哪个租户
  annotations:
    eipIPV4: "xxx"      # 可指定 IPV4 的 EIP，但得在 EgressGateway 的 eipRanges 内
    eipIPV6: "xxx"      # 可指定 IPV6 的 EIP，但得在 EgressGateway 的 eipRanges 内
spec:
  appliedTo:
    podSelector:        # 可通过标签来选择适用的 Pod
      matchLabels:
        app: "shopping"
    podSubnet:          # 与 podSelector 只能二选一，直接通过 IP 地址来选择适用的 Pod
      ipv4PodSubnet:    # 为数组
      - "172.28.30.1"   # 支持单个的 IP
      - "172.28.30.10-172.28.30.15" # 支持 IP 段
      - "172.29.16.0/24"    # 支持 CIDR 的格式
      ...
      ipv6PodSubnet:    # 同 IPV4
      - "xxx"
      ...
  destSubnet:           # 目的范围。支持方式如 podSubnet
    ipv4:
    - "xxx"
    ipv6:
    - "xxx"
  egressGateway: "eg1"  # 只能选择一个符合要求的 EgressGateway 进行绑定
  
```

### EgressEndpointSlice

不可手动创建，是 policy 的附属资源，用以聚合关于 policy 的信息。每个 EgressEndpointSlice 中的 EndPoint 个数默认不超过 100，最大值可以进行设置

```
apiVersion: egressgateway.spidernet.io/v1
kind: EgressEndpointSlice
metadata:
  name: "egPolicy1"             # 跟 policy 名称一致
  namespaces: "default"         # 跟 policy 一样，也是租户级
  ownerReferences:              # 在创建的时候设置
  - apiVersion: egressgateway.spidernet.io/v1
    blockOwnerDeletion: true
    controller: false
    kind: EgressGatewayPolicy
    name: "egPolicy1"
    uid: xxx
status:
  eip: "10.6.167.100"   # 记录该 policy 使用的 EIP
  eipNode: "node1"      # 使用的 EIP 所在的节点
  mark: "0x26xx0000"    # 标签值
  destSubnet:             # 同 policy 的 destSubnet
  - "10.6.1.0/24"
  endPoints:             # 记录命中的 Pod 信息，ipset 的内容来自于此
    - podName: "xxx"      # 仅仅展示作用
      ipv4: "172.29.30.123" # Pod IPV4
      ipv6: "xxx"         # Pod 的IPV6
      node: "node1"       # Pod 所在节点，决定了在该节点上是否要下发非网关节点的规则
  ...
  

```

### EgressNode

不可手动创建。每个 Node 都会创建一个对应的 EgressNode，主要记录了本节点隧道及该节点上生效的 EIP 信息。

```
apiVersion: egressgateway.spidernet.io/v1
kind: EgressNode
metadata:
  name: "node1"
spec:
status:
  tunnel:       # 存放隧道信息
    phase: "Succeeded"              # 表示 EgressNode 的状态，’Succeeded’ 隧道IP已分配，且隧道已建成，’Pending’ 等待分配IP，’Init’ 分配隧道 IP 成功，’Failed’ 隧道 IP 分配失败
    vxlanIPv4IP: "172.31.0.10/16"   # 隧道 IPV4 地址
    vxlanIPv6IP: "fe80::/64"        # 隧道 IPV6 地址
    tunnelMac: "xx:xx:xx:xx:xx"     # 隧道 Mac 地址
    physicalInterface: "eth1"       # 隧道父网卡
    physicalInterfaceIPv4: ""       # 父网卡 IPV4 地址
    physicalInterfaceIPv6: ""       # 父网卡 IPV6 地址
  eips:     # 存放在该节点上生效的的所有 EIP 信息，生效规则时，由该字段决定本节点要生效哪些规则
    ipv4:   # 本节点生效的所有 ipv4 EIP
    - "10.6.167.100"
    - "10.6.167.110"
    ...
    ipv6:   # 本节点生效的所有 ipv6 EIP
    - "fd::11"
    - "fd::10"
    ...
  ...
  
```

## controller

### EgressNode

### Init

#### EgressNode Event
*   Del：
*   Other：

#### Node Event
*   Del：
*   Other：

### EgressGateway
*   Del：
*   Other：

#### EgressGateway Event
*   Del：
*   Other：

#### EgressGatewayPolicy Event
*   Del：
*   Other：

#### Node Event
*   Del：
*   Other：

### EgressGatewayPolicy

#### EgressGatewayPolicy Event
*   Del：
*   Other：


### EgressEndpointSlice

#### EgressEndpointSlice Event
*   Del：
*   Other：

#### EgressGatewayPolicy Event
*   Del：
*   Other：

#### Pod Event
*   Del：
*   Other：

## agent

### EgressNode
*   Del：
*   Other：

### EgressEndpointSlice
*   Del：
*   Other：


## 其他

1、dummy 网卡及 EIP：每个节点只有一个名为 eg-eip 的 dummy 网卡，所有的 EIP 都生效在该节点上
```
    # 创建 dummy 网卡
    ip link add eip type  dummy
    ip link set eip up

    # 设置 EIP
    ip addr add 10.6.168.100  dev eip
```

2、由于 EIP 是生效在 dummy 网卡上的，所有需要配置 ARP 代答，
```
    sysctl -w net.ipv4.conf.all.arp_ignore=0

    # 所有的物理网卡都需要设置代答，不确定从那种网卡出去
    sysctl -w net.ipv4.conf.xxx.arp_ignore=0
```

3、mangle-FORWARD match 重新打标签
因 为NODE_MARK = 0x26 + value + 0000，所以匹配时只要匹配前面16 位
```
iptables -t mangle -I FORWARD 1 -m mark --mark 0x26000000/0x26000000 -j MARK --set-mark 0x12000000 -m comment --comment "egress gateway: change mark"

```

4、更新 ipset 内容，CRD 中聚合了最新的 IP 内容，可以先创建临时 ipset 再通过 swap 进行交换，大量简化 ipset 操作，提高效率
```
ipset create egress-dst-v4-xxx-tmp 
ipset add egress-dst-v4-xxx-tmp <$new_ip_range>
ipset swap egress-dst-v4-xxx egress-dst-v4-xxx-tmp 
```