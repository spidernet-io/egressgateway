# 出口网关节点间迁移 Egress IP

## 使用场景

* 我们通过 EgressGateway 可以选择多个 Node 作为 EgressNode，当 Node 需要维护时， 可以通过 cli 命令手动迁移该 Node 的 vip 到另外一个 Node。
* 其他原因，需要手动将某个 Node 的 VIP 到另外一个 Node 时。

## 使用步骤

我们执行 `kubectl get egw egressgateway -o yaml` 查看的 EgressGateway 定义。

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

在迁移前，Egress IP 在 workstation2 节点。

```shell
node@workstation:~$ kubecti get egp
NAME       GATEWAY          IPW4          IPV6       EGRESSNODE
policy-1   egressgateway    10.6.92.209              workstation3
```

我们通过执行下面命令将 `workstation3` 的 Egress IP 迁移到  `workstation2` Node。

```log
node@workstation:~$ kubectl exec -it egressgateway-controller-86c84f4858-b6dz4 bash
egctl vip move --egressGatewayName egressgateway --vip 10.6.92.209 --targetNode workstation2
Moving VIP 10.6.92.209 to node workstation2...
Successfully moved VIP 10.6.92.209 to node workstation2
```

迁移后 Egress IP 节点已经转移到 workstation2 节点。

```shell
node@workstation:~$ kubecti get egp
NAME       GATEWAY          IPW4          IPV6       EGRESSNODE
policy-1   egressgateway    10.6.92.209              workstation2
```