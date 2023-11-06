# Cluster Level Default EgressGateway

## Introduction

Setting a default EgressGateway for the entire cluster can simplify the process of using EgressPolicy under a namespace or using EgressClusterPolicy at the cluster level, as it eliminates the need to specify the EgressGateway name each time. Please note that only one default EgressGateway can be set for the cluster.

## Prerequisites

- EgressGateway component is installed.

## Steps

1. When creating an EgressGateway, you can specify `spec.clusterDefault` as `true` to make it the default EgressGateway for the cluster. If `spec.egressGatewayName` is not specified in EgressClusterPolicy, and `spec.egressGatewayName` is not specified in EgressPolicy and the tenant has not configured a default EgressGateway, the cluster's default EgressGateway will be automatically used.

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