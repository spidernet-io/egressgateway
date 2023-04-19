# roadmap

| kind             | feature                                                                                  | schedule | status |
|------------------|------------------------------------------------------------------------------------------|----------|-------|
| gateway          | support multiple instances of gateway class                                              |          | doing |
|                  | support for namespace                                                                    |          | doing |
|                  | support default gateway class                                                            |          | doing |
|                  | all data stream could loadbalance to all gateway node                                    |          | doing |
|                  | when a gateway node breakdown , all data stream could to to healthy gateway node         |          | doing |
|                  | could specify the node interface for tunnel, by hand, or auto select a reasonable one    |          | doing |
| tunnel protocol  | vxlan                                                                                    |          | doing |
|                  | geneve                                                                                   |          |       |
| encryption       | ipsec                                                                                    |          |       |
|                  | wireGuard                                                                                |          |       |
| destination CIDR | could auto distinguish internal CIDR (calico、flannel etc, or by hand) and outside CIDR   |          | doing |
|                  | could specify the outside CIDR by hands                                                  |          | doing |
| data protocl     | tcp                                                                                      |          | doing |
|                  | udp                                                                                      |          | doing |
|                  | websocket                                                                                |          |       |
|                  | sctp                                                                                     |          |       |
|                  | multicast                                                                                |          |       |
| policy           | support priority                                                                         |          | doing |
|                  | support cluster scope policy                                                             |          | doing |
|                  | support namespace scope policy                                                           |          | doing |
| support cni      | calico                                                                                   |          | doing |
|                  | flannel                                                                                  |          | doing |
|                  | weave                                                                                    |          | doing |
|                  | macvlan+spiderpool                                                                       |          | doing |
| ip stack         | ipv4-only                                                                                |          | doing |
|                  | ipv6-only                                                                                |          | doing |
|                  | dual stack                                                                               |          | doing |
| source IP        | support EIP for application                                                              |          | doing |
|                  | support EIP for namespace                                                                |          | doing |
|                  | use node ip                                                                              |          | doing |
| datapath         | iptables with low and high version                                                       |          | doing |
|                  | ebpf                                                                                     |          |       |
| performance      | big cluster, with lots of gateway nodes                                                  |          |       |
|                  | big cluster, with lots of  nodes                                                         |          |       |
|                  | big cluster, with lots of  pods                                                          |          |       |
|                  | when gateway node down, the whole cluster could change to healthy gateway node within 2s |          |       |
|                  | after apply or modify lots of policy, it could quick take effect in a big cluster        |          |       |
|                  | after apply or modify gatewaynode, it could quick take effect in a big cluster           |          |       |
|                  | forward throughput of each gateway node                                                  |          |       |
|                  | CPU and memory usage under pressure                                                      |          |       |
| HA               | all component pods could recovery quickly and serve after breakdown                      |          |       |
|                  | all pods could run for one week without failure                                          |          |       |
| insight          | metrics                                                                                  |          |       |
|                  | log                                                                                      |          |       |
| doc              | design, usage, debug docs                                                                |          |       |
| architecture     | amd and arm                                                                              |          |       |

