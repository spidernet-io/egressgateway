# Troubleshooting Guide

## 1. Ensure Components Are Running

```bash
kubectl get pods -n kube-system | grep -i egress
```

Run this command to check if all egress Pods are in the Running state. If any Pod is not Running, it may be due to the following reasons:

- Image Pull Back Off: Check the Pod description to confirm if there is an ImagePullBackOff error.
- Pod Restart: Check the Pod restart count to see if any Pod is restarting frequently.

## 2. Check Calico CNI Configuration

If you are using Calico CNI, follow this step. If you are using another CNI, skip this step. Ensure that Calico's chainInsertMode is set to Append. You can check this with the following command:

```bash
kubectl get FelixConfiguration -n kube-system default -o yaml | grep chainInsertMode
```

If the output is not `Append` or the field is missing, you can modify it with:

```bash
kubectl patch FelixConfiguration -n kube-system default --type='json' -p='[{"op": "replace", "path": "/spec/chainInsertMode", "value": "Append"}]'
```

This determines whether EgressGateway's iptables rules will be overridden by Calico. You can verify this directly on the host with:

```bash
iptables -t mangle -nvL PREROUTING
```

If the first rule starts with `cali-*`, it means Calico's rules have overridden EgressGateway's rules.

Normally, you should see the following rules at the top:

```bash
$ iptables -t mangle -nvL PREROUTING

Chain PREROUTING (policy ACCEPT 0 packets, 0 bytes)
 pkts bytes target                      prot opt in     out     source        destination
    6    6K EGRESSGATEWAY-MARK-REQUEST  0    --  *      *       0.0.0.0/0     0.0.0.0/0     /* egw:Lh98b3mb9WlZrgw7 */ /* Checking for EgressPolicy matched traffic */
    0     0 ACCEPT                      0    --  *      *       0.0.0.0/0     0.0.0.0/0     /* egw:4vaggeYl6c-Gn0Yv */ /* EgressGateway traffic accept datapath rule */ mark match 0x26000000/0xff000000
```

## 3. Check EgressGateway Tunnel Configuration

```shell
$ kubectl get egt
NAME    TUNNELMAC           TUNNELIPV4       TUNNELIPV6   MARK         PHASE
node1   66:c5:d6:cc:29:54   192.200.142.89                0x2698ee37   Ready
node2   66:66:41:3e:23:6b   192.200.201.40                0x26b7b4e4   Ready
```

You can see the TUNNELIPV4 address of each EgressGateway node. Try pinging these addresses to confirm they are reachable.

If the TUNNEL IP addresses are unreachable, possible reasons include:

- Network connectivity issues between nodes: Check the network connection between nodes to ensure they can ping each other.
- Use `ip a show egress.vxlan` to check if the EgressGateway VXLAN interface exists and is UP or UNKNOWN. If it is DOWN, the VXLAN interface may not have been created correctly.

## 4. Check EgressPolicy Rules

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

Check the EgressPolicy configuration and ensure the following:

- `spec.egressGatewayName` correctly specifies the EgressGateway name.
- `spec.appliedTo.podSelector` correctly matches the Pods that need to use the EgressPolicy.
- `spec.destSubnet` correctly specifies the destination subnet.
- `spec.egressIP` has allocated an egress IP address.

You can check the ipset rules on the host where the Pod is located. If you do not have the ipset tool installed, install it with:

```bash
# Ubuntu/Debian
apt-get install ipset

# CentOS / RHEL / Rocky Linux / AlmaLinux
yum install ipset
```

First, check the source IP rules for the EgressPolicy:

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

The above command lists all egress-src-* ipset rules. Check if the corresponding rule contains the Pod's IP address.

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

The above command lists all `egress-dst-*` ipset rules. Check if the corresponding rule contains the destination subnet's IP address.

If you cannot find the corresponding ipset rule, it may be because `spec.destSubnet` is not set in the EgressPolicy. In this case, EgressGateway will forward all traffic except for traffic within the Kubernetes cluster.

Matched traffic will be marked with a `0x26*` mark. You can check whether any traffic has been marked using the following command:

