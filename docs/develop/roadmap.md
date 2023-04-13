# roadmap

| kind             | feature                                                                                  | schedule | status |
|------------------|------------------------------------------------------------------------------------------|----------|--------|
| gateway          | support multiple instances of gateway class                                              |          |        |
|                  | support for namespace                                                                    |          |        |
|                  | support default gateway class                                                            |          |        |
|                  | all data stream could loadbalance to all gateway node                                    |          |        |
|                  | when a gateway node breakdown , all data stream could to to healthy gateway node         |          |        |
|                  | could specify the node interface for tunnel, by hand, or auto select a reasonable one    |          |        |
| tunnel protocol  | vxlan                                                                                    |          |        |
|                  | geneve                                                                                   |          |        |
| encryption       | ipsec                                                                                    |          |        |
|                  | wireGuard                                                                                |          |        |
| destination CIDR | could auto distinguish internal CIDR (calico、flannel etc, or by hand) and outside CIDR   |          |        |
|                  | could specify the outside CIDR by hands                                                  |          |        |
| data protocl     | tcp                                                                                      |          |        |
|                  | udp                                                                                      |          |        |
|                  | websocket                                                                                |          |        |
|                  | sctp                                                                                     |          |        |
|                  | multicast                                                                                |          |        |
| support cni      | calico                                                                                   |          |        |
|                  | flannel                                                                                  |          |        |
|                  | weave                                                                                    |          |        |
|                  | macvlan+spiderpool                                                                       |          |        |
| ip stack         | ipv4-only                                                                                |          |        |
|                  | ipv6-only                                                                                |          |        |
|                  | dual stack                                                                               |          |        |
| source IP        | support EIP for application                                                              |          |        |
|                  | support EIP for namespace                                                                |          |        |
|                  | use node ip                                                                              |          |        |
| datapath         | iptables with low and high version                                                       |          |        |
|                  | ebpf                                                                                     |          |        |
| performance      | big cluster, with lots of gateway nodes                                                  |          |        |
|                  | big cluster, with lots of  nodes                                                         |          |        |
|                  | big cluster, with lots of  pods                                                          |          |        |
|                  | when gateway node down, the whole cluster could change to healthy gateway node within 2s |          |        |
|                  | after apply or modify lots of policy, it could quick take effect in a big cluster        |          |        |
|                  | after apply or modify gatewaynode, it could quick take effect in a big cluster           |          |        |
|                  | forward throughput of each gateway node                                                  |          |        |
|                  | CPU and memory usage under pressure                                                      |          |        |
| HA               | all component pods could recovery quickly and serve after breakdown                      |          |        |
|                  | all pods could run for one week without failure                                          |          |        |
| insight          | metrics                                                                                  |          |        |
|                  | log                                                                                      |          |        |
| doc              | design, usage, debug docs                                                                |          |        |
| architecture     | amd and arm                                                                              |          |        |

