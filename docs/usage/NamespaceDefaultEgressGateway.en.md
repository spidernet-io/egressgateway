# Namespace Level Default EgressGateway

## Introduction

Setting a default EgressGateway for a namespace simplifies the process of specifying the EgressGateway name when using EgressPolicy under the namespace. The priority of the namespace level default EgressGateway is higher than that of the cluster level. In other words, when a namespace level default gateway is specified, the tenant's default settings will be used first. Otherwise, the cluster's default settings will be used.

## Prerequisites

- EgressGateway component is installed.
- An EgressGateway CR has been created.

## Steps

1. Use the following command to specify the default EgressGateway name for the tenant:

    ```bash
    kubectl label ns default spidernet.io/egressgateway-default=egressgateway
    ```

2. Use the following definition to create an EgressPolicy, ignoring the definition of the `spec.egressGatewayName` field:

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

3. Run the following command again to confirm that the EgressPolicy has been set to the default EgressGateway:

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