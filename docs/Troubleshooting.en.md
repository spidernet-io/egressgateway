## VXLAN Speed

EgressGateway uses the vxlan tunnel, and testing shows that vxlan loss is around 10%. If you find that the speed of EgressGateway does not meet the standard, you can follow these steps to check:

1. Confirm that the speed of the host-to-node matches the expected speed;
    1. The offload setting of the network card used by vxlan on the host will have a small impact on the speed of the vxlan interface (there will only be a difference of 0.5 Gbits/sec in the 10G network card test), you can run `ethtool --offload host-interface-name rx on tx on` to turn on offload;
2. The offload setting of the vxlan network card can significantly impact the speed of the vxlan interface. In 10G network card tests, the speed is 2.5 Gbits/sec without offload enabled, and 8.9 Gbits/sec with offload enabled. You can run `ethtool -k egress.vxlan` to check whether checksum offload is turned off, and you can enable offload by setting the `feature.vxlan.disableChecksumOffload` configuration in helm values to `false`.

### Benchmark

| Name   | CPU                                       | MEM  | Interface    |
|:-------|:------------------------------------------|:-----|:-------------|
| Node 1 | Intel(R) Xeon(R) CPU E5-2680 v4 @ 2.40GHz | 62G  | 10G Mellanox |
| Node 2 | Intel(R) Xeon(R) CPU E5-2680 v4 @ 2.40GHz | 125G | 10G Mellanox |


| Item         | Detail                                            |
|:-------------|:--------------------------------------------------|
| node to node | `9.10 Gbits/sec sender - 9.09 Gbits/sec receiver` |
| egress vxlan | `8.73 Gbits/sec sender - 8.73 Gbits/sec receiver` |