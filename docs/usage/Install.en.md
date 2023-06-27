## Requirement

Egressgateway currently supports collaboration with Calico CNI and will support collaboration with more CNIs in the future. 
Below are the configuration methods for different CNIs:

### Calico

Required settings `chainInsertMode` to `Append`, for example in the code, 
more reference [calico docs](https://projectcalico.docs.tigera.io/reference/resources/felixconfig):

```yaml
apiVersion: projectcalico.org/v3
kind: FelixConfiguration
metadata:
  name: default
spec:
  ipv6Support: false
  ipipMTU: 1400
  chainInsertMode: Append # this line
```

## Install

### Add helm repository

```shell
helm repo add egressgateway https://spidernet-io.github.io/egressgateway/
helm repo update
```

### Install egressgateway

The following is a common chart setting option:

```yaml
feature:
  enableIPv4: true
  enableIPv6: false # Required pod support IPv6 Stack
  tunnelIpv4Subnet: "192.200.0.1/16" # IPv4 tunnel subnet
  tunnelIpv6Subnet: "fd01::21/112"   # IPv6 tunnel subnet
  forwardMethod: "active-active"     # Support active-active or active-passive
```

```shell
helm install egressgateway egressgateway/egressgateway --values values.yaml --wait --debug
```

```shell
kubectl get crd | grep egress
```

## Create EgressGateway

Create an EgressGateway CR that can set a node as an egress gateway node through matchLabels.

```shell
apiVersion: egressgateway.spidernet.io/v1
kind: EgressGateway
metadata:
  name: default
spec:
  nodeSelector:
    matchLabels:
      kubernetes.io/hostname: workstation2 # change me, select a node in your cluster
```

## Create Example App

Create a testing Pod to simulate an application that requires egress.

```shell
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: mock-app
  name: mock-app
  namespace: default
spec:
  containers:
      image: nginx
      imagePullPolicy: IfNotPresent
      name: nginx
      resources: {}
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: workstation1 # change me, select a non-egress gateway node in your cluster
```

## Create EgressGatewayPolicy

By creating an EgressGatewayPolicy CR, you can control which Pod accesses which address needs to go through the egress gateway.

```yaml
apiVersion: egressgateway.spidernet.io/v1
kind: EgressGatewayPolicy
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

Now traffic from mock-app accessing 10.6.1.92 will be forwarded by the egress gateway.
