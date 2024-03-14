# egctl cli reference

`egctl` is a command line tool for managing EgressGateway related resources.

## Command Overview

### vip move

Move a VIP to a specified node.

* `--egressGatewayName`: Specifies the name of the EgressGateway.
* `--vip`: The Egress IP address you want to move.
* `--targetNode`: The name of the target EgressGateway Node where the Egress IP will take effect.

```shell
egctl vip move --egressGatewayName <egress-gateway-name> --vip <vip-address> --targetNode <node-name>
```