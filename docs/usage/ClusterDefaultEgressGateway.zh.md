# 集群级默认 EgressGateway

## 介绍

为整个集群设置默认 EgressGateway，可以简化在租户下使用 EgressPolicy 或在集群级使用 EgressClusterPolicy 时，每次指定 EgressGateway 名称的步骤。注意集群默认 EgressGateway 只能设置一个。

## 实施要求

* 已安装 EgressGateway 组件

## 步骤

1. 创建 EgressGateway 时可以通过设置 `spec.clusterDefault` 为 `true`，将其指定为集群的默认 EgressGateway，在 EgressClusterPolicy 没有指定 `spec.egressGatewayName` 时，以及 EgressPolicy 没有指定 `spec.egressGatewayName` 且租户没有配置默认 EgressGateway 时，自动使用集群默认的 EgressGateway。

    ```yaml
    apiVersion: egressgateway.spidernet.io/v1beta1
    kind: EgressGateway
    metadata:
      name: default
    spec:
      clusterDefault: true
      ippools:
        ipv4:
          - 10.6.1.55
          - 10.6.1.56
        ipv4DefaultEIP: 10.6.1.55
        ipv6:
          - fd00::55
          - fd00::56
        ipv6DefaultEIP: fd00::56
      nodeSelector:
        selector:
          matchLabels:
            egress: "true"    
    ```

2. 使用以下定义创建 EgressPolicy，忽略 `spec.egressGatewayName` 字段的定义：

    ```yaml
    apiVersion: egressgateway.spidernet.io/v1beta1
    kind: EgressPolicy
    metadata:
      name: mock-app
    spec:
      appliedTo:
        podSelector:
          matchLabels:
            app: mock-app
      destSubnet:
        - 10.6.1.92/32
    ```

3. 再次运行以下命令，确认 EgressPolicy 已被设置为默认的 EgressGateway：

    ```shell
    $ kubectl get egresspolicies mock-app -o yaml
    apiVersion: egressgateway.spidernet.io/v1beta1
    kind: EgressPolicy
    metadata:
      creationTimestamp: "2023-08-09T11:54:34Z"
      generation: 1
      name: mock-app
      namespace: default
      resourceVersion: "6233341"
      uid: 5692c5e6-a72b-41bd-a611-1106abd41bc2
    spec:
      appliedTo:
        podSelector:
          matchLabels:
            app: mock-app
      destSubnet:
      - 10.6.1.92/32
      - fd00::92/128
      - 172.30.40.0/21
      egressGatewayName: default
    ```
