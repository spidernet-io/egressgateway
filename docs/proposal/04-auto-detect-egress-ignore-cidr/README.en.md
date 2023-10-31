# Egress ignore CIDR

## Motivation

To simplify the configuration of the Egress policy, the Egress Ignore CIDR feature is introduced to allow manual and automatic acquisition of the cluster's CIDR. when the `destSubnet` field of the EgressGatewayPolicy is empty, the data plane automatically matches the EgressClusterInfo CR with traffic outside of the CIDR and forwards it to the Egress gateway. CIDR in the EgressClusterInfo CR and forwards it to the Egress gateway.

## Objective

* Optimize the EgressGatewayPolicy experience.

## Design

### EgressClusterInfo CRD

Cluster-level CRD.

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressClusterInfo
metadata:
  name: default  # 1
spec:
  autoDetect:
    clusterIP: true # 2
    nodeIP: true # 3
    podCidrMode: auto # 4
  extraCidr: # 5
  - 10.10.10.1
status:
  clusterIP: # 6
    ipv4:
    - 172.41.0.0/16
    ipv6:
    - fd41::/108
  extraCidr: # 7
  - 10.10.10.1
  nodeIP: # 8
    egressgateway-control-plane:
      ipv4:
      - 172.18.0.3
      ipv6:
      - fc00:f853:ccd:e793::3
    egressgateway-worker:
      ipv4:
      - 172.18.0.2
      ipv6:
      - fc00:f853:ccd:e793::2
    egressgateway-worker2:
      ipv4:
      - 172.18.0.4
      ipv6:
      - fc00:f853:ccd:e793::4
  podCIDR: # 9
    default-ipv4-ippool:
      ipv4:
      - 172.40.0.0/16
    default-ipv6-ippool:
      ipv6:
      - fd40::/48
    test-ippool:
      ipv4:
      - 177.70.0.0/16
  podCidrMode: calico # 10
```

1. the name is `default', only one can be created by system maintenance; 2.
2. `clusterIP`, if set to `true`, the `Service CIDR` will automatically detect the `nodeIP` associated with the `nodeIP`.
3. `nodeIP`, if set to `true`, it will automatically detect changes in `nodeIP` and dynamically update it to `status.nodeIP` in `EgressClusterInfo`. 4.
4. `podCidrMode`, currently supports `k8s`, `calico`, `auto`, `""`, indicates to automatically detect the corresponding podCidr, default is `auto`, if `auto` means to automatically detect the cni used by the cluster, and use the cluster's podCidr if it can't be detected. if `""` means not to detect the podCidr, if `""` means not to detect the podCidr. If `""` is used, the cluster's podCidr is used.
5. `extraCidr`, you can manually fill in the `IP` cluster to be ignored.
6. `status.clusterIP`, if `spec.autoDetect.clusterIP` is `true`, the cluster `Service CIDR` is automatically detected and updated here.
7. `status.extraCidr`, which corresponds to `spec.extraCidr
8. `status.nodeIP`, if `spec.autoDetect.nodeIP` is `true`, then automatically detect cluster `nodeIP` and update here
9. `status.podCIDR`, corresponding to `spec.autoDetect.podCidrMode`, performs the relevant `podCidr` update
10. `status.podCidrMode`, corresponding to scenarios where `spec.autoDetect.podCidrMode` is `auto

### Data plane policy

#### ipset

The data-plane changes the iptables match policy to `-match-set !egress-ingore-cidr dst` when dealing with an EgressGatewayPolicy where `destSubnet` is empty.

```shell
iptables -A EGRESSGATEWAY-MARK-REQUEST -t mangle -m conntrack --ctdir ORIGINAL \
-m set --match-set !egress-ingore-cidr dst  \
-m set --match-set $IPSET_RULE_SRC_NAME src  \
-j MARK --set-mark $NODE_MARK -m comment --comment "rule uuid: mark request packet"
```

### Code design

#### Controller

Add a new control loop that watches the cluster's relevant resources based on the `spec.autoDetect` configuration, updating the auto-detected CIDR to the `status` of the EgressClusterInfo CR.

#### Agent

* Processes EgressClusterInfo updates into an ipset named `egress-ingore-cidr` in the control loop for Policy;
* For EgressGatewayPolicy policies when the `destSubnet` field is empty, match traffic using the ipset named `egress-ingore-cidr`.
