# 在 AWS Cilium CNI 下使用 EgressGateway

## 介绍

本文介绍了在 AWS Kubernetes 的 Cilium CNI 网络环境下，运行 EgressGateway。EgressGateway 支持多个 Node 作为 Pod 的高可用（HA）出口网关，你可以通过 EgressGateway 来节省公网 IP 费用，同时实现对需要访问外部网络的 Pod 进行精细化控制。

EgressGateway 相对于 Cilium 的 Egress 功能，支持 HA 高可用。如果你没有此需要，应当先考虑使用 Cilium 的 Egress 功能。

接下来的章节将逐步引导您安装 EgressGateway，创建一个示例 Pod，并为该 Pod 配置 Egress 策略，使其通过出口网关节点访问互联网。

## 创建集群及安装 Cilium

参考 [Cilium 安装指南](https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default) 文档创建 AWS 集群，并安装 Cilium。 编写本文时，使用的 Cilium 版本为 1.15.6，如果您在其他版本出现非预期情况，请和我们反馈。

你创建的 Kubernetes 集群时，加入的 EC2 节点要具备[公网 IP](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-instance-addressing.html)。你可以 `ssh root@host` 到您的节点进行测试。

```shell
curl ipinfo.io
```

通过 curl 您可以看到返回结果包含你 Node 的公网 IP。


## 安装 EgressGateway

添加和更新 Helm 仓库以从指定来源安装 EgressGateway。

```shell
helm repo add egressgateway https://spidernet-io.github.io/egressgateway/
helm repo update
```

我们  `feature.enableIPv4=true` 启用 IPv4 ，通过 `feature.enableIPv6=false` 禁用 IPv6。在安装过程中，我们可以通过 ``feature.clusterCIDR.extraCidr`` 集群的内部 CIDR，这将修改 `EgressPolicy` 的行为。如果您创建一个 `EgressPolicy` CR 并且没有指定 `spec.destSubnet`，EgressGateway 将把 Pod 的所有访问外部的流量（内部 CIDR 除外）转发到网关节点。相反，如果指定了 `spec.destSubnet`，EgressGateway 将仅将指定的流量转发到网关节点。

```shell
helm install egress --wait \
 --debug egressgateway/egressgateway \
 --set feature.enableIPv4=true \
 --set feature.enableIPv6=false \
 --set feature.clusterCIDR.extraCidr[0]=172.16.0.0/16
```

## 创建 EgressGateway CR

查看当前节点。

```shell
~ kubectl get nodes -A -owide
NAME                             STATUS   ROLES    AGE   VERSION               INTERNAL-IP      EXTERNAL-IP                         
ip-172-16-103-117.ec2.internal   Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.103.117   34.239.162.85  
ip-172-16-61-234.ec2.internal    Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.61.234    54.147.15.230
ip-172-16-62-200.ec2.internal    Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.62.200    54.147.16.130  
```

我们选择 `ip-172-16-103-117.ec2.internal` 和 `ip-172-16-62-200.ec2.internal` 作为网关节点。给节点设置 `egress=true` 标签。

```shell
kubectl label node ip-172-16-103-117.ec2.internal role=gateway
kubectl label node ip-172-16-62-200.ec2.internal role=gateway
```

创建 EgressGateway CR，我们通过 `role: gateway` 来选择节点作为出口网关。

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressGateway
metadata:
  name: "egressgateway"
spec:
  nodeSelector:
    selector:
      matchLabels:
        role: gateway
```

## 创建测试 Pod

查看当前节点。

```shell
~ kubectl get nodes -A -owide
NAME                             STATUS   ROLES    AGE   VERSION               INTERNAL-IP      EXTERNAL-IP                         
ip-172-16-103-117.ec2.internal   Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.103.117   34.239.162.85  
ip-172-16-61-234.ec2.internal    Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.61.234    54.147.15.230
ip-172-16-62-200.ec2.internal    Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.62.200    54.147.16.130  
```

我们选择 ip-172-16-61-234.ec2.internal 节点运行 Pod。

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: mock-app
  labels:
    app: mock-app
spec:
  nodeName: ip-172-16-61-234.ec2.internal
  containers:
  - name: nginx
    image: nginx
```

查看确保 Pods 处于 Running 状态。

```shell
~ kubectl get pods -o wide
NAME                                        READY   STATUS    RESTARTS   AGE   IP               NODE                             NOMINATED NODE   READINESS GATES
egressgateway-agent-zw426                   1/1     Running   0          15m   172.16.103.117   ip-172-16-103-117.ec2.internal   <none>           <none>
egressgateway-agent-zw728                   1/1     Running   0          15m   172.16.61.234    ip-172-16-61-234.ec2.internal    <none>           <none>
egressgateway-controller-6cc84c6985-9gbgd   1/1     Running   0          15m   172.16.51.178    ip-172-16-61-234.ec2.internal    <none>           <none>
mock-app                                    1/1     Running   0          12m   172.16.51.74     ip-172-16-61-234.ec2.internal    <none>           <none>
```

## 创建 EgressPolicy CR

我们创建下面 YAML，EgressGateway CR，我们使用 `spec.podSelector` 来匹配上面创建的 Pod。`spec.egressGatewayName` 则制定了了我们上面创建的网关。
使用 `spec.egressIP.useNodeIP` 来指定使用节点的 IP 作为访问互联网的地址。

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressPolicy
metadata:
  name: test-egw-policy
  namespace: default
spec:
  egressIP:
    useNodeIP: true
  appliedTo:
    podSelector:
      matchLabels:
        app: mock-app
  egressGatewayName: egressgateway
```

### 测试出口 IP 地址

使用 exec 进入容器，运行 `curl ipinfo.io`，你可以看到当前节点的 Pod 已经使用网关节点访问互联网，`ipinfo.io` 会回显主机 IP。
由于 EgressGateway 使用主备实现 HA，当 EIP 节点发生切换时，Pod 会自动切换到匹配的备用节点，同时出口 IP 也会发生变化。

```shell
kubectl exec -it -n default mock-app bash
curl ipinfo.io
{
  "ip": "34.239.162.85",
  "hostname": "ec2-34-239-162-85.compute-1.amazonaws.com",
  "city": "Ashburn",
  "region": "Virginia",
  "country": "US",
  "loc": "39.0437,-77.4875",
  "org": "AS14618 Amazon.com, Inc.",
  "postal": "20147",
  "timezone": "America/New_York",
  "readme": "https://ipinfo.io/missingauth"
}
```