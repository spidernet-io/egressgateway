EgressClusterInfo CRD 为了简化 Egress 策略的配置，引入 Egress Ignore CIDR 功能，允许自动获取集群的 CIDR。当 EgressGatewayPolicy 的 `destSubnet` 字段为空时，数据面将会自动匹配 EgressClusterStatus CR 中的 CIDR 之外的流量，并将其转发到 Egress 网关。

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

1. 名称默认为 `default`，由系统维护，只能创建一个，不可被修改。
2. `egressIgnoreCIDR` 定义 EgressGateway 要忽略的 CIDR。
3. `clusterIP` 集群默认的 service-cluster-ip-range。是否开启，由 EgressGateway 配置文件默认的 `egressIgnoreCIDR.autoDetect.clusterIP` 指定。
4. `nodeIP` 集群节点的 IP（只取 Node yaml `status.address` 中的 IP，多卡情况下，其他网卡 IP 被视作集群外 IP 处理）集合。是否开启，由 EgressGateway 配置文件默认的 `egressIgnoreCIDR.autoDetect.nodeIP` 指定。
5. `podCIDR` 集群的 CNI 使用的 CIDR。由 egressgateway 配置文件默认的 `egressIgnoreCIDR.autoDetect.podCIDR` 指定。
