# Install EgressGateway from AWS Marketplace

This article introduces how to install the EgressGateway supported by DaoCloud via AWS Marketplace. Compared to the open-source version, this service adopts a monthly subscription model, allowing users to enjoy higher-quality services and more comprehensive technical support while maintaining the same functionality as the open-source version.

EgressGateway supports multiple Nodes as highly available (HA) egress gateways for Pods. With EgressGateway, you can save on public IP costs (AWS NAT Gateway is priced at $0.045/hour, amounting to approximately $32.4 per month) while enabling fine-grained control over Pods that need access to external networks.

## Purchase EgressGateway

Visit [EgressGateway AWS Marketplace](https://aws.amazon.com/marketplace/pp/prodview-b5ip2fo7qduma/) and subscribe to the product.

## Setup a Kubernetes cluster

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

## Install EgressGateway

```shell
export HELM_EXPERIMENTAL_OCI=1
aws ecr get-login-password --region us-east-1 | helm registry login --username AWS --password-stdin 709825985650.dkr.ecr.us-east-1.amazonaws.com
mkdir awsmp-chart && cd awsmp-chart
helm pull oci://709825985650.dkr.ecr.us-east-1.amazonaws.com/daocloud-hong-kong/egressgateway --version 0.0.2
tar xf $(pwd)/* && find $(pwd) -maxdepth 1 -type f -delete
helm install --generate-name --namespace <ENTER_NAMESPACE_HERE> ./*
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
