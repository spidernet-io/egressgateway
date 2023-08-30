To ensure that the running applications are not affected before uninstalling EgressGateway, it is recommended to perform the following steps:

1. Check if the number of resources related to EgressGateway is 0. Run the following commands:

    ```shell
    kubectl get egressclusterpolicies.egressgateway.spidernet.io -o name | wc -l
    kubectl get egresspolicies.egressgateway.spidernet.io -o name | wc -l
    kubectl get egressgateways.egressgateway.spidernet.io -o name | wc -l
    ```

    These commands will output the number of EgressGateway-related EgressClusterPolicy, EgressPolicy, and EgressGateway resources. If the output result is 0, it means there are no resources associated with EgressGateway. If the output result is not 0, further processing is needed to ensure that the uninstall operation does not affect the ongoing business applications.

    If the output is not 0, you should continue with the following commands. Otherwise, skip to step 2.

    ```shell
    kubectl get egressclusterpolicies.egressgateway.spidernet.io
    kubectl get egresspolicies.egressgateway.spidernet.io -o wide
    kubectl get egressgateways.egressgateway.spidernet.io
    ```
   
    For example, if you find there are still resources of EgressPolicies not deleted, you should check the resource details:

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
   
    Ensure that deleting will not affect business applications by searching for `appliedTo.podSelector`, then execute the following command to delete:

    ```shell
    kubectl delete egresspolicies <resource-name> --namespace <resource-namespace>
    ```

2. Query the EgressGateway installed in the current cluster. Run the following command:

    ```shell
    helm ls -A | grep -i egress
    ```

    This will output the name, namespace, version, and other information of the EgressGateway installed in the current cluster.

3. Uninstall EgressGateway. If you are sure you want to uninstall EgressGateway, you can run the following command:

    ```shell
    helm uninstall <egressgateway-name> --namespace <egressgateway-namespace>
    ```

    Replace `<egressgateway-name>` with the name of the EgressGateway you want to uninstall, and replace `<egressgateway-namespace>` with the namespace where EgressGateway is located.

    It is worth noting that before uninstalling EgressGateway, it is recommended to back up related data and ensure that the uninstall operation does not affect the ongoing business applications.

4. During the uninstallation process, sometimes the EgressTunnels CRD of EgressGateway may remain in a waiting state for deletion. If you encounter this situation, you can try using the following command to resolve the issue:

    ```shell
    kubectl patch crd egresstunnels.egressgateway.spidernet.io -p '{"metadata":{"finalizers": []}}' --type=merge
    ```

    This command removes the finalizer in the EgressGateway CRD, allowing Kubernetes to delete it. This issue is caused by the controller-manager, and we are monitoring the Kubernetes team's progress on fixing it.
