# Architecture

EgressGateway consists of two parts: the control plane and the data plane. The control plane is composed of four control loops, and the data plane is composed of three control loops. The control plane is deployed as a Deployment, supporting multiple replicas for high availability, and the data plane is deployed as a DaemonSet. The control loops are as follows in the diagram below:

![arch](../proposal/03-egress-ip/arch.png)

## Controller

### EgressTunnel reconcile loop (a)

#### Initialization

1. obtain the dual-stack open condition and the corresponding tunnel CIDR from the ConfigMap configuration file
2. Generate a unique label value by node name according to the algorithm.
3. check if the Node has a corresponding EgressTunnel, if not, create a corresponding EgressTunnel and set the status to `Pending`. If there is a tunnel IP, it will bind the IP to the node, before binding, it will check if the IP is legal, if not, it will set the status to `Pending`.

#### EgressTunnel Event

- Del: release the tunnel IP first, then delete it. If the node corresponding to EgressTunnel still exists, recreate EgressTunnel.
- Other:
  - EgressTunnel Other. = `Init` || phase ! = `Ready`: the IP is allocated and the status is set to `Init` for successful allocation and `Failed` for failed allocation. This is the only place globally where the tunnel IP will be assigned.
  - mark ! = algorithm(NodeName): this field is forbidden to be modified, and will be returned as an error.

#### Node Event

- Del: delete the corresponding EgressTunnel.
- Other:
  - If the corresponding EgressTunnel does not exist, create an EgressTunnel.
  - No tunnel IP, set phase to `Pending`.
  - With tunnel IP, verify if the tunnel is legal, if not, set phase to `Pending`.
  - If the tunnel IP is legal, check if the IP is assigned to this node, if not, set phase to `Pending`.
  - If the tunnel IP is assigned to this node and the phase state is not `Ready`, set phase to `Init`.

### EgressGateway reconcile loop (b)

#### EgressGateway Event

- Del:
  - The Webhook determines if the Policy is still referenced by other Policies, and if so, does not allow it to be deleted.
  - If it passes the Webhook's check, it is not referenced and the rule is cleaned up, so it can be deleted.

- Other:
  - EIP reduction, if the EIP is referenced, modification is prohibited. When allocating IPV4 and IPV6, it is required that the number of IPV4 and IPV6 correspond to each other, so the number of IPV4 and IPV6 should be the same.
  - If the nodeSelector is modified, get the old Node information from status and compare it with the latest Node. Redistribute the EIP from the deleted node to the new Node. Update the EIP information in the corresponding EgressTunnel.

#### EgressPolicy Event

- Del: lists the EgressPolicy, finds the referenced EgressGateway, and then unbinds the EgressPolicy to the EgressGateway. To unbundle the EgressPolicy and EgressGateway, we need to find the corresponding EIP information. If the EIP is used, then determine whether the EIP should be reclaimed; if the EIP is no longer used by the policy, then reclaim the EIP and update the EIP information of itself and the EgressTunnel.
- Other:
  - EgressPolicy cannot modify the bound EgressGateway. if it is allowed to do so. list the EgressGateway. find the original bound EgressGateway and unbind it. and then bind the new one. Find the originally bound EgressGateway and unbind it.
  - If you add a new EgressPolicy, then bind the EgressPolicy to the EgressGateway, and in the process of binding, determine if you need to assign an EIP.

#### Node Event

- Del: list the EgressGateway to pick out the EIPs that are in effect on this node and reassign those EIPs to the new node. Updates the EgressGateway's eip.policy.
- Other:
  - The NoReady event is equivalent to triggering a deletion event when the NoReady event.
  - Label modification by iterating through all the information of the EgressGateway, whether nodeSelector is involved or not. if the old label does not involve EgressPolicy, nothing is done. If there is an involvement, it is equivalent to triggering a delete event. If the new label matches the EgressGateway condition, update the status information of the corresponding EgressGateway.

### EgressPolicy Selection Gateway Node and EIP Assignment Logic

An EgressPolicy selects a node as a gateway node according to the gateway node selection policy. The decision to assign an EIP is based on whether or not the EIP is used, and the assigned EIP is bound to the selected gateway node.

The allocation logic is all for a single EgressGateway, not all EgressGateways.

#### EgressPolicy Modes for Selecting Gateway Nodes

- Average selection: When a gateway node needs to be selected, select the node with the least number of nodes as gateway nodes.
- Minimum Node Selection: Try to select the same node as a gateway node.
- Limit selection: a node can only be a gateway node for several EgressPolicy, the limit can be set, the default is 5. Before the limit is reached, the node is preferred to be selected, and when the limit is reached, the other nodes will be selected first, and then randomly selected if all the limits are reached.

#### EIP allocation logic

- Random allocation: randomly select one of all EIPs, regardless of whether the EIP has been allocated or not
- Priority use of unallocated EIPs: use unallocated EIPs first, and then randomly allocate a used EIP if they are all used.
- Limit selection: an EIP can only be used by several EgressPolicy at most, the limit can be set, the default is 5, before the limit is reached, the EIP will be assigned first, and if the limit is reached, then other EIPs will be selected; if the limit is reached, then the EIPs will be assigned randomly.

#### EIP Recycle Logic

When an EIP is not used, it will be reclaimed, reclaiming means deleting the EIP field in `eips`.

### EgressClusterInfo reconcile loop (d)

#### Node Event

- Create: node ip is automatically added to egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP` when node is created.
- Update: When the node ip is updated, the node's ip is automatically updated to egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP`.
- Delete: Remove the node's ip from egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP` when the node is deleted.

#### Calico IPPool Event

Listen for Calico's IPPool Event when the `egressIgnoreCIDR.autoDetect.podCIDR` of the egressgateway profile is "calico".

- Create: automatically add the IPPool CIDR to the EgressClusterInfo CR `status.egressIgnoreCIDR.podCIDR` when the Calico IPPool is created.
- Update: When calico IPPool has an update, automatically update the IPPool CIDR into EgressClusterInfo CR `status.egressIgnoreCIDR.podCIDR`.
- Delete: removes the IPPool CIDR from EgressClusterInfo CR `status.egressIgnoreCIDR.podCIDR` when the calico IPPool is deleted.

#### configuration file

Modify the configuration file to add the following configuration:

```yaml
feature.
  egressIgnoreCIDR.
    autoDetect.
      podCIDR: "" # 1
      clusterIP: true # 2
      nodeIP: true # 3
    custom.
      - "10.6.1.0/24"
```

1. `podCIDR`, currently supports `calico`, `k8s`. The default is `k8s`.
2. `clusterIP`, supports setting to Service CIDR auto-detection.
3. `nodeIP`, supports setting to Node IP auto-detection.
