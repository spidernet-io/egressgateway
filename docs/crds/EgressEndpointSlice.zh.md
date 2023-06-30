EgressEndpointSlice CRD 用于聚合 EgressPolicy 所匹配中的 Pods 地址信息，此资源仅供内部使用，用于提升控制面的性能。

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressEndpointSlice
metadata:
  generateName: ns-policy-
  labels:
    spidernet.io/policy-name: ns-policy          # (1)
  name: ns-policy-zp667
  namespace: default
  ownerReferences:
    - apiVersion: egressgateway.spidernet.io/v1beta1  # (2)
      blockOwnerDeletion: true
      controller: true
      kind: EgressPolicy
      name: ns-policy
      uid: fdca1dd5-9c3b-4d58-b043-451e10f15ea8
endpoints:                                             # (3)
  - ipv4:
      - 10.21.60.74                                    # (4)
    ipv6:
      - fd00:21::5328:9c2:3579:8cca                    # (5)
    node: workstation3                                 # (6)
    ns: ns1                                            # (7)
    pod: ns2-mock-app-5c4cd6bb87-g4fdj                 # (8)
```

1. 此标签值表示 EgressEndpointSlice 所属的 EgressPolicy。
2. 通过使用 `ownerReferences`，该 CRD 与其父资源关联，实现 EgressPolicy 删除时自动回收 EgressEndpointSlice 功能。
3. EgressEndpointSlice 对象用于汇总 EgressPolicy 匹配到的 Pods 地址信息，默认在超过 100 个匹配结果时，将创建新的 EgressEndpointSlice。
4. Pods 的 IPv4 地址列表。
5. Pods 的 IPv6 地址列表。
6. Pods 所在节点的信息。
7. Pods 所属租户的信息。
8. Pods 的名称。