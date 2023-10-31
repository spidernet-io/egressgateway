EgressGateway 由控制面和数据面 2 部分组成，控制面由 4 个控制循环组成，数据面由 3 个控制循环组成。控制面以 Deployment 方式部署，支持多副本高可用，数据面以 DaemonSet 的方式部署。控制循环具体如下图：

![arch](../proposal/03-egress-ip/arch.png)


## Controller

### EgressTunnel reconcile loop (a) 

#### 初始化

1. 从 ConfigMap 配置文件中获取双栈开启情况及对应的隧道 CIDR
2. 通过节点名称根据算法生成唯一的标签值
3. 会检查 Node 是否有对应的 EgressTunnel，没有的话就创建对应的 EgressTunnel，且状态设置为 `Pending`。有隧道 IP 则将 IP 与节点绑定，绑定前会检查 IP 是否合法，不合法则将状态设置为 `Pending`

#### EgressTunnel Event

- Del：先释放隧道 IP，再删除。如果 EgressTunnel 对应的节点还存在，重新创建 EgressTunnel
- Other：
  - phase != `Init` || phase != `Ready`：则分配 IP，分配成功将状态设置为 `Init`，分配失败将状态设置为 `Failed`。这里是全局唯一会分配隧道 IP 的地方
  - mark != algorithm(NodeName)：该字段禁止修改，直接报错返回

#### Node Event

- Del：删除对应的 EgressTunnel
- Other：
  - 节点对应的 EgressTunnel 不存在，则创建 EgressTunnel
  - 无隧道 IP，设置 phase 为 `Pending`
  - 有隧道 IP，校验隧道是否合法，不合法则设置 phase 为 `Pending`
  - 隧道 IP 合法，校验 IP 是否分配给本节点，不是则设置 phase 为 `Pending`
  - 隧道 IP 是分配给本节点，phase 状态不为 `Ready` 则设置 phase 为 `Init`

### EgressGateway reconcile loop (b)

#### EgressGateway Event

- Del：
  * Webhook 判断是否还被其他 Policy 引用，如果存在则不允许删除。
  * 通过了 Webhook 的校验说明没有被引用，所以的规则也被清理，则可以直接删除。

- Other：
  * EIP 减少，如果 EIP 被引用，禁止修改。分配 IPV4 与 IPV6 时，要求一一对应，所以两者的个数需要一致。
  * 如果 nodeSelector 被修改，从 status 获取旧的 Node 信息，与最新的 Node 进行对比。将删除节点上的 EIP 重新分配到新的 Node 上。更新对应 EgressTunnel 中的 EIP 信息。

#### EgressPolicy Event

- Del：列出 EgressPolicy 找到被引用的 EgressGateway，再对 EgressPolicy 与 EgressGateway 解绑。解绑需要做的事情有，找到对应的 EIP 信息。如果使用了 EIP，则判断是否需要回收 EIP。如果此时 EIP 已经没有 policy 使用，则回收 EIP，更新自身及 EgressTunnel 的 EIP 信息。
- Other：
  * EgressPolicy 不能修改绑定的 EgressGateway。如果允许修改，则列出 EgressGateway 找到原先绑定的 EgressGateway，进行解绑。再对新的进行绑定。
  * 新增 EgressPolicy，则将 EgressPolicy 与 EgressGateway 进行绑定，绑定中，判断是否需要分配 EIP。

#### Node Event

- Del：列出 EgressGateway 挑选出在该节点生效的 EIP，将这些 EIP 重新分配到新的节点上。更新 EgressGateway 的 eip.policy。
- Other：
  * NoReady 事件时，相当于触发删除事件。
  * 标签修改，通过遍历 EgressGateway 所有信息，是否涉及 nodeSelector。如果旧标签不涉及 EgressPolicy，则不做任何处理。如果有涉及，相当于触发了删除事件。如果新的标签符合 EgressGateway 条件，则更新对应的 EgressGateway 的 status 信息。

### EgressPolicy 选网关节点及 EIP 分配逻辑

一个 EgressPolicy 会根据选网关节点的策略，选择一个节点作为网关节点。然后根据是否使用 EIP，来决定是否分配 EIP。分配的 EIP 将绑定到所选的网关节点上。

分配逻辑都是以单个 EgressGateway 为对象，而不是所有的 EgressGateway。

#### EgressPolicy 选网关节点的模式

- 平均选择：当需要选择网关节点时，选择作为网关节点最少的一个节点。
- 最少节点选择：尽量选同一个节点作为网关节点。
- 限度选择：一个节点最多只能成为几个 EgressPolicy 的网关节点，限度可以设置，默认为 5。没有达到限度前，则优选选择该节点，达到限度就先选其他的节点，如果都达到了限度，则再随机选择。


#### EIP 分配逻辑

- 随机分配：在所有的 EIP 中随机选择一个，不管该 EIP 是否已经分配
- 优先使用未分配的 EIP：先使用未分配的 EIP，如果都使用了则再随机分配一个已使用的 EIP
- 限度选择：一个 EIP 最多只能被几个 EgressPolicy 使用，限度可以设置，默认为 5，没有达到限度前，则先分配该 EIP，达到限度则选其他的 EIP。都达到限度则随机分配。


#### EIP 回收逻辑

当一个 EIP 没有被使用时，则回收该 EIP，回收就是在 `eips` 中将该 EIP 字段删除。


### EgressClusterInfo reconcile loop (d)

#### Node Event

- Create：node 创建时，将 node 的 ip 自动添加到 egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP` 中。
- Update：node ip 有更新时，将 node 的 ip 自动更新到 egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP` 中。
- Delete：node 被删除时，将 node 的 ip 从 egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP` 中删除。

#### Calico IPPool Event

当 egressgateway 配置文件的 `egressIgnoreCIDR.autoDetect.podCIDR` 为 "calico" 时，监听 Calico 的 IPPool Event。
- Create：Calico IPPool 创建时，将 IPPool CIDR 自动添加到 EgressClusterInfo CR `status.egressIgnoreCIDR.podCIDR` 中。
- Update：calico IPPool 有更新时，将 IPPool CIDR 自动更新到 EgressClusterInfo CR `status.egressIgnoreCIDR.podCIDR` 中。
- Delete：calico IPPool 被删除时，将 IPPool CIDR 从 EgressClusterInfo CR `status.egressIgnoreCIDR.podCIDR` 中删除。

#### 配置文件

修改配置文件，增加如下配置：

```yaml
feature:
  egressIgnoreCIDR:
    autoDetect:
      podCIDR: ""      # 1
      clusterIP: true  # 2
      nodeIP: true     # 3
    custom:
      - "10.6.1.0/24"
```

1. `podCIDR`，目前支持 `calico`、`k8s`。默认为 `k8s`。
2. `clusterIP`，支持设置为 Service CIDR 自动检测。
3. `nodeIP`，支持设置为 Node IP 自动检测。