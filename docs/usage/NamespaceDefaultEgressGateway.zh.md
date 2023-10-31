# 租户级默认 EgressGateway

## 介绍

为租户设置默认 EgressGateway 可以简化在租户下使用 EgressPolicy 时每次指定 EgressGateway 名称的步骤。租户级默认 EgressGateway 的优先级大于集群默认 EgressGateway，换句话说，当指定了租户级的默认网关，会优先使用租户默认设置，如果租户没有设置默认网关，则会使用集群默认设置。

## 实施要求

* 已安装 EgressGateway 组件
* 已创建一个 EgressGateway CR

## 步骤

1. 使用以下命令为租户指定默认的 EgressGateway 名称：
    
    ```bash
    kubectl label ns default spidernet.io/egressgateway-default=egressgateway
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
      creationTimestamp: "2023-08-09T10:54:34Z"
      generation: 1
      name: mock-app
      namespace: default
      resourceVersion: "6233341"
      uid: 5692c5e6-a71b-41bd-a611-1106abd41ba3
    spec:
      appliedTo:
        podSelector:
          matchLabels:
            app: mock-app
      destSubnet:
      - 10.6.1.92/32
      - fd00::92/128
      - 172.30.40.0/21
      egressGatewayName: egressgateway
    ```
