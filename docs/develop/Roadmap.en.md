| Kind             | Feature                                                                                  | Schedule | Status |
|------------------|------------------------------------------------------------------------------------------|----------|--------|
| Gateway          | Support multiple instances of gateway class                                              |          | v0.4.0 |
|                  | Support for namespace                                                                    |          |        |
|                  | Support default gateway class                                                            |          | v0.4.0 |
|                  | All data stream could load-balance to all gateway node                                   |          | v0.4.0 |
|                  | When a gateway node breakdown , all data stream could to to healthy gateway node         |          | v0.4.0 |
|                  | Could specify the node interface for tunnel, by hand, or auto select a reasonable one    |          | v0.4.0 |
| Tunnel protocol  | VXLAN                                                                                    |          | v0.4.0 |
|                  | Geneve                                                                                   |          |        |
| Encryption       | IPSec                                                                                    |          |        |
|                  | WireGuard                                                                                |          |        |
| Destination CIDR | Could auto distinguish internal CIDR (calico, flannel etc, or by hand) and outside CIDR  |          | v0.4.0 |
|                  | Could specify the outside CIDR by hands                                                  |          | v0.4.0 |
| Data protocol    | TCP                                                                                      |          | v0.4.0 |
|                  | UDP                                                                                      |          | v0.4.0 |
|                  | WebSocket                                                                                |          | v0.4.0 |
|                  | sctp                                                                                     |          |        |
|                  | Multicast                                                                                |          |        |
| Policy           | Support priority                                                                         |          |        |
|                  | Support cluster scope policy                                                             |          | v0.4.0 |
|                  | Support namespace scope policy                                                           |          | v0.4.0 |
| Support CNI      | Calico                                                                                   |          | v0.4.0 |
|                  | Flannel                                                                                  |          | v0.4.0 |
|                  | Weave                                                                                    |          | v0.4.0 |
|                  | macvlan+spiderpool                                                                       |          | v0.3.0 |
| IP Stack         | IPv4-only                                                                                |          | v0.4.0 |
|                  | IPv6-only                                                                                |          | v0.4.0 |
|                  | Dual stack                                                                               |          | v0.4.0 |
| Source IP        | Support EIP for application                                                              |          | v0.4.0 |
|                  | Support EIP for namespace                                                                |          | v0.4.0 |
|                  | Use node IP                                                                              |          | v0.4.0 |
| Datapath         | Iptables with low and high version                                                       |          | v0.4.0 |
|                  | ebpf                                                                                     |          |        |
| Performance      | Big cluster, with lots of gateway nodes                                                  |          |        |
|                  | Big cluster, with lots of  nodes                                                         |          |        |
|                  | Big cluster, with lots of  pods                                                          |          |        |
|                  | When gateway node down, the whole cluster could change to healthy gateway node within 2s |          |        |
|                  | After apply or modify lots of policy, it could quick take effect in a big cluster        |          |        |
|                  | After apply or modify gateway node, it could quick take effect in a big cluster          |          |        |
|                  | Forward throughput of each gateway node                                                  |          |        |
|                  | CPU and memory usage under pressure                                                      |          |        |
| HA               | All component pods could recovery quickly and serve after breakdown                      |          | v0.4.0 |
|                  | All pods could run for one week without failure                                          |          | v0.4.0 |
| Insight          | Metrics                                                                                  |          | v0.4.0 |
|                  | Log                                                                                      |          | v0.4.0 |
| Architecture     | AMD, ARM                                                                                 |          | v0.4.0 |

