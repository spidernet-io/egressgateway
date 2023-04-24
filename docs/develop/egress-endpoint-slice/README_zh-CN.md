# EgressEndpointSlice

## 简介
聚合 EgressGatewayPolicy 匹配中的端点，以提高扩展性，仅支持 EgressGatewayPolicy 使用 `podSelector` 的方式匹配 Pod 的情况。每个 EgressEndpointSlice 中的 Endpoint 个数默认不超过 100，最大值可以进行设置。是 EgressGatewayPolicy 的附属资源。

## CRD

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressEndpointSlice
metadata:
  name: "policy-test-dx66t"     # 1
  namespace: "default"         
  labels:
    egressgateway.spidernet.io/egressgatewaypolicy: "policy-test"  # 2
  ownerReferences:   # 3
  - apiVersion: egressgateway.spidernet.io/v1beta1
    blockOwnerDeletion: true
    controller: true
    kind: EgressGatewayPolicy
    name: "policy-test"
    uid: 1b2ec0a8-b929-4528-8f99-499f981d319e
data:
  endpoints:                   # 4
  - podName: "web-app"         
    ipv4List:
    - "172.29.30.123" 
    ipv6List:
    - "xxx"         
    nodeName: "node1"          # 5
    uuid: ""
```

1. 名称由 `policy-name-xxxxx` 组成，后面 5 位随机生成；
2. 所属的 EgressGatewayPolicy 名称；
3. 创建时同步设置 ownerReferences；
4. 匹配中的 endpoints 的列表；
5. Pod 所在的节点。

## 代码设计

### 初始化

### Controller

### agent

## 其他