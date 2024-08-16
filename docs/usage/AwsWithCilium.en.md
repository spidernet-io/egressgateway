#  EgressGateway with Cilium CNI on AWS

## Introduction

This article introduces the use of EgressGateway in a Cilium CNI networking environment on AWS Kubernetes. EgressGateway supports multiple nodes as high-availability (HA) exit gateways for pods. You can use EgressGateway to save on public IP costs while achieving fine-grained control over pods that need to access external networks.

Compared to Cilium's Egress feature, EgressGateway supports HA. If you don't need HA, consider using Cilium's Egress feature first.

The following sections will guide you step-by-step to install EgressGateway, create a sample Pod, and configure an EgressPolicy for the Pod to access the internet via the gateway node.

## Create Cluster and Install Cilium

Refer to the [Cilium Quick Installation Guide](https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default) to create an AWS cluster and install Cilium. At the time of writing, the Cilium version used is 1.15.6. If you encounter unexpected issues with other versions, please let us know.

Ensure that the EC2 nodes added to your Kubernetes cluster have [public IPs](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-instance-addressing.html). You can test this by `ssh root@host` into your node.

```shell
curl ipinfo.io
```

Using curl, you should see a response that includes your node's public IP.

## Install EgressGateway

Add and update the Helm repository to install egressgateway from the specified source.

```shell
helm repo add egressgateway https://spidernet-io.github.io/egressgateway/
helm repo update
```

We enable IPv4 with `feature.enableIPv4=true` and disable IPv6 with `feature.enableIPv6=false`. We can optionally specify `feature.clusterCIDR.extraCidr` the internal CIDR of the cluster during installation, which will modify the behavior of the `EgressPolicy`. If you create an `EgressPolicy` CR and do not specify `spec.destSubnet`, the EgressGateway will forward all traffic from the Pod, except for the internal CIDR, to the gateway node. Conversely, if `spec.destSubnet` is specified, the EgressGateway will only forward the designated traffic to the gateway node.

```shell
helm install egress --wait \
  --debug egressgateway/egressgateway \
  --set feature.enableIPv4=true \
  --set feature.enableIPv6=false \
  --set feature.clusterCIDR.extraCidr[0]=172.16.0.0/16
```

## Create EgressGateway CR

List the current nodes.

```shell
~ kubectl get nodes -A -owide
NAME                             STATUS   ROLES    AGE   VERSION               INTERNAL-IP      EXTERNAL-IP   
ip-172-16-103-117.ec2.internal   Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.103.117   34.239.162.85  
ip-172-16-61-234.ec2.internal    Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.61.234    54.147.15.230
ip-172-16-62-200.ec2.internal    Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.62.200    54.147.16.130  
```

We select `ip-172-16-103-117.ec2.internal` and `ip-172-16-62-200.ec2.internal` as gateway nodes. Label the nodes with `egress=true`.

```shell
kubectl label node ip-172-16-103-117.ec2.internal role=gateway
kubectl label node ip-172-16-62-200.ec2.internal role=gateway
```

Create the EgressGateway CR, using `role: gateway` to select nodes as exit gateways.

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

## Create a Test Pod

List the current nodes.

```shell
~ kubectl get nodes -A -owide
NAME                             STATUS   ROLES    AGE   VERSION               INTERNAL-IP      EXTERNAL-IP                         
ip-172-16-103-117.ec2.internal   Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.103.117   34.239.162.85  
ip-172-16-61-234.ec2.internal    Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.61.234    54.147.15.230
ip-172-16-62-200.ec2.internal    Ready    <none>   25m   v1.30.0-eks-036c24b   172.16.62.200    54.147.16.130  
```

We select `ip-172-16-61-234.ec2.internal` to run the pod.

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

Ensure the pods are in the Running state.

```shell
~ kubectl get pods -o wide
NAME                                        READY   STATUS    RESTARTS   AGE   IP               NODE                             NOMINATED NODE   READINESS GATES
egressgateway-agent-zw426                   1/1     Running   0          15m   172.16.103.117   ip-172-16-103-117.ec2.internal   <none>           <none>
egressgateway-agent-zw728                   1/1     Running   0          15m   172.16.61.234    ip-172-16-61-234.ec2.internal    <none>           <none>
egressgateway-controller-6cc84c6985-9gbgd   1/1     Running   0          15m   172.16.51.178    ip-172-16-61-234.ec2.internal    <none>           <none>
mock-app                                    1/1     Running   0          12m   172.16.51.74     ip-172-16-61-234.ec2.internal    <none>           <none>
```

## Create EgressPolicy CR

We create the following YAML for the EgressGateway CR. We use `spec.podSelector` to match the pod created above and `spec.egressGatewayName` to specify the gateway created earlier. `spec.egressIP.useNodeIP` specifies using the node's IP to access the internet.

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

### Test Exit IP Address

Use exec to enter the container and run `curl ipinfo.io`. You should see that the pod on the current node is accessing the internet through the gateway node, and `ipinfo.io` will return the host IP. Since EgressGateway implements HA using master-backup, when an EIP node switches, the pod will automatically switch to the matching backup node, and the exit IP will change accordingly.

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
