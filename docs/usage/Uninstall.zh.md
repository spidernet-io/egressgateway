为了确保在卸载 EgressGateway 之前不影响正在使用的业务应用，建议执行以下步骤：

1. 检查与 EgressGateway 相关的资源数量是否为 0。执行以下命令：

    ```shell
    kubectl get egressclusterpolicies.egressgateway.spidernet.io -o name | wc -l
    kubectl get egresspolicies.egressgateway.spidernet.io -o name | wc -l
    kubectl get egressgateways.egressgateway.spidernet.io -o name | wc -l
    ```
   
    这些命令将输出与 EgressGateway 相关的 EgressClusterPolicy、EgressPolicy 和 EgressGateway 资源的数量。如果输出结果为 0，则表示没有与 EgressGateway 相关联的资源。如果输出结果不为 0，则需要进一步处理，以确保卸载操作不会影响正在使用的业务应用。

    如果输出不为 0，你应该继续下面命令检查，否则跳转到到步骤 2。

    ```shell
    kubectl get egressclusterpolicies.egressgateway.spidernet.io
    kubectl get egresspolicies.egressgateway.spidernet.io -o wide
    kubectl get egressgateways.egressgateway.spidernet.io
    ```

    例如你发现 EgressPolicies 还有未删除资源时，应该查看资源详情

    ```shell
    kubectl get egresspolicies <resource-name> --namespace <resource-namespace> -o yaml
    ```

    ```yaml
    apiVersion: egressgateway.spidernet.io/v1beta1
    kind: EgressPolicy
    metadata:
      name: ns-policy
      namespace: default
    spec:
      appliedTo:
        podSelector:
          matchLabels:
            app: mock-app
      egressGatewayName: egressgateway
    status:
      eip:
        ipv4: 10.6.1.55
        ipv6: fd00::55
      node: workstation2
    ```

    通过检索 `appliedTo.podSelector` 匹配到的 Pod 确保删除不会影响业务应用时，在执行如下命令进行删除。

    ```shell
    kubectl delete egresspolicies <resource-name> --namespace <resource-namespace>
    ```
   
2. 查询当前集群安装的 EgressGateway。执行以下命令：

    ```shell
    helm ls -A | grep -i egress
    ```

    这将输出当前集群中安装的 EgressGateway 的名称、命名空间、版本等信息。

3. 卸载 EgressGateway。如果您确定要卸载 EgressGateway，可以执行以下命令：

    ```shell
    helm uninstall <egressgateway-name> --namespace <egressgateway-namespace>
    ```

    将 `<egressgateway-name>` 替换为要卸载的 EgressGateway 的名称，将 `<egressgateway-namespace>` 替换为 EgressGateway 所在的命名空间。

    需要注意的是，在卸载 EgressGateway 之前，建议先备份相关数据，并确保卸载操作不会影响正在使用的业务应用。

4. 在卸载过程中，有时候会遇到 EgressGateway 的 EgressTunnels CRD 一直处于等待删除的情况。如果您遇到了这种情况，可以尝试使用下面的命令解决问题：

    ```shell
    kubectl patch crd egresstunnels.egressgateway.spidernet.io -p '{"metadata":{"finalizers": []}}' --type=merge
    ```

    这个命令的作用是删除 EgressGateway CRD 中的 finalizer，从而允许 Kubernetes 删除这个 CRD。此问题是由 controller-manager 引起的，我们正在关注 Kubernetes 团队对此问题的修复情况。

