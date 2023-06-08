# EgressNode

## 简介

主要用于记录跨节点通信的隧道网卡信息。集群级资源，与 Kubernetes Node 资源名称一一对应。

## CRD

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressNode
metadata:
   name: "node1"
status:
   tunnel:
      ipv4: "192.200.222.157"  # 1
      ipv6: "fd01::f2"         # 2        
      mac: "66:50:85:cb:b2:bf" # 3
      parent:
         name: "ens160"        # 4
         ipv4: "10.6.1.21/16"  # 5
         ipv6: "fd00::21/112"  # 6
   phase: "Succeeded"          # 7
   mark: "0x26000000"          # 8
```

1. 隧道 IPv4 地址
2. 隧道 IPv6 地址
3. 隧道 MAC 地址
4. 隧道父网卡
5. 隧道父网卡 IPv4 地址
6. 隧道父网卡 IPv6 地址
7. 当前隧道就绪阶段，`Succeeded` 隧道IP已分配，且隧道已建成，`Pending` 等待分配IP，`Init` 分配隧道 IP 成功，`Failed` 隧道 IP 分配失败
8. mark 值，此为新增此段，创建时生成。每个节点对应一个，全局唯一的标签。标签由前缀 + 唯一标识符生成。标签格式如下 `NODE_MARK = 0x26 + value + 0000`，`value` 为 16 位，支持的节点总数为 `2^16`。在下发 policy 规则时所打的标签，取决于该规则的网关节点。

## 代码设计

### Controller

#### 初始化

1. 从 CM 中获取双栈开启情况及对应的隧道 CIDR
2. 通过节点名称根据算法生成唯一的标签值
3. 会检查 node 是否有对应的 EgressNode，没有的话就创建对应的EgressNode，且状态设置为 `Pending`。有隧道 IP 则将 IP 与节点绑定，绑定前会检查 IP 是否合法，不合法则将状态设置为 `Pending`

#### EgressNode Event

- Del：先释放隧道 IP，再删除。如果 EgressNode 对应的节点还存在，重新创建 EgressNode
- Other：
  - phase != `Init` || phase != `Succeeded`：则分配 IP，分配成功将状态设置为 `Init`，分配失败将状态设置为 `Failed`。这里是全局唯一会分配隧道 IP 的地方
  - mark != algorithm(NodeName)：该字段禁止修改，直接报错返回
  
#### Node Event
- Del：删除对应的 EgressNode
- Other：
  - 节点对应的 EgressNode 不存在，则创建 EgressNode
  - 无隧道 IP，设置 phase == `Pending`
  - 有隧道 IP，校验隧道是否合法，不合法则设置 phase == `Pending`
  - 隧道 IP 合法，校验 IP 是否分配给本节点，不是则设置 phase == `Pending`。
  - 隧道 IP 是分配给本节点，phase != `Succeeded` 则设置 phase == `Init`

### Agent
#### 初始化


## 其他
