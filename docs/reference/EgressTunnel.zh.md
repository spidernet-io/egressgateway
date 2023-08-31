EgressTunnel CRD 用于记录跨节点通信的隧道网卡信息。这是一个集群级资源，它与 Kubernetes Node 资源名称一一对应。

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressTunnel
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
   phase: "Ready"              # 7
   mark: "0x26000000"          # 8
```

1. 隧道 IPv4 地址
2. 隧道 IPv6 地址
3. 隧道 MAC 地址
4. 隧道父网卡
5. 隧道父网卡 IPv4 地址
6. 隧道父网卡 IPv6 地址
7. 当前隧道状态
    - `Pending`：等待分配 IP
    - `Init`：分配隧道 IP 成功
    - `Ready`：隧道 IP 已分配，且隧道已建成
    - `Failed`：隧道 IP 分配失败
8. 数据包 mark 值，每个节点对应一个。例如节点 A 的有 Egress 流量需要转发到网关节点 B，会对 A 节点的流量打 mark 进行标记。