```bash
$ iptables -t mangle -nvL PREROUTING
Chain PREROUTING (policy ACCEPT 0 packets, 0 bytes)
pkts bytes target     prot opt in     out     source               destination
8395K 1939M EGRESSGATEWAY-MARK-REQUEST  0    --  *      *       0.0.0.0/0            0.0.0.0/0            /* egw:Lh98b3mb9WlZrgw7 */ /* Checking for EgressPolicy matched traffic */
0         0 ACCEPT     0    --  *      *       0.0.0.0/0            0.0.0.0/0            /* egw:4vaggeYl6c-Gn0Yv */ /* EgressGateway traffic accept datapath rule */ mark match 0x26000000/0xff000000

$ iptables -t mangle -nvL EGRESSGATEWAY-MARK-REQUEST
Chain EGRESSGATEWAY-MARK-REQUEST (1 references)
pkts bytes target     prot opt in     out     source               destination
0        0 MARK       0    --  *      *       0.0.0.0/0            0.0.0.0/0            /* egw:c2Rxsf_p-hXQYsiG */ /* Set mark for EgressPolicy default-test */ match-set egress-src-v4-738bb014438bdbfe7 src match-set egress-dst-v4-738bb014438bdbfe7 dst ctdir ORIGINAL MARK set 0x26b7b4e4

You can see here that traffic matching the ipsets egress-src-v4-738bb014438bdbfe7 and egress-dst-v4-738bb014438bdbfe7 is marked with 0x26b7b4e4.

$ iptables -t nat -nvL POSTROUTING
Chain POSTROUTING (policy ACCEPT 155K packets, 9365K bytes)
pkts bytes target                  prot opt in     out       source               destination
155K 9361K EGRESSGATEWAY-SNAT-EIP  0    --  *      *         0.0.0.0/0            0.0.0.0/0            /* egw:x1tdBi75jif7GCxh */ /* SNAT for egress traffic */
0        0 ACCEPT                  0    --  *      *         0.0.0.0/0            0.0.0.0/0            /* egw:lefvkAAcigCbsdOb */ /* Accept for egress traffic from pod going to EgressTunnel */ mark match 0x26000000
155K 9378K KUBE-POSTROUTING        0    --  *      *         0.0.0.0/0            0.0.0.0/0            /* kubernetes postrouting rules */
0        0 MASQUERADE              0    --  *      !docker0  172.17.0.0/16        0.0.0.0/0
154K 9354K FLANNEL-POSTRTG         0    --  *      *         0.0.0.0/0            0.0.0.0/0            /* flanneld masq */
```

In addition, on the node where the Pod resides, you can check the routing table using the following command:

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

Traffic that matches will be forwarded through the VXLAN tunnel to the EgressGateway node. As shown in the command output above, traffic marked with `0x26b7b4e4` will be routed to the `649573604` routing table, where the default route forwards it via the `egress.vxlan` interface to the corresponding Egress Node.
If you don’t see the rule for `0x26b7b4e4`, you can check the logs of the Egress Agent Pod on that node.

For the node where the Egress IP is located, you can verify if the Egress IP is correctly configured using the following command:

```bash
$ sudo iptables -t nat -nvL POSTROUTING
Chain POSTROUTING (policy ACCEPT 19126 packets, 1152K bytes)
pkts bytes target                  prot opt  in     out     source               destination
9858  596K EGRESSGATEWAY-SNAT-EIP  0    --  *      *        0.0.0.0/0            0.0.0.0/0            /* egw:x1tdBi75jif7GCxh */ /* SNAT for egress traffic */
2      244 ACCEPT                  0    --  *      *        0.0.0.0/0            0.0.0.0/0            /* egw:lefvkAAcigCbsdOb */ /* Accept for egress traffic from pod going to EgressTunnel */ mark match 0x26000000
9905  599K KUBE-POSTROUTING        0    --  *      *        0.0.0.0/0            0.0.0.0/0            /* kubernetes postrouting rules */
0        0 MASQUERADE              0    --  *      !docker0 172.17.0.0/16        0.0.0.0/0
9875  597K FLANNEL-POSTRTG         0    --  *      *        0.0.0.0/0            0.0.0.0/0            /* flanneld masq */

$ sudo iptables -t nat -nvL EGRESSGATEWAY-SNAT-EIP
Chain EGRESSGATEWAY-SNAT-EIP (1 references)
pkts bytes target     prot opt in     out     source               destination
2      144   SNAT        0  --  *      *      0.0.0.0/0            0.0.0.0/0            /* egw:Jv9_F-2cYSllqljc */ /* snat policy default-test */ match-set egress-src-v4-738bb014438bdbfe7 src match-set egress-dst-v4-738bb014438bdbfe7 dst ctdir ORIGINAL to:172.16.25.189
```

First, the `POSTROUTING` chain should include the `EGRESSGATEWAY-SNAT-EIP` rule, and it should appear in the first position.
If you see a SNAT rule in the `EGRESSGATEWAY-SNAT-EIP` chain and the `to:` field points to the Egress IP, then it means the Egress IP has been correctly configured. The purpose of this rule is to perform SNAT on the EgressGateway traffic, converting the source IP address to the Egress IP.

## 5. Check if the Egress Node Can Access the destSubnet Network

Usually, run the following command inside the Pod to check if it can access the destination subnet and confirm whether the returned source IP is the EgressGateway's Egress IP:

```bash
curl dest_subnet_ip:port
```

Similarly, you can run this command on the Egress Node to confirm if it can access the destination subnet. EgressGateway will select the network interface based on routing. You can check which interface is actually used with `ip r get dest_subnet_ip`.

## 6. Packet Capture Check

You can use tcpdump on both the Pod's node and the Egress IP's node to capture packets and check if the traffic is being forwarded correctly. If the capture shows that traffic is not being forwarded correctly, you can determine whether the node with the Egress IP has received the traffic sent by the Pod.

If the traffic is received but there is no return traffic, and the traffic with the Egress IP as the source IP is being forwarded correctly, it is possible that the host platform is dropping this part of the traffic. Please check the firewall rules of the host platform.

If none of the above steps resolve the issue, please visit our project's issue page, submit your problem, and include your environment information and relevant logs. We will help you resolve the issue as soon as possible.
