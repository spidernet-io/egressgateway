对需要生效的规则分为三类：所有节点，相对于 EgressGatewayPolicy 的「网关节点」和「非网关节点」。只有当业务 Pod 调度到的「非网关节点」，该节点上的规则才会生效。

## 所有节点

1. 各节点之间，隧道需要打通的规则就就不一一展开；
2. 将 policy 命中的流量，重新打标签。节点第一次变成网关节点时更新，或者节点 join 时做一次，后面不更新；
   ```shell
   iptables -t mangle -N EGRESSGATEWAY-RESET-MARK
   iptables -t mangle -I FORWARD 1  -j EGRESSGATEWAY-RESET-MARK -m comment --comment "egress gateway: mark egress packet"
   
   iptables -t mangle -A EGRESSGATEWAY-RESET-MARK \
       -m mark --mark $NODE_MARK/0x26000000 \
       -j MARK --set-mark 0x12000000 \
       -m comment --comment "egress gateway: change mark"
   ```

3. 保持 policy 命中流量的标签。直接创建一次，不需要更新；
   ```shell
   iptables -t filter -I FORWARD 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"

   iptables -t filter -I OUTPUT 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"

   iptables -t mangle -I POSTROUTING 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"
   ```

4. 聚合 policy 命中流量打标签的链。直接创建一次，不需要更新；
   ```shell
   iptables -t mangle -N EGRESSGATEWAY-MARK-REQUEST

   iptables -t mangle -I PREROUTING 1 -j EGRESSGATEWAY-MARK-REQUEST -m comment --comment "egress gateway: mark egress packet"
   ```

5. 聚合不需要做 SNAT 规则的链。直接创建一次，不需要更新；
   ```shell
   iptables -t nat -N EGRESSGATEWAY-NO-SNAT

   iptables -t nat -I POSTROUTING 1  -j EGRESSGATEWAY-NO-SNAT -m comment --comment "egress gateway: no snat"
   
   iptables -t nat -A EGRESSGATEWAY-NO-SNAT -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: no snat"
   ```

6. 聚合需要做 SNAT 规则的链。直接创建一次，不需要更新。
   ```shell
   iptables -t nat -N EGRESSGATEWAY-SNAT-EIP

   # 需要在不需要 SNAT 的规则后面插入，才能保证该链在最前面
   iptables -t nat -I POSTROUTING 1  -j EGRESSGATEWAY-SNAT-EIP -m comment --comment "egress gateway: snat EIP"
   ```

7. egress-ingore-cidr 当 EgressGatewayPolicy 的 `destSubnet` 字段为空时，数据面将会自动匹配 EgressClusterStatus CR 中的 CIDR 之外的流量，并将其转发到 Egress 网关。
    ```shell
   IPSET_RULE_DEST_NAME=egress-ingore-cidr

   ipset x $IPSET_RULE_DEST_NAME

   ipset create $IPSET_RULE_DEST_NAME hash:net

   ipset add $IPSET_RULE_DEST_NAME 10.6.105.150/32
   ```

## 相对于 EIP 的非 Egress Gateway 节点

1. policy 命中的源 IP、目的 IP 的 ipset；
   ```shell
   IPSET_RULE_DEST_NAME=egress-dest-uuid

   ipset x $IPSET_RULE_DEST_NAME

   ipset create $IPSET_RULE_DEST_NAME hash:net

   ipset add $IPSET_RULE_DEST_NAME 10.6.105.150/32

   
   IPSET_RULE_SRC_NAME=egress-src-uuid

   ipset x $IPSET_RULE_SRC_NAME

   ipset create $IPSET_RULE_SRC_NAME hash:net

   ipset add $IPSET_RULE_SRC_NAME 172.29.234.173/32
   ```

2. policy 命中的流量打标签，保证能从隧道走。其中 NODE_MARK 的值根据 policy 对应的 EIP 所在节点决定。
   ```shell
   iptables -A EGRESSGATEWAY-MARK-REQUEST -t mangle -m conntrack --ctdir ORIGINAL \
   -m set --match-set $IPSET_RULE_DEST_NAME dst  \
   -m set --match-set $IPSET_RULE_SRC_NAME src  \
   -j MARK --set-mark $NODE_MARK -m comment --comment "rule uuid: mark request packet"
   ```

3. 策略路由规则
   ```shell
   ip rule add fwmark $NODE_MARK table $TABLE_NUM
   ```

4. 适配 Weave 避免做 SNAT 成 Egress 隧道的 IP。做成开关
   ```shell
   iptables -t nat -A EGRESSGATEWAY-NO-SNAT \
   -m set --match-set $IPSET_RULE_DEST_NAME dst  \
   -m set --match-set $IPSET_RULE_SRC_NAME src  \
   -j ACCEPT -m comment --comment "egress gateway: weave does not do SNAT"
   ```

## 相对于 EIP 的 Egress Gateway 节点
1. policy 命中的源 IP、目的 IP 的 ipset；
   ```shell
   IPSET_RULE_DEST_NAME=egress-dest-uuid

   ipset x $IPSET_RULE_DEST_NAME

   ipset create $IPSET_RULE_DEST_NAME hash:net

   ipset add $IPSET_RULE_DEST_NAME 10.6.105.150/32
   

   IPSET_RULE_SRC_NAME=egress-src-uuid

   ipset x $IPSET_RULE_SRC_NAME

   ipset create $IPSET_RULE_SRC_NAME hash:net

   ipset add $IPSET_RULE_SRC_NAME 172.29.234.173/32
   ```

2. policy 命中的流量。出网关时做 SNAT。实时更新。
   ```shell
   iptables -t nat -A EGRESSGATEWAY-SNAT-EIP \
       -m set --match-set $IPSET_RULE_SRC_NAME src \
       -m set --match-set $IPSET_RULE_DST_NAME dst \
       -j SNAT --to-source $EIP
   ```

## 其他
1. NODE_MARK：每个节点对应一个，全局唯一的标签。标签由前缀 + 唯一标识符生成。标签格式如下 `NODE_MARK = 0x26 + value + 0000`，`value` 为 16 位，支持的节点总数为 `2^16`。
2. TABLE_NUM：
* 由于每个主机只能 [0, 255] 张路由表（其中 0、253、254、255 已被系统使用），超出表的张数时，会导致节点路由没法计算，从而节点失联。而且表名与表的 ID 匹配，如果没有匹配，则内核会随机分配。所以为了保险起见，控制表的的张数（n 表示，默认值为 100）也就是网关节点的上限，可以通过变量设置。
* TABLE_NUM 算法：用户可以设置一个起始值（s 表示，默认值为 3000），则表名的范围为 [s, (s+n)]，用户需要保证 [s, (s+n)] 的表名没有被占用。随机从 [s, (s+n)] 取一个起始值，依次增加，环形取值，直到获得一个本节点未使用的表名，未找到则报错。