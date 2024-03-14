# egctl 命令行工具说明

`egctl` 是一个命令行工具，用于管理 EgressGateway 相关资源。

## 命令概述

### vip move

移动 VIP 到指定的节点。

* `--egressGatewayName`: 指定 EgressGateway 的名称。
* `--vip`: 您想要移动的 Egress IP 地址。
* `--targetNode`: Egress IP 将生效的目标 EgressGateway 节点的名称。

```shell
egctl vip move --egressGatewayName <egress-gateway-name> --vip <vip-address> --targetNode <node-name>
```
