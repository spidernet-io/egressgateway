# Migration of Egress IP Between Gateway Nodes

## Use Cases

* With EgressGateway, we can select multiple Nodes as EgressNodes. When a Node requires maintenance, we can manually migrate the VIP of that Node to another Node using CLI commands.
* For other reasons, when it's necessary to manually move a Node's VIP to another Node.

## Steps for Use

We examine the definition of EgressGateway by executing `kubectl get egw egressgateway -o yaml`.

```yaml
apiVersion: egressgateway.spidernet.io/v1beta1
kind: EgressGateway
metadata:
  finalizers:
  - egressgateway.spidernet.io/egressgateway
  name: egressgateway
spec:
  ippools:
    ipv4:
    - 10.6.91.1-10.6.93.125
    ipv4DefaultEIP: 10.6.92.222
  nodeSelector:
    selector:
      matchLabels:
        egress: "true"
status:
  ipUsage:
    ipv4Free: 37
    ipv4Total: 637
    ipv6Free: 0
    ipv6Total: 0
  nodeList:
  - name: workstation2
    status: Ready
  - name: workstation3
    status: Ready
    eips:
    - ipv4: 10.6.92.209
      policies:
      - name: policy-1
        namespace: default
```

Before the migration, the Egress IP was on the workstation2 node.

```shell
node@workstation:~$ kubectl get egp
NAME       GATEWAY          IPV4          IPV6       EGRESS NODE
policy-1   egressgateway    10.6.92.209              workstation3
```

We migrate the Egress IP of `workstation3` to the `workstation2` Node by executing the command below.

```log
kubectl exec -it egressgateway-controller-86c84f4858-b6dz4 bash
egctl vip move --egressGatewayName egressgateway --vip 10.6.92.209 --targetNode workstation2
Moving VIP 10.6.92.209 to node workstation2...
Successfully moved VIP 10.6.92.209 to node workstation2
```

After migration, the Egress IP node has been moved to the workstation2 node.

```shell
node@workstation:~$ kubectl get egress
NAME       GATEWAY          IPV4          IPV6       EGRESS NODE
policy-1   egressgateway    10.6.92.209              workstation2
```