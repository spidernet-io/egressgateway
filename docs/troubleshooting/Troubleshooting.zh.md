# 排障指南

## 1. 确保组件运行

```bash
kubectl get pods -n kube-system | grep -i egress
```

运行该命令检查所有 egress 的 Pod 是否都在 Running 状态。如果有 Pod 处于非 Running 状态，可能是由于以下原因：

- Image Pull Back Off：请检查 Pod 的描述信息，确认是否存在 ImagePullBackOff 错误。
- Pod Restart：检查 Pod 的重启次数，确认是否有 Pod 频繁重启的情况。

## 2. 检查 Calico CNI 配置

如果使用 Calico CNI，如果是使用的是其他 CNI，请忽略此步骤。确保 Calico 的 chainInsertMode 处于 Append 模式。可以通过以下命令检查：

```bash
kubectl get FelixConfiguration -n kube-system default -o yaml | grep chainInsertMode
```

如果输出结果不是 `Append` 或者没有该字段，可以通过以下命令修改：

```bash
kubectl patch FelixConfiguration -n kube-system default --type='json' -p='[{"op": "replace", "path": "/spec/chainInsertMode", "value": "Append"}]'
```

这个决定了 EgressGateway 的 iptables 规则是否会被 Calico 覆盖。你可以在主机上直接执行下面的命令来验证：

```bash
iptables -t mangle -nvL PREROUTING
```

如果第一条是 `cali-*` 开头的规则，说明 Calico 的规则覆盖了 EgressGateway 的规则。

通常应当看到如下规则位于最前面：

```bash
$ iptables -t mangle -nvL PREROUTING

Chain PREROUTING (policy ACCEPT 0 packets, 0 bytes)
 pkts bytes target                      prot opt in     out     source        destination
    6    6K EGRESSGATEWAY-MARK-REQUEST  0    --  *      *       0.0.0.0/0     0.0.0.0/0     /* egw:Lh98b3mb9WlZrgw7 */ /* Checking for EgressPolicy matched traffic */
    0     0 ACCEPT                      0    --  *      *       0.0.0.0/0     0.0.0.0/0     /* egw:4vaggeYl6c-Gn0Yv */ /* EgressGateway traffic accept datapath rule */ mark match 0x26000000/0xff000000
```

## 3. 检查 EgressGateway Tunnel 配置

```shell
$ kubectl get egt
NAME    TUNNELMAC           TUNNELIPV4       TUNNELIPV6   MARK         PHASE
node1   66:c5:d6:cc:29:54   192.200.142.89                0x2698ee37   Ready
node2   66:66:41:3e:23:6b   192.200.201.40                0x26b7b4e4   Ready
```

我看可以看到 EgressGateway 每个节点的 TUNNELIPV4 地址，你可以 ping 这些地址，确认它们是否可以互通。

如果 TUNNEL IP 地址不通，可能是以下原因：

- 节点间网络不通：检查节点间的网络连接，确保它们可以互相 ping 通。
- 通过 `ip a show egress.vxlan` 命令检查 EgressGateway 的 VXLAN 接口是否存在，并且状态是 UP 或者 UNKNOWN，如果是 DOWN 状态，可能是 VXLAN 接口没有正确创建。

## 4. 检查 EgressPolicy 规则

```bash
$ kubectl get egresspolicy -A
NAMESPACE   NAME   GATEWAY   IPV4            IPV6   EGRESSNODE
default     test   default   172.16.25.189          node2

$ kubectl get egresspolicy test -o yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressPolicy
metadata:
  generation: 1
  name: test
  namespace: default
spec:
  appliedTo:
    podSelector:
      matchLabels:
        app: visitor
  destSubnet:
  - 172.16.25.183/32
  egressGatewayName: default
  egressIP:
    allocatorPolicy: default
    useNodeIP: false
status:
  eip:
    ipv4: 172.16.25.189
  node: node2
```

检查 EgressPolicy 的配置，确保以下几点：

- `spec.egressGatewayName` 是否正确指定了 EgressGateway 的名称。
- `spec.appliedTo.podSelector` 是否正确匹配了需要使用 EgressPolicy 的 Pod。
- `spec.destSubnet` 是否正确指定了目标子网。
- `spec.egressIP` 是否分配了出口 IP 地址。

你可以通过在 Pod 所在节点的主机上执行以下命令来检查 ipset 规则，如果你没有安装 ipset 工具，可以通过以下命令安装：

```bash
# Ubuntu/Debian
apt-get install ipset

# CentOS / RHEL / Rocky Linux / AlmaLinux
yum install ipset
```

