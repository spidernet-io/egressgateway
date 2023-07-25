EgressClusterInfo CRD 为了简化 Egress 策略的配置，引入 Egress Ignore CIDR 功能，允许自动获取集群的 CIDR。当 EgressGatewayPolicy 的 `destSubnet` 字段为空时，数据面将会自动匹配 EgressClusterStatus CR 中的 CIDR 之外的流量，并将其转发到 Egress 网关。

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

1. 名称为 `default`，由系统维护只能创建一个;
2. `clusterIP`，如果设置为 `true`，`Service CIDR` 会自动检测
3. `nodeIP`，如果设置为 `true`，会自动检测 `nodeIP` 相关变化，并动态更新到 `EgressClusterInfo` 的 `status.nodeIP` 中
4. `podCidrMode`，目前支持 `k8s`、 `calico`、`auto`、 `""`，表示要自动检测对应的 podCidr，默认为 `auto`，如果为 `auto` 表示自动检测集群使用的 cni， 如果检测不到，则使用 集群的 podCidr。如果为 `""` 表示不检测
5. `extraCidr`，可手动填写要忽略掉的 `IP` 集合
6. `status.clusterIP`，如果 `spec.autoDetect.clusterIP` 为 `true`，则自动检测集群 `Service CIDR`，并更新到此处
7. `status.extraCidr`，对应 `spec.extraCidr` 
8. `status.nodeIP`，如果 `spec.autoDetect.nodeIP` 为 `true`，则自动检测集群 `nodeIP`，并更新到此处
9. `status.podCIDR`，对应 `spec.autoDetect.podCidrMode`，进行相关 `podCidr` 的更新
10. `status.podCidrMode`，对应 `spec.autoDetect.podCidrMode` 为 `auto` 的场景
