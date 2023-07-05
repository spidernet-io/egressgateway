## 要求

Egressgateway 目前兼容 Calico CNI，并将在未来支持更多 CNI。 以下是不同 CNI 的配置方法。

### Calico

将 FelixConfiguration 中 `chainInsertMode` 的设置更改为 `Append`，更多参考 [calico 文档](https://projectcalico.docs.tigera.io/reference/resources/felixconfig)：

```yaml
apiVersion: projectcalico.org/v3
kind: FelixConfiguration
metadata:
  name: default
spec:
  ipv6Support: false
  ipipMTU: 1400
  chainInsertMode: Append # (1)
```

1. 更改此行

## 安装

### 添加 helm 仓库

```shell
helm repo add egressgateway https://spidernet-io.github.io/egressgateway/
helm repo update
```

### 安装 egressgateway

以下是一个常用的的 Helm Chart `Values.yaml` 设置选项：

```yaml
feature:
  enableIPv4: true
  enableIPv6: false # (1)
  tunnelIpv4Subnet: "192.200.0.1/16" # (2)
  tunnelIpv6Subnet: "fd01::21/112"   # (3)
```

1. 如果需要 IPv6 则开启该选项，该选项要求 Pod 的网络堆栈是 IPv6
2. IPv4 隧道子网
3. IPv6 隧道子网

```shell
helm install egressgateway egressgateway/egressgateway \
  --values values.yaml \
  --wait --debug
```

```shell
kubectl get crd | grep egress
```

## 创建 EgressGateway CR 对象

创建一个 EgressGateway CR，通过 `matchLabels` 可以将节点设置为出口网关节点。

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressGateway
metadata:
  name: default
spec:
  ippools:
    ipv4:
    - "10.6.1.60-10.6.1.66" # (1)
  nodeSelector:
    selector:
      matchLabels:
        kubernetes.io/hostname: workstation2 # (2)
```

1. EgressGateway 可以使用 Egress IP 地址范围
2. 通过 label 选择一个或者一组节点作为出口网关，本组网关用于生效 Egress IP

## 创建一个测试 Pod 应用

创建一个测试 Pod，以模拟需要出口 Egress IP 的应用程序。

```yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: mock-app
  name: mock-app
  namespace: default
spec:
  containers:
   - image: nginx
     imagePullPolicy: IfNotPresent
     name: nginx
     resources: {}
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: workstation1 # (1)
```

1. 更改 `nodeName` 的值，选择一个非出口网关的节点。

## 创建 EgressPolicy CR 对象

通过创建 EgressPolicy CR，您可以控制哪些 Pod 在访问特定地址时需要经过出口网关。

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressPolicy
metadata:
  name: mock-app
spec:
  egressGatewayName: "default" # (1)
  appliedTo:
    podSelector:
      matchLabels:             # (2)
        app: mock-app
  destSubnet:
    - 10.6.1.92/32             # (3)
```

1. 通过设置此值，选取上述创建的名为 `default` 的 EgressGateway 网关。
2. 通过设置 `matchLabels` 来选择需要进行 Egress 操作的 Pod。
3. 通过设置 `destSubnet`，可以使匹配的 Pod 在访问特定子网时才进行 Egress 操作。

现在，来自 mock-app 访问 10.6.1.92 的流量将通过出口网关转发。

## 测试结果

我们可以看到 mock-app 访问外部服务时对端看到的 IP 是 EgressGateway 的 IP 地址。

```shell
kubectl exec -it mock-app bash
$ curl 10.6.1.92:8080
Remote IP: 10.6.1.60
```
