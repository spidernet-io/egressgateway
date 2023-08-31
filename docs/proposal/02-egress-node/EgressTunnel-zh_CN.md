## EgressTunnel CRD

```yaml
apiVersion: egressgateway.spidernet.io/v1
kind: EgressTunnel
metadata:
  name: "node1"
spec:
status:
  phase: "Ready"
  vxlanIPv4IP: "172.31.0.10/16"
  vxlanIPv6IP: "fe80::/64"
  tunnelMac: "xx:xx:xx:xx:xx"
  physicalInterface: "eth1"
  physicalInterfaceIPv4: ""
  physicalInterfaceIPv6: ""
```

用以存储各节点的隧道的信息，通过监控节点来生成

字段说明
* status
    * `phase` 表示 EgressTunnel  的状态，’Ready’ 隧道IP已分配，且隧道已建成，’Pending’ 等待分配IP，’Init’ 分配隧道 IP 成功，’Failed’ 隧道 IP 分配失败
    * `vxlanIPv4IP` 隧道 IPV4 地址
    * `vxlanIPv6IP` 隧道 IPV6 地址
    * `tunnelMac` 隧道 Mac 地址
    * `physicalInterface` 隧道父网卡
    * `physicalInterfaceIPv4` 父网卡 IPV4 地址
    * `physicalInterfaceIPv6` 父网卡 IPV6 地址


## controller 实现

### 初始化
1. 从 CM中获取 IPv4、IPv6 及对应的 CIDR
2. 会检查node 是否有对应的 EgressTunnel，没有的话就创建对应的EgressTunnel，且状态设置为 “pending”。有隧道 IP 则将 IP 与节点绑定，绑定前会检查 IP 是否合法，不合法则将状态设置为 “Pending”

### 节点事件：
- 删除事件：删除对应的 EgressTunnel
- 其他事件：如果没有对应的 EgressTunnel，则创建 EgressTunnel
- 其他事件：如果有对应的 EgressTunnel，则对EgressTunnel进行校验。校验逻辑如下：

- -     无隧道IP，将状态置为 “Pending”
        如果有隧道IP，判断是否合法，不合法，就将状态置为 “Pending”
        如果合法，校验 IP 是否已分配，如果已分配，且分配给其他节点了，则将状态置为 “Pending”
        未分配给其他节点，就分配给本 “EgressTunnel”，将状态设置为 “Init”
        如果已分配，且就是分配给本节点的，则将状态设置为 “Init”

### EgressTunnel事件：
- 删除事件：先释放IP。如果 EgressTunnel 对应的节点存在，则释放IP，重新创建 EgressTunnel。
- 其他事件：如果 EgressTunnel 状态为 “Init” 或 者“Ready” 时，不做任何处理。如果不是，则分配 IP，分配成功将状态设置为 “Init”，分配失败将状态设置为 “Failed”。这里是全局唯一会分配隧道 IP 的地方


## 分配隧道 IP
- controller 启动时，从 config 中拿到隧道 IP CRID，并在内存中维护一个 map 记录 IP 是否被分配
- 隧道 IP 采用中心式，所以使用串行方式分配IP，在未分配的 IP 中随机分配
- 分配前，检测隧道 IP 冲突，不冲突再分配（实现方式待定）

## 其他
- controller 启动时，对所的 CRD 的 IP 进行校验，且最终状态设置为 “Init” 或 “Failed” （此项待商榷）
- agent 监测到对应的 CRD 中 phase 字段为 “Init” 时，创建相应的隧道及路由，创建成功更新为 “Ready” 状态。失败则不更新
- Mac地址格式：根据节点名称，经过 SHA1 算法生成，所以每个节点的 Mac 地址是固定的


