# 从 AWS Marketplace 安装 EgressGateway

本文介绍了如何通过 AWS Marketplace 安装由 DaoCloud 提供支持的 EgressGateway。该服务像对于开源版本，采用按月付费模式，用户可享受更优质的服务与更全面的技术支持，功能与开源版本一致。

EgressGateway 支持多个 Node 作为 Pod 的高可用（HA）出口网关，你可以通过 EgressGateway 来节省公网 IP 费用（AWS 的 NAT Gateway 定价为 $0.045/小时，每月成本约为 $32.4），同时实现对需要访问外部网络的 Pod 进行精细化控制。

## 订阅 EgressGateway

访问 [EgressGateway AWS Marketplace](https://aws.amazon.com/marketplace/pp/prodview-b5ip2fo7qduma/) 页面进行订阅。

## 创建 Kubernetes 集群

```shell
export NAME="k8s-$RANDOM"
cat <<EOF >eks-config.yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig

metadata:
  name: ${NAME}
  region: eu-west-1

managedNodeGroups:
- name: ng-1
  desiredCapacity: 2
  privateNetworking: true
  # taint nodes so that application pods are
  # not scheduled/executed until Cilium is deployed.
  # Alternatively, see the note below.
  taints:
   - key: "node.cilium.io/agent-not-ready"
     value: "true"
     effect: "NoExecute"
EOF
eksctl create cluster -f ./eks-config.yaml
cilium install
cilium status --wait
```

## 安装 EgressGateway

```shell
export HELM_EXPERIMENTAL_OCI=1
aws ecr get-login-password --region us-east-1 | helm registry login --username AWS --password-stdin 709825985650.dkr.ecr.us-east-1.amazonaws.com
mkdir awsmp-chart && cd awsmp-chart
helm pull oci://709825985650.dkr.ecr.us-east-1.amazonaws.com/daocloud-hong-kong/egressgateway --version 0.0.2
tar xf $(pwd)/* && find $(pwd) -maxdepth 1 -type f -delete
helm install --generate-name --namespace <ENTER_NAMESPACE_HERE> ./*
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
