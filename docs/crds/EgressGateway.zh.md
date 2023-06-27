## 简介

用于选择一组节点作为 Egress 网关节点，Egress IP 可以在该范围浮动。集群级资源。

## CRD

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
    policy: "doing"   # 8
status:                         # 9
  nodeList:                     # 10
    - name: "node1"             # 11
      status: "Ready"           # 12
      epis:                     # 13
        - ipv4: "10.6.1.55"     # 14
          ipv6: "fd00::55"      # 15
          policies:             # 16
            - name: "app"         # 17
              namespace: "default"  # 18
```

1. ippools: 设置 Egress IP 的范围；
2. ipv4([]string): EIP 的 IPV4 内容，支持设置单个 IP `10.6.0.1` ，和段 `10.6.0.1-10.6.0.10 ` ， CIDR `10.6.0.1/26` 共3 种方式；
3. ipv6([]string): EIP 的 IPV6 内容，如果开启双栈要求，IPv4 的数量和 IPv6 的数量要求一致，支持的格式与 IPV4 一致；
4. ipv4DefaultEIP(string): 默认使用的 IPV4 EIP，如果 egp 未指定 EIP，且 EIP 分配的策略为 'default'，则该 egp 分配到的 EIP 就是 ipv4DefaultEIP；
5. ipv6DefaultEIP(string): 默认使用的 IPV6 EIP，规则如 ipv4DefaultEIP；
6. nodeSelector: 设置网关节点的匹配条件及策略
7. selector: 设置节点的匹配内容
8. policy(string): egp 选择网关节点的策略，目前只支持平均选择，其他策略待实现。
9. status: 展示其所选网关节点、EIP 、被 policy 引用情况
10. nodeList([]EgressIPStatus):
11. name(string): 网关节点的名称
12. status(string): 网关节点的状态
13. eips([]Eips): 该网关节点上生效的 EIP 相关信息
14. ipv4(string): IPV4 EIP，如果 egp、egcp 使用节点 IP，则该字段为空
15. ipv6(string): IPV6 EIP，双栈情况下，IPV6 与 IPV4 是一一对应的
16. policies([]string): 以该节点作为网关节点的 egp、egcp 集合
17. name(string): egp、egcp 的名称
18. namespace(string): egp 的 NS，如果是 egcp 则为空

## 代码设计

### 初始化

### Controller
#### EgressGateway Event
- Del：
    * webhook 判断是否还被其他 Policy 引用，如果存在则不允许删除。
    * 通过了 webhook 的校验说明没有被引用，所以的规则也被清理，则可以直接删除

- Other：
    * EIP 减少，如果 EIP 被引用，禁止修改。分配 IPV4 与 IPV6 时，要求一一对应，所以两者的个数需要一致
    * 如果 nodeSelector 被修改，从 status 获取旧的 Node 信息，与最新的 Node 进行对比。将删除节点上的 EIP 重新分配到新的 Node 上。更新对应 EgressNode 中的 EIP 信息。


#### EgressGatewayPolicy Event
- Del：list EgressEndpointSlice 找到被引用的 EgressGateway，再对 policy 与 EgressGateway 解绑。解绑需要做的事情有，找到对应的 EIP 信息。如果使用了 EIP，则判断是否需要回收 EIP。如果此时 EIP 已经没有 policy 使用，则回收 EIP，更新自身及 EgressNode 的 EIP 信息
- Other：
    * 暂定 policy 不能修改绑定的 EgressGateway。如果允许修改，则 list EgressGateway 找到原先绑定的 EgressGateway，进行解绑。再对新的进行绑定。
    * 新增 policy，则将 policy 与 EgressGateway 进行绑定，绑定中，判断是否需要分配 EIP

#### Node Event
- Del：list EgressGateway 挑选出在该节点生效的 EIP，将这些 EIP 重新分配到新的节点上。更新 EgressGateway 的 eip.policy
- Other：
    * NoReady 事件时，相当于触发删除事件
    * 标签修改，通过遍历 EgressGateway 所有信息，是否涉及 nodeSelector。如果旧标签不涉及 EgressPolicy，则不做任何处理。如果有涉及，相当于触发了删除事件。如果新的标签符合 EgressGateway 条件，则更新对应的 EgressGateway 的 status 信息


### agent
无

## 其他
### policy 选网关节点及 EIP 分配逻辑
一个 policy 会根据选网关节点的策略，选择一个节点作为网关节点。然后根据是否使用 EIP，来决定是否分配 EIP。分配的 EIP 将绑定到所选的网关节点上

分配逻辑都是以单个 EgressGateway 为对象，而不是所有的 EgressGateway。

#### policy 选网关节点的模式
- 平均选择：当需要选择网关节点时，选择作为网关节点最少的一个节点。
- 最少节点选择：尽量选同一个节点作为网关节点
- 限度选择：一个节点最多只能成为几个 policy 的网关节点，限度可以设置，默认为 5。没有达到限度前，则优选选择该节点，达到限度就先选其他的节点，如果都达到了限度，则再随机选择


#### EIP 分配逻辑
- 随机分配：在所有的 EIP 中随机选择一个，不管该 EIP 是否已经分配
- 优先使用未分配的 EIP：先使用未分配的 EIP，如果都使用了则再随机分配一个已使用的 EIP
- 限度选择：一个 EIP 最多只能被几个 policy 使用，限度可以设置，默认为 5，没有达到限度前，则先分配该 EIP，达到限度则选其他的 EIP。都达到限度则随机分配。


#### EIP 回收逻辑
当一个 EIP 没有被使用时，则回收该 EIP，回收就是在 eips 中将该 EIP 字段删除