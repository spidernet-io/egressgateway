The EgressClusterInfo CRD introduces the Egress Ignore CIDR feature to simplify the configuration of Egress policies and allows automatic acquisition of the cluster's CIDR. When the `destSubnet` field of the EgressGatewayPolicy is empty, the data plane will automatically match traffic outside the CIDR in the EgressClusterStatus CR and forward it to the Egress gateway.

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

1. The name is `default`, only one can be created by the system maintenance;
2. `clusterIP`, if set to `true`, `Service CIDR` will be detected automatically
3. `nodeIP`, if it is set to `true`, it will automatically detect changes related to `nodeIP` and dynamically update it to `status.nodeIP` of `EgressClusterInfo`
4. `podCidrMode` currently supports `k8s`, `calico`, `auto`, and `""`. It indicates whether to automatically detect the corresponding `podCidr` setting. The default value is `auto`. When set to `auto`, it means that the cluster's used CNI (Container Network Interface) will be automatically detected. If detection fails, the cluster's `podCidr` will be used. If set to `""`, it signifies no detection.
5. `extraCidr`, you can manually fill in the `IP` set to be ignored
6. `status.clusterIP`, if `spec.autoDetect.clusterIP` is `true`, then automatically detect the cluster `Service CIDR`, and update here
7. `status.extraCidr`, corresponding to `spec.extraCidr`
8. `status.nodeIP`, if `spec.autoDetect.nodeIP` is `true`, then automatically detect cluster `nodeIP`, and update here
9. `status.podCIDR`, corresponding to `spec.autoDetect.podCidrMode`, update related `podCidr`
10. `status.podCidrMode` corresponding to `spec.autoDetect.podCidrMode` being set to `auto`