首先检查 EgressPolicy 的源 IP 地址规则：

```bash
$ ipset list | grep egress-src-
Name: egress-src-v4-738bb014438bdbfe7

$ ipset list egress-src-v4-738bb014438bdbfe7
Name: egress-src-v4-738bb014438bdbfe7
Type: hash:net
Revision: 7
Header: family inet hashsize 1024 maxelem 65536 bucketsize 12 initval 0x498b6df2
Size in memory: 504
References: 1
Number of entries: 1
Members:
10.200.0.22
```

上面这个命令会列出所有的 egress-src-* 的 ipset 规则，可以检查对应规则是否包含 Pod 的 IP 地址。

```shell
$ ipset list | grep egress-dst-
Name: egress-dst-v4-738bb014438bdbfe7

$ ipset list egress-dst-v4-738bb014438bdbfe7
Name: egress-dst-v4-738bb014438bdbfe7
Type: hash:net
Revision: 7
Header: family inet hashsize 1024 maxelem 65536 bucketsize 12 initval 0x14fe6b9e
Size in memory: 504
References: 1
Number of entries: 1
Members:
172.16.25.183
```

上面这个命令会列出所有的 `egress-dst-*` 的 ipset 规则，可以检查对应规则是否包含目标子网的 IP 地址。

如果没有找到对应的 ipset 规则，可能是 EgressPolicy 没有填写 `spec.destSubnet`，如果是这样设置，那么 EgressGateway 会将所有流量都转发出去，除了 Kubernetes 集群内的流量。

经过匹配的流量会被打上 `0x26*` 标记的流量，你可以通过以下命令检查是否有流量被标记：

```bash
$ iptables -t mangle -nvL PREROUTING
Chain PREROUTING (policy ACCEPT 0 packets, 0 bytes)
 pkts bytes target     prot opt in     out     source               destination
8395K 1939M EGRESSGATEWAY-MARK-REQUEST  0    --  *      *       0.0.0.0/0            0.0.0.0/0            /* egw:Lh98b3mb9WlZrgw7 */ /* Checking for EgressPolicy matched traffic */
    0     0 ACCEPT     0    --  *      *       0.0.0.0/0            0.0.0.0/0            /* egw:4vaggeYl6c-Gn0Yv */ /* EgressGateway traffic accept datapath rule */ mark match 0x26000000/0xff000000
    
$ iptables -t mangle -nvL EGRESSGATEWAY-MARK-REQUEST
Chain EGRESSGATEWAY-MARK-REQUEST (1 references)
 pkts bytes target     prot opt in     out     source               destination
    0     0 MARK       0    --  *      *       0.0.0.0/0            0.0.0.0/0            /* egw:c2Rxsf_p-hXQYsiG */ /* Set mark for EgressPolicy default-test */ match-set egress-src-v4-738bb014438bdbfe7 src match-set egress-dst-v4-738bb014438bdbfe7 dst ctdir ORIGINAL MARK set 0x26b7b4e4
```

你可以看到这里 ipset 匹配 egress-src-v4-738bb014438bdbfe7 和 egress-dst-v4-738bb014438bdbfe7 的流量被打上了 `0x26b7b4e4` 的标记。


```bash
$ iptables -t nat -nvL POSTROUTING
Chain POSTROUTING (policy ACCEPT 155K packets, 9365K bytes)
 pkts bytes target                  prot opt in     out       source               destination
 155K 9361K EGRESSGATEWAY-SNAT-EIP  0    --  *      *         0.0.0.0/0            0.0.0.0/0            /* egw:x1tdBi75jif7GCxh */ /* SNAT for egress traffic */
    0     0 ACCEPT                  0    --  *      *         0.0.0.0/0            0.0.0.0/0            /* egw:lefvkAAcigCbsdOb */ /* Accept for egress traffic from pod going to EgressTunnel */ mark match 0x26000000
 155K 9378K KUBE-POSTROUTING        0    --  *      *         0.0.0.0/0            0.0.0.0/0            /* kubernetes postrouting rules */
    0     0 MASQUERADE              0    --  *      !docker0  172.17.0.0/16        0.0.0.0/0
 154K 9354K FLANNEL-POSTRTG         0    --  *      *         0.0.0.0/0            0.0.0.0/0            /* flanneld masq */
```

另外在 Pod 所在节点，通过如下命令可以检查路由表：

