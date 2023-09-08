# underlay CNI supports

## 动机

EgressGateway 在 Underlay CNI 环境下不适用

## 目标

EgressGateway 支持纳管 Underlay CNI 环境下的流量

## 需要解决的问题
如图所示，Underlay 访问外部 Server 来回的 datapath 为："Process <-> A <-> B <-> Server"。

<img src="./underlay_datapath.png" width="70%"></img>

EgressGateway 的规则根本不生效，要想将 Underlay 的流量进行纳管，则需要解决两件事，将流量劫持到 Pod 的所在的主机上，及当应答的流量到达 Pod 所在主机时，避免路由不对称报文被丢弃


## 将匹配 EgressGateway  policy 的 Pod 报文劫持到它所在的主机上
要解决该问题需要做两件事：
1、打通 Pod 与主机之间的通道
2、使符合 policy 的报文通过通道转发到主机

事请一，overlay + underlay 时可以借助 overlay 的网卡实现。单一的 underlay 在 Pod 创建时，使用 veth pair 一端在主机，另一端插在 Pod 网络命名空间内。要完成上述事情，在 Pod 创建时 kubelet 调用 CNI 可以完成，spiderpool 插件刚好可以完成，或者是特权 agent Pod 来完成（路子有一点野）。

事情二，可以通过路由、iptables 等方式将匹配的流量通过前面的 veth pair 转发到主机上。可以通过设置 spiderpool crd  spidercoordinators.spec.hijackCIDR 来完成，spiderpool 会设置相应的路由。也可以通过 sidecar 设置相应的规则。


### 发送 datapath

如图所示，通过新增 veth pair，并通过路由将流量通过 veth 转发到主机上，此时的 datapath 与 overlay 其实是一样的。

<img src="./underlay_send_datapath.png" width="70%"></img>

### 应答 datapath

如图所示，返回的 datapath 为 "Server->D->C->B->E->Process"

<img src="./underlay_error_reply_datapath.png" width="70%"></img>

- 报文经过 D 段 datapath 到达 EgressGateway 时的 srcIP=ServerIP、dstIP=EIP
- C 段 datapath 会查询连接跟踪表，会将报文进行 NAT，srcIP=ServerIP、dstIP=PodIP
- B 段 datapath，因为是 underlay 环境，所以 EgressNode 可以直接与 Pod 通信，经过交换机或者路由器，报文直达 Pod，可以看到此时并没有经过主机的网络命名空间。这样就会出现路由不对称问题

## 返回的报文路由不对称问题

在这里梳理一下为什么会出现该问题：
- 三次握手的第一个报文 SYN，从 Pod 到达主机，经过主机的网络协议栈，封包后转发出去
- 三次握手的第二个报文 SYN+ACK，报文到达 EgressNode 节点，然后直接到 Pod 所在节点的物理网卡，再直接到达 Pod，没有经过主机的网络协议栈，导致来回路由不一致
- 三次握手的第三个报文 ACK，从 Pod 到达主机，因为没有收到 SYN+ACK 报文，就直接收到 ACK 报文，此时会认为 ACK 报文是无效的包，从而被 kube-proxy 的一条 DROP 规则命中丢弃，导致三次握手失败 

``` 
Chain KUBE-FORWARD (1 references)
 pkts bytes target     prot opt in     out     source               destination
    0     0 DROP       all  --  *      *       0.0.0.0/0            0.0.0.0/0            ctstate INVALID
```

要解决该问题，需要保证回包也要经过 Pod 所在主机的网络命名空间，才能避免上述问题。可以在出口网关节点上通过设置相应的规则，将回包通过 EgressGateway 的隧道回到业务 Pod 所在节点，而不是直接返回给 Pod。实现方式如下：

```
打上出口网关节点上，从隧道过来的新连接打 Mark，标记这是 EgressGateway 命中的报文
iptables -t mangle -A PREROUTING -i egress.vxlan  -m conntrack --ctstate NEW -j MARK --set-mark 0x27

将 mark 添加到连接跟踪表，以便回包时进行恢复
iptables -t mangle -A PREROUTING -m mark --mark 0x27 -j CONNMARK --save-mark


ESTABLISHED 的连接，报文需要根据连接跟踪表记录的内容进行恢复 Mark
iptables -t mangle -A PREROUTING  -m conntrack --ctstate ESTABLISHED -j CONNMARK --restore-mark


添加路由使回包走隧道，每个都要添加一条路由
ip rule add from all fwmark 0x27 lookup 600
ip r add <Pod IP> via <Pod Node> dev egress.vxlan t 600

清除内层包的的 Mark，避免干扰外层的报文
iptables -t mangle -A  POSTROUTING -m mark --mark 0x27 -j MARK --set-mark 0x00
 
```

如图所示，经过上面的规则，新的应答 datapath 为 "Server->D->C->B->A->Process"

<img src="./underlay_reply_datapath.png" width="70%"></img>

最大的不同就是，从网关节点到 Pod 所在节点，是通过 EgressGateway 隧道，报文到达 Pod 所在节点后，通过路由指定从 veth pair 转发给 Pod，spiderpool 在前面给 Pod 创建 veth pair 的同时，会下发对应的路由，或者可以通过 agent 下发相应的路由规则。因为经过了主机的网络协议栈。从而规避了路由不对称问题

## 总结

除了在网关节点上需要下发规则，使应答报文通过 EgressGateway 返回到 Pod 所在节点外，其他需要完成的事情，通过配置 spiderpool 都可以帮忙完成。