# Architecture

EgressGateway consists of two parts: the control plane and the data plane. The control plane is composed of four control loops, and the data plane is composed of three. The control plane is deployed as a Deployment, supporting multiple replicas for high availability, and the data plane is deployed as a DaemonSet. The control loops are as follows in the diagram below:

![arch](../images/arch.png)

## Controller

### EgressTunnel Reconcile Loop (a)

#### Initialization

1. Obtain the dual-stack open condition and the corresponding tunnel CIDR from the ConfigMap configuration file
2. Generate a unique label value by node name according to the algorithm.
3. Check if the node has a corresponding EgressTunnel, if not, create a corresponding EgressTunnel and set the status to `Pending`. If there is a tunnel IP, it will check if the IP is legal before binding the IP to the node.  If not, it will set the status to `Pending`.

#### EgressTunnel Event

- Del: release the tunnel IP first, and then delete it. If the node corresponding to EgressTunnel still exists, recreate EgressTunnel.
- Other:
  - EgressTunnel Other. = `Init` || phase ! = `Ready`: the IP is allocated and the status is set to `Init` for successful allocation and `Failed` for failed allocation. This is the only place globally where the tunnel IP will be assigned.
  - mark ! = algorithm(NodeName): this field is forbidden to be modified, and will be returned as an error.

#### Node Event

- Del: delete the corresponding EgressTunnel.
- Other:
  - If the corresponding EgressTunnel does not exist, create an EgressTunnel.
  - No tunnel IP, set phase to `Pending`.
  - If there is a tunnel IP, verify if the tunnel is legal. If not, set phase to `Pending`.
  - If the tunnel IP is legal, check if the IP is assigned to this node. If not, set phase to `Pending`.
  - If the tunnel IP is assigned to this node and the phase state is not `Ready`, set phase to `Init`.

### EgressGateway Reconcile Loop (b)

#### EgressGateway Event

- Del:
  - The Webhook determines if the Policy is still referenced by other Policies, and if so, deletion is not allowed.
  - Passing the Webhook's check indicates there is no references and all rules have been cleaned up, so deletion is allowed.

- Other:
  - If the number of EIPs decreases and EIP is referenced, modification is prohibited. When allocating IPV4 and IPV6, it is required that the number of IPV4 and IPV6 correspond to each other, so the number of IPV4 and IPV6 should be the same.
  - If the nodeSelector is modified, get the old Node information from status and compare it with the latest Node. Reallocate the EIP from the deleted node to the new Node. Update the EIP information in the corresponding EgressTunnel.

#### EgressPolicy Event

- Del: list out the EgressPolicy, and find the referenced EgressGateway, and then unbind the EgressPolicy to the EgressGateway. To perform unbinding, we need to find the corresponding EIP information. If the EIP is used, then determine whether the EIP should be reclaimed; if the EIP is no longer used by the policy, then reclaim the EIP and update itself the EIP information of the EgressTunnel.
- Other:
  - EgressPolicy cannot modify the bound EgressGateway. If it is allowed to do so, list out the EgressGateway, and then find the original bound EgressGateway and unbind it. Then, bind it to the new EgressGateway.
  - If you add a new EgressPolicy, then bind the EgressPolicy to the EgressGateway, and in the process of binding, determine if you need to assign an EIP.

#### Node Event

- Del: list out the EgressGateway to pick out the EIPs that are active on this node and reassign those EIPs to the new node. Update the EgressGateway's eip.policy.
- Other:
  - The NoReady event is equivalent to triggering a deletion event.
  - For label modifications, iterate through all EgressGateway information to check if it involves nodeSelector. If the old labels do not involve any EgressPolicies, no action is taken. If they are involved, it is equivalent to triggering a deletion event. If the new labels meet the conditions for the EgressGateway, update the status information of the corresponding EgressGateway.

### Gateway Node Selection EgressPolicy and EIP Assignment Logic

An EgressPolicy selects a node as a gateway node according to the gateway node selection policy. The decision to assign an EIP is based on whether or not the EIP is used, and the assigned EIP is bound to the selected gateway node.

The allocation logic applies to individual EgressGateways rather than all EgressGateways.

#### EgressPolicy Gateway Node Selection Modes

- Average selection: when a gateway node needs to be selected, select the node with the least number of nodes as gateway nodes.
- Minimum node selection: try to select the same node as a gateway node.
- Limit selection: a node can only serve as a gateway for up to a certain number of EgressPolicies. This limit can be set and defaults to 5. If a node hasn't reached the limit, it is preferred. Once the limit is reached, other nodes are chosen first. If all nodes have reached their limits, a random selection is made.

#### EIP Allocation Logic

- Random allocation: randomly select an EIP from all available EIPs, regardless of whether it has been assigned
- Preferred use of unallocated EIPs: use unallocated EIPs first, and then randomly allocate a used EIP if they are all used.
- Limit selection: an EIP can be used by a maximum number of EgressPolicies, which can be set with a default value of 5. Until the limit is reached, the EIP is preferred for allocation. Once the limit is reached, other EIPs are selected. If all EIPs have reached their limits, a random selection is made.

#### EIP Recycle Logic

When an EIP is not used, it will be reclaimed, which means deleting the EIP field in `eips`.

### EgressClusterInfo Reconcile Loop (d)

#### Node Event

- Create: node IP is automatically added to egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP` when node is created.
- Update: when the node IP is updated, the node's IP is automatically updated to egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP`.
- Delete: remove the node's IP from egressclusterinfos CR `status.egressIgnoreCIDR.nodeIP` when the node is deleted.

#### Calico IPPool Event

Listen for Calico's IPPool Event when the `egressIgnoreCIDR.autoDetect.podCIDR` of the egressgateway profile is "calico".

- Create: automatically add the IPPool CIDR to the EgressClusterInfo CR `status.egressIgnoreCIDR.podCIDR` when the Calico IPPool is created.
- Update: when calico IPPool has an update, automatically update the IPPool CIDR into EgressClusterInfo CR `status.egressIgnoreCIDR.podCIDR`.
- Delete: remove the IPPool CIDR from EgressClusterInfo CR `status.egressIgnoreCIDR.podCIDR` when the calico IPPool is deleted.

#### Configuration File

Modify the configuration file to add the following configuration:

```yaml
feature.
  egressIgnoreCIDR.
    autoDetect.
      podCIDR: "" # (1)
      clusterIP: true # (2)
      nodeIP: true # (3)
    custom.
      - "10.6.1.0/24"
```

1. `podCIDR`, currently support `calico` and `k8s`. The default is `k8s`.
2. `clusterIP`, support setting to Service CIDR auto-detection.
3. `nodeIP`, support setting to Node IP auto-detection.