```bash
$ ip rule
0:	from all lookup local
99:	from all fwmark 0x26b7b4e4 lookup 649573604
32766:	from all lookup main
32767:	from all lookup default

$ ip r show table 649573604
default via 192.200.201.40 dev egress.vxlan

$ ip r show table 649573604
default via 192.200.201.40 dev egress.vxlan

$ kubectl get egt
NAME    TUNNELMAC           TUNNELIPV4       TUNNELIPV6   MARK         PHASE
node1   66:c5:d6:cc:29:54   192.200.142.89                0x2698ee37   Ready
node2   66:66:41:3e:23:6b   192.200.201.40                0x26b7b4e4   Ready

$ kubectl get egp
NAME   GATEWAY   IPV4            IPV6   EGRESSNODE
test   default   172.16.25.189          node2
```

对于匹配的流量会经过 vxlan tunnel 转发到 EgressGateway 节点上。如上命令输出，打 `0x26b7b4e4` mark 标记的流量会被路由到 `649573604` 这个路由表中，默认路由是通过 `egress.vxlan` 接口转发到对应的 Egress Node 节点上。
如果没有看到 `0x26b7b4e4` 的规则，可以查看该节点的 Egress Agent Pod 日志。

对于 Egress IP 所在节点，你可以通过以下命令检查 Egress IP 是否正确配置：

```bash
$ sudo iptables -t nat -nvL POSTROUTING
Chain POSTROUTING (policy ACCEPT 19126 packets, 1152K bytes)
 pkts bytes target                  prot opt  in     out     source               destination
 9858  596K EGRESSGATEWAY-SNAT-EIP  0    --  *      *        0.0.0.0/0            0.0.0.0/0            /* egw:x1tdBi75jif7GCxh */ /* SNAT for egress traffic */
    2   244 ACCEPT                  0    --  *      *        0.0.0.0/0            0.0.0.0/0            /* egw:lefvkAAcigCbsdOb */ /* Accept for egress traffic from pod going to EgressTunnel */ mark match 0x26000000
 9905  599K KUBE-POSTROUTING        0    --  *      *        0.0.0.0/0            0.0.0.0/0            /* kubernetes postrouting rules */
    0     0 MASQUERADE              0    --  *      !docker0 172.17.0.0/16        0.0.0.0/0
 9875  597K FLANNEL-POSTRTG         0    --  *      *        0.0.0.0/0            0.0.0.0/0            /* flanneld masq */

$ sudo iptables -t nat -nvL EGRESSGATEWAY-SNAT-EIP
Chain EGRESSGATEWAY-SNAT-EIP (1 references)
 pkts bytes target     prot opt in     out     source               destination
    2   144 SNAT       0    --  *      *       0.0.0.0/0            0.0.0.0/0            /* egw:Jv9_F-2cYSllqljc */ /* snat policy default-test */ match-set egress-src-v4-738bb014438bdbfe7 src match-set egress-dst-v4-738bb014438bdbfe7 dst ctdir ORIGINAL to:172.16.25.189
```

首先 `POSTROUTING` 链中应该有 `EGRESSGATEWAY-SNAT-EIP` 规则，这个规则位于第一个位置。
如果你看到 `EGRESSGATEWAY-SNAT-EIP` 规则中有 `SNAT` 规则，并且 `to:` 后面是 Egress IP，那么说明 Egress IP 已经正确配置，这个规则的作用是将 EgressGateway 的流量进行 SNAT 操作，将源 IP 地址转换为 Egress IP。

## 5. 检查 Egress Node 是否可以访问 destSubnet 网络

通常我们会在 Pod 内执行如下命令，检查是否能访问目标子网，并确认返回的 source IP 是否为 EgressGateway 的 Egress IP：

```bash
curl dest_subnet_ip:port
```

同样地，也可以在 Egress Node 上执行该命令，确认 Egress Node 能否访问目标子网。EgressGateway 会根据路由选择网卡，你可以通过 `ip r get dest_subnet_ip` 检查实际使用的网卡是否符合预期。

## 6. 抓包检查

你可以在 Pod 所在节点和 Egress IP 所在节点上使用 tcpdump 抓包，检查流量是否正确转发。如果抓包结果显示流量没有正确转发，可以明确 Egress IP 所在节点是否收到 Pod 发出的流量。
如果收到了发出的流量，但没有返回流量，并且以 Egress IP 为源 IP 的流量被正确转发出去，那么可能流量被主机平台丢弃了该部分流量，请检查主机平台的防火墙规则。

如果上面所有步骤都没有解决问题，请访问我们项目的 issue 页面，提交你的问题，并附上你的环境信息和相关日志，我们会尽快帮助你解决问题。
