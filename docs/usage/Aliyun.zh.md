# 在阿里云中使用 EgressGateway

本文说明如何在阿里云中使用 EgressGateway。在阿里云中，由于阿里云的 IP（包括弹性公有 IP）和节点一一绑定，无法实现 Egress IP 在节点间漂移的功能。我们在下文中使用节点 IP （非指定 ippool 的方式）作为 Egress IP，使用节点 IP 作为 Egress IP 时，如果选择多个节点作为 Egress 网关以实现 HA 高可用时，若一个节点挂掉的时候，Egress IP 将会切换成另一个节点的 IP。

使用案例如下：

* 在 VPC 网络的东西向访问中，有 A 和 B 集群，集群 B 要求访问者的网络 IP 在白名单列表，因此在集群 A 部署 EgressGateway，使访问 B 集群的网络都是用 Egress IP，用此 IP 的流量会在外部应用特殊的策略。
* 在 VPC 网络南北向网络访问场景中，集群业务节有需要访问互联网，但业务节点不购买公有 IP，需要访问外部网络的 Pod 可通过集群内的 Egress 节点的绑定的公网 IP 实现连接外部网络。

## 要求

* Kubernetes 集群至少 2 个节点
* 已经安装 Calico 网络组件

## 安装 EgressGateway

安装前设置 Calico 的 iptables 模式为 Append。

如果您是通过 YAML 安装的 Calico，则应该执行下面命令：
```shell
kubectl set env daemonset -n calico-system calico-node FELIX_CHAININSERTMODE=Append
```

如果您是通过 Calico Operator 管理 Calico 则应该执行下面命令：
```shell
kubectl patch felixconfigurations  default --type='merge' -p '{"spec":{"chainInsertMode":"Append"}}'
```

添加 Helm 仓库。

```shell
helm repo add egressgateway https://spidernet-io.github.io/egressgateway/
helm repo update
```

通过 helm 安装 EgressGateway。

```shell
helm install egress --wait --debug egressgateway/egressgateway
```

检查所有 Pod 是否处于 Running 状态。

```shell
root@node1:~# kubectl get pods -A | grep egressgateway
default    egressgateway-agent-lkglz                  1/1     Running   0    86m
default    egressgateway-agent-s5xwk                  1/1     Running   0    86m
default    egressgateway-controller-6cd86df57-xm2d4   1/1     Running   0    86m
```

## 部署测试服务

我们新创建一台机器，作为 VPC 网络东西向的服务器，在这里我启动的机器 IP 为 `172.17.81.29`。

![new-vm](./new-vm.png)

运行下面命令启动测试服务器，他的功能是 `curl ip:8080`，它会返回客户端的 IP 地址，可以供我们检查 Egress IP 运作是否正常。

```shell
docker run -d --net=host ghcr.io/spidernet-io/egressgateway-nettools:latest /usr/bin/nettools-server -protocol web -webPort 8080
```

## 创建测试 Pod

查看我们当前集群的节点。

```shell
$ kubectl get nodes
NAME    STATUS   ROLES           AGE   VERSION
node1   Ready    control-plane   66m   v1.30.0
node2   Ready    <none>          66m   v1.30.0
```

在这里我们将 Pod，部署到 node1 节点，稍后我们将 EgressGateway 的能力实现，node1 的 Pod 跳到 node2 节点，并使用 node2 的 IP 访问外部网络。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  containers:
    - image: nginx
      imagePullPolicy: IfNotPresent
      name: nginx
      resources: {}
  nodeName: node1
```

查看 Pod 是否处于 Running 状态。

```shell
root@node1:~# kubectl get pods -o wide | grep nginx
nginx  1/1  Running  0  77m  10.200.166.133  node1  <none>  <none>
```

## 创建 EgressGateway CR

EgressGateway 的 CR 的作用是可以选择集群的一组节点作为 Egress 出口网关。在下面的定义中，`nodeSelector` 将匹配 node2 作为 Egress 网关。

```shell
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressGateway
metadata:
  name: "egressgateway"
spec:
  nodeSelector:
    selector:
      matchLabels:
        egress: "true"
```

## 选择一个节点作为 Egress 出口

查看我们当前集群的节点。我的 node2 的 Public IP 是 `8.217.200.161`。

```shell
$ kubectl get nodes
NAME    STATUS   ROLES           AGE   VERSION
node1   Ready    control-plane   66m   v1.30.0
node2   Ready    <none>          66m   v1.30.0
```

在这里我们 node2 打标签，以使其被我们上面的 EgressGateway 匹配中。

```shell
kubectl label node node2 egress=true
```

当使用 `kubectl label` 给节点打标签后，可以通过下面命令获取 EgressGateway CR 查看 `status.nodeList` 列表
是否存在刚才标记的 node2 的节点。

```shell
$ kubectl get egw egressgateway -o yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressGateway
metadata:
  name: egressgateway
spec:
  nodeSelector:
    selector:
      matchLabels:
        egress: "true"
status:
  nodeList:
  - name: node2
    status: Ready
```

## 创建 EgressPolicy

EgressPolicy CR 的作用是匹配 Pod，被匹配中的 Pod 的流量，会通过 Egress 网关离开集群。
在下面 EgressPolicy 的定义中，`34.117.186.192` 是 `ipinfo.io` 的地址，可以通过 `dig ipinfo.io` 获得。

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressPolicy
metadata:
  name: nginx-egress-policy
spec:
  egressGatewayName: egressgateway
  egressIP:
    useNodeIP: true
  appliedTo:
    podSelector:
      matchLabels:
        app: nginx
  destSubnet:
    - 172.17.81.29/32   # 东西向测试服务的 IP
    - 34.117.186.192/32 # ipinfo.io 的地址，用于集群南北向网络访问的测试
```

## 东西向网络访问测试

此时，我们使用 kubectl exec 命令进入 nginx Pod 进行测试。

```shell
$ curl 172.17.81.29:8080
Remote IP: 172.17.81.28:59022
```

我们看到返回结果是前面设置的 IP `172.17.81.28`，到这里 IP 作为 Egress 的实验就结束了。

## 南北向网络访问测试

测试 Pod 访问南北向网络的服务，我们可以看到 node1 的 Pod 使用 node2 的节点绑定的公网 IP 完成了互联网访问。

```shell
$ curl ipinfo.io
{
  "ip": "8.217.200.161",
  "city": "Hong Kong",
  "region": "Hong Kong",
  "country": "HK",
  "loc": "22.2783,114.1747",
  "org": "AS45102 Alibaba (US) Technology Co., Ltd.",
  "timezone": "Asia/Hong_Kong",
  "readme": "https://ipinfo.io/missingauth"
}
```
