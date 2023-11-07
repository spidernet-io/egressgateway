# Datapath

Rules that need to take effect are categorized into three categories: all nodes, "gateway nodes" relative to the EgressGatewayPolicy, and "non-gateway nodes". The rules on a node will only take effect when the Pod is scheduled to a "Non-Gateway Node."

## All nodes

1. Detailed tunnel requirements between nodes are not listed.
2. Traffic matching the policy is retagged. This update occurs when a node becomes a gateway node for the first time or during node join, but it is not updated thereafter.

    ```shell
    iptables -t mangle -N EGRESSGATEWAY-RESET-MARK
    iptables -t mangle -I FORWARD 1 -j EGRESSGATEWAY-RESET-MARK -m comment --comment "egress gateway: mark egress packet"
   
    iptables -t mangle -A EGRESSGATEWAY-RESET-MARK \
        -m mark --mark $NODE_MARK/0x26000000 \
        -j MARK --set-mark 0x12000000 \
        -m comment --comment "egress gateway: change mark"
    \ -m -comment "egress gateway: change mark
    ```

3. Preserve the labels for traffic matching the policy. Create them once without requiring updates.

    ```shell
    iptables -t filter -I FORWARD 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"

    iptables -t filter -I OUTPUT 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"

    iptables -t mangle -I POSTROUTING 1 -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: keep mark"
    ```

4. Aggregate chains for tagging policy-matched traffic. Create them once without needing updates.

    ```shell
    iptables -t mangle -N EGRESSGATEWAY-MARK-REQUEST

    iptables -t mangle -I PREROUTING 1 -j EGRESSGATEWAY-MARK-REQUEST -m comment --comment "egress gateway: mark egress packet"
    ```

5. Aggregate chains that do not need to do SNAT rules. It is created directly once and does not need to be updated;

    ```shell
    iptables -t nat -N EGRESSGATEWAY-NO-SNAT

    iptables -t nat -I POSTROUTING 1 -j EGRESSGATEWAY-NO-SNAT -m comment --comment "egress gateway: no snat"
   
    iptables -t nat -A EGRESSGATEWAY-NO-SNAT -m mark --mark 0x12000000 -j ACCEPT -m comment --comment "egress gateway: no snat"
    ```

6. Aggregate chains that need to do SNAT rules. It is created directly once and does not need to be updated.

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

## Non-Egress Gateway node Relative to EIP

1. ipsets for policy-matched source and destination IPs.

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

2. Tag policy-matched traffic to ensure it goes through the tunnel. The NODE_MARK value depends on the node where the corresponding EIP resides.

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

4. Adapt Weave to avoiding SNAT into IPs for Egress tunnels. Make a switch

    ```shell
    iptables -t nat -A EGRESSGATEWAY-NO-SNAT \ \
    -m set --match-set $IPSET_RULE_DEST_NAME dst \
    -m set --match-set $IPSET_RULE_SRC_NAME src \
    -j ACCEPT -m comment --comment "egress gateway: weave does not do SNAT"
    ```

## Egress Gateway Node Relative to EIP

1. ipsets for policy-matched source and destination IPs.

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

2. Apply SNAT to policy-matched traffic during egress. Keep this rule updated in real-time.

    ```shell
    iptables -t nat -A EGRESSGATEWAY-SNAT-EIP \
        -m set --match-set $IPSET_RULE_SRC_NAME src \
        -m set --match-set $IPSET_RULE_DST_NAME dst \
        -j SNAT --to-source $EIP
    ðŸñ'ðŸñ'ðŸñ'ðŸñ'ðŸñ'ñ
    ```

## Others

1. NODE_MARK: each node corresponds to a globally unique label. The label is generated by combining a prefix and a unique identifier. The format of the label is as follows: `NODE_MARK = 0x26 + value + 0000`, where `value` is a 16-bit number. The total number of supported nodes is `2^16`.

2. TABLE_NUM:

    * Since each host can have [0, 255] routing tables (where 0, 253, 254, and 255 are already used by the system), exceeding the maximum number of tables will result in the inability to calculate routes for nodes, leading to node disconnection. Additionally, table names must match the table ID, and if there is no match, the kernel will assign a random name. To be on the safe side, the number of controlled tables (represented by variable n with a default value of 100) is limited, which also serves as the upper limit for gateway nodes.
    * TABLE_NUM algorithm: users can set a starting value (represented by variable s with a default value of 3000), and the range of table names will be [s, (s+n)]. Users need to ensure that the table names within this range are not occupied. Start with a randomly selected value from [s, (s+n)] and increment it circularly until an unused table name for the current node is obtained. If none is found, an error is reported.
