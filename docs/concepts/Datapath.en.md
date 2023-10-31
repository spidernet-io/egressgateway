# Datapath

Rules that need to take effect are categorized into three categories: all nodes, "gateway nodes" relative to the EgressGatewayPolicy, and "non-gateway nodes". Rules on a Non-Grid Node will only take effect if it is a Non-Grid Node that the Service Pod is dispatched to.

## All nodes

1. The rules for tunneling between nodes will not be expanded. 2;
2. relabel the traffic that the policy hits. This is done when the node first becomes a gateway node, or once when the node joins, but not later;

   ```shell
   iptables -t mangle -N EGRESSGATEWAY-RESET-MARK
   iptables -t mangle -I FORWARD 1 -j EGRESSGATEWAY-RESET-MARK -m comment --comment "egress gateway: mark egress packet"
   
   iptables -t mangle -A EGRESSGATEWAY-RESET-MARK \
       -m mark --mark $NODE_MARK/0x26000000 \
       -j MARK --set-mark 0x12000000 \
       -m comment --comment "egress gateway: change mark"
   \ -m -comment "egress gateway: change mark

3. keep policy hit traffic labeled. Create it once directly, without updating it;

   ```shell
   iptables -t filter -I FORWARD 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"

   iptables -t filter -I OUTPUT 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"

   iptables -t mangle -I POSTROUTING 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"
   ```

4. aggregation policy hit traffic labeled chain. It is created directly once and does not need to be updated;

   ```shell
   iptables -t mangle -N EGRESSGATEWAY-MARK-REQUEST

   iptables -t mangle -I PREROUTING 1 -j EGRESSGATEWAY-MARK-REQUEST -m comment --comment "egress gateway: mark egress packet"
   ```

5. aggregates chains that do not need to do SNAT rules. It is created directly once and does not need to be updated;

   ```shell
   iptables -t nat -N EGRESSGATEWAY-NO-SNAT

   iptables -t nat -I POSTROUTING 1 -j EGRESSGATEWAY-NO-SNAT -m comment --comment "egress gateway: no snat"
   
   iptables -t nat -A EGRESSGATEWAY-NO-SNAT -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: no snat"
   ```

6. aggregating chains that need to do SNAT rules. It is created directly once and does not need to be updated.

   ```shell
   iptables -t nat -N EGRESSGATEWAY-SNAT-EIP

   # Need to insert after rules that don't require SNAT to keep the chain at the top
   iptables -t nat -I POSTROUTING 1 -j EGRESSGATEWAY-SNAT-EIP -m comment --comment "egress gateway: snat EIP"
   ```

7. egress-ingore-cidr When the `destSubnet` field of the EgressGatewayPolicy is empty, the data plane will automatically match traffic outside the CIDR in the EgressClusterStatus CR and forward it to the Egress gateway.

    ```shell
   IPSET_RULE_DEST_NAME=egress-ingore-cidr

   ipset x $IPSET_RULE_DEST_NAME

   ipset create $IPSET_RULE_DEST_NAME hash:net

   ipset add $IPSET_RULE_DEST_NAME 10.6.105.150/32
   ```

## Non-Egress Gateway node relative to EIP

1. policy hit source IP, destination IP for ipset;

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

2. policy Hit traffic is labeled to ensure that it can go through the tunnel. The value of NODE_MARK is based on the node of the policy's corresponding EIP.

   ```shell
   iptables -A EGRESSGATEWAY-MARK-REQUEST -t mangle -m conntrack --ctdir ORIGINAL \
   -m set --match-set $IPSET_RULE_DEST_NAME dst \
   -m set --match-set $IPSET_RULE_SRC_NAME src \
   -j MARK --set-mark $NODE_MARK -m comment --comment "rule uuid: mark request packet"
   ```

3. Policy routing rules

   ```shell
   ip rule add fwmark $NODE_MARK table $TABLE_NUM
   ```

4. adapting Weave to avoid SNAT into IPs for Egress tunnels. make a switch

   ```shell
   iptables -t nat -A EGRESSGATEWAY-NO-SNAT \ \
   -m set --match-set $IPSET_RULE_DEST_NAME dst \
   -m set --match-set $IPSET_RULE_SRC_NAME src \
   -j ACCEPT -m comment --comment "egress gateway: weave does not do SNAT"
   ```

## Egress Gateway node relative to EIP

1. policy hit source IP, destination IP for ipset;

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

2. policy hit traffic. Do SNAT on the way out of the gateway. real-time updates.

   ```shell
   iptables -t nat -A EGRESSGATEWAY-SNAT-EIP \
       -m set --match-set $IPSET_RULE_SRC_NAME src \
       -m set --match-set $IPSET_RULE_DST_NAME dst \
       -j SNAT --to-source $EIP
   ðŸñ'ðŸñ'ðŸñ'ðŸñ'ðŸñ'ñ
    ```

## Other

1. NODE_MARK: Each node corresponds to one, globally unique label. The label is generated by prefix + unique identifier. The label format is `NODE_MARK = 0x26 + value + 0000`, `value` is 16-bit, and the total number of supported nodes is `2^16`. 2.

2. TABLE_NUM:

* Since each host can only have [0, 255] routing tables (of which 0, 253, 254, 255 are already used by the system), exceeding the number of tables will result in node routes not being calculated, and the nodes will be disconnected. Moreover, the table name matches the ID of the table, and if it doesn't, the kernel will randomly assign it. Therefore, to be on the safe side, the number of tables (n, default value is 100), which is the maximum number of gateway nodes, can be set by a variable.
* TABLE_NUM algorithm: user can set a starting value (s, default value is 3000), the table name range is [s, (s+n)], user needs to ensure that the table name of [s, (s+n)] is not occupied. Randomly take a starting value from [s, (s+n)], and then increase the value in a circular manner until it gets a table name that is not used by this node, or report an error if it is not found.
