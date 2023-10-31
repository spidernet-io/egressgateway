The EgressTunnel CRD is used to record tunnel network interface information for cross-node communication. It is a cluster scope resource that corresponds one-to-one with the Kubernetes Node resource name.

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

1. Tunnel IPv4 address
2. Tunnel IPv6 address
3. Tunnel MAC address
4. Tunnel parent network interface
5. Tunnel parent network interface IPv4 address
6. Tunnel parent network interface IPv6 address
7. Current tunnel status
    - `Pending`: Waiting for IP allocation
    - `Init`: Tunnel IP allocation successful
    - `Ready`: Tunnel IP allocated and tunnel established
    - `Failed`: Tunnel IP allocation failed
8. Packet mark value, one for each node. For example, if node A has egress traffic that needs to be forwarded to gateway node B, the traffic of node A will be marked with a mark.