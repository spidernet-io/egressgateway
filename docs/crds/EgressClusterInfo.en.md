The EgressClusterInfo CRD introduces the Egress Ignore CIDR feature to simplify the configuration of Egress policies and allows automatic acquisition of the cluster's CIDR. When the `destSubnet` field of the EgressGatewayPolicy is empty, the data plane will automatically match traffic outside the CIDR in the EgressClusterStatus CR and forward it to the Egress gateway.

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressClusterInfo
metadata:
  name: "default"    # 1
spec: {}
status:
  egressIgnoreCIDR:  # 2
    clusterIP:       # 3
      ipv4:
      - "172.41.0.0/16"
      ipv6:
      - "fd41::/108"
    nodeIP:
      ipv4:
      - "172.18.0.3"
      - "172.18.0.4"
      - "172.18.0.2"
      ipv6:
      - "fc00:f853:ccd:e793::3"
      - "fc00:f853:ccd:e793::4"
      - "fc00:f853:ccd:e793::2"
    podCIDR:
      ipv4:
      - "172.40.0.0/16"
      ipv6:
      - "fd40::/48"
```

1. The name defaults to `default`, maintained by the system, only one can be created, and it cannot be modified.
2. `egressIgnoreCIDR` defines the CIDR that EgressGateway should ignore.
3. `clusterIP` is the default service-cluster-ip-range for the cluster. Whether it is enabled is specified by the EgressGateway configuration file's default `egressIgnoreCIDR.autoDetect.clusterIP`.
4. `nodeIP` is the collection of IP addresses for the cluster nodes (only taking the IP from the Node yaml `status.address`, in the case of multiple network cards, other network card IPs are treated as external IPs). Whether it is enabled is specified by the EgressGateway configuration file's default `egressIgnoreCIDR.autoDetect.nodeIP`.
5. `podCIDR` is the CIDR used by the cluster's CNI. It is specified by the egressgateway configuration file's default `egressIgnoreCIDR.autoDetect.podCIDR`.