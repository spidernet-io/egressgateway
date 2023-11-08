# Install EgressGateway on a Self-managed Cluster

## Introduction

This page provides instructions for quickly installing EgressGateway on a self-managed Kubernetes cluster.

## Prerequisites

1. A self-managed Kubernetes cluster with a minimum of 2 nodes.

2. Helm has been installed in your cluster.

3. EgressGateway currently supports the following CNI plugins:

=== "Calico"

    If your cluster is using [Calico](https://www.tigera.io/project-calico/)  as the CNI plugin, run the following command to ensure that EgressGateway's iptables rules are not overridden by Calico rules. Failure to do so may cause EgressGateway to malfunction.

    ```shell
    # set chainInsertMode
    $ kubectl patch FelixConfiguration default --patch '{"spec": {"chainInsertMode": "Append"}}'
      
    # check status
    $ kubectl get FelixConfiguration default -o yaml
      apiVersion: crd.projectcalico.org/v1
        kind: FelixConfiguration
        metadata:
          generation: 2
          name: default
          resourceVersion: "873"
          uid: 0548a2a5-f771-455b-86f7-27e07fb8223d
          spec:
          chainInsertMode: Append
          ......
    ```

    > Regarding `spec.chainInsertMode`, refer to [Calico docs](https://projectcalico.docs.tigera.io/reference/resources/felixconfig) for details

=== "Flannel"

    [Flannel](https://github.com/flannel-io/flannel) CNI does not require any configuration, so you can skip this step.

=== "Weave"

    [Weave](https://github.com/flannel-io/flannel) CNI does not require any configuration, so you can skip this step.

=== "Spiderpool"

    If your cluster is using [Spiderpool](https://github.com/spidernet-io/spiderpool) in conjunction with another CNI, follow these steps:

    Add the service addresses outside the cluster to the 'hijackCIDR' field in the 'default' object of spiderpool.spidercoordinators. This ensures that when Pods access these external services, the traffic is routed through the host where the Pod is located, allowing the EgressGateway rules to match.

    ```
    # For running Pods, you need to restart them for these routing rules to take effect within the Pods.
    kubectl patch spidercoordinators default  --type='merge' -p '{"spec": {"hijackCIDR": ["1.1.1.1/32", "2.2.2.2/32"]}}'
    ```


## Install EgressGateway

### Add EgressGateway Repo

```shell
helm repo add egressgateway https://spidernet-io.github.io/egressgateway/
helm repo update
```

### Install EgressGateway

1. Quickly install EgressGateway through the following command:

    ```shell
    helm install egressgateway egressgateway/egressgateway \
		  -n kube-system \
			--set feature.tunnelIpv4Subnet="192.200.0.1/16" \
			--wait --debug
    ```

   In the installation command, please consider the following points:

   * Make sure to provide the IPv4 and IPv6 subnets for the EgressGateway tunnel nodes in the installation command. These subnets should not conflict with other addresses within the cluster.
   * You can customize the network interface used for EgressGateway tunnels by using the `--set feature.tunnelDetectMethod="interface=eth0"` option. By default, it uses the network interface associated with the default route.
   * If you want to enable IPv6 support, set the `--set feature.enableIPv6=true` option and also `feature.tunnelIpv6Subnet`.
   * The EgressGateway Controller supports high availability and can be configured using `--set controller.replicas=2`.
   * To enable return routing rules on the gateway nodes, use `--set feature.enableGatewayReplyRoute=true`. This option is required when using Spiderpool to work with underlay CNI.

2. Verify that all EgressGateway Pods are running properly.

    ```shell
    $ kubectl get pod -n kube-system | grep egressgateway
    egressgateway-agent-29lt5                  1/1     Running   0          9h
    egressgateway-agent-94n8k                  1/1     Running   0          9h
    egressgateway-agent-klkhf                  1/1     Running   0          9h
    egressgateway-controller-5754f6658-7pn4z   1/1     Running   0          9h
    ```

3. Any feature configurations can be achieved by adjusting the Helm values of the EgressGateway application.

## Create EgressGateway Instances

1. EgressGateway defines a group of nodes as the cluster's egress gateway, responsible for forwarding egress traffic out of the cluster. To define a group of EgressGateway, run the following command:

    ```shell
    cat <<EOF | kubectl apply -f -
    apiVersion: egressgateway.spidernet.io/v1beta1
    kind: EgressGateway
    metadata:
      name: default
    spec:
      ippools:
        ipv4:
        - "172.22.0.100-172.22.0.110"
      nodeSelector:
        selector:
          matchLabels:
            egressgateway: "true"
    EOF
    ```

   Descriptions:

   * In the provided YAML example, adjust `spec.ippools.ipv4` to define egress exit IP addresses based on your specific environment.
   * Ensure that the CIDR of `spec.ippools.ipv4` matches the subnet of the egress interface on the gateway nodes (usually the interface associated with the default route). Mismatched subnets can cause connectivity issues for egress traffic.
   * Use `spec.nodeSelector` in the EgressGateway to select a group of nodes as the egress gateway. You can select multiple nodes to achieve high availability.

2. Label the egress gateway nodes by applying labels to them. For production environments, it is recommended to label at least 2 nodes. For POC environments, label 1 node.

    ```shell
    kubectl label node $NodeName egressgateway="true"
    ```

3. Check the status:

    ```shell
    $ kubectl get EgressGateway default -o yaml
    apiVersion: egressgateway.spidernet.io/v1beta1
    kind: EgressGateway
    metadata:
      name: default
      uid: 7ce835e2-2075-4d26-ba63-eacd841aadfe
    spec:
      clusterDefault: true
      ippools:
        ipv4:
        - 172.22.0.100-172.22.0.110
        ipv4DefaultEIP: 172.22.0.110
      nodeSelector:
        selector:
          matchLabels:
            egressgateway: "true"
    status:
      nodeList:
      - name: egressgateway-worker1
        status: Ready
      - name: egressgateway-worker2
        status: Ready
    ```

   Descriptions:

   * The `status.nodeList` field indicates the nodes that match the `spec.nodeSelector`, along with the status of their corresponding EgressTunnel objects.
   * The `spec.ippools.ipv4DefaultEIP` field randomly selects one IP address from `spec.ippools.ipv4` as the default VIP for this group of EgressGateways. This default VIP is used when creating EgressPolicy objects for applications that do not specify a VIP address.

## Create Applications and Egress Policies

1. Create an application that will be used to test Pod access to external resources and apply labels to it.

    ```shell
    kubectl create deployment visitor --image nginx
    ```

2. Create an EgressPolicy CR object for your application.

   An EgressPolicy instance is used to define which Pods' egress traffic should be forwarded through EgressGateway nodes, along with other configuration details.
   You can create an example as follows. When a matching Pod accesses any external address in the cluster (excluding Node IP, CNI Pod CIDR, ClusterIP), it will be forwarded through EgressGateway nodes.
   Note that EgressPolicy objects are tenant-level, so they must be created under the tenant of the selected application.

    ```shell
    cat <<EOF | kubectl apply -f -
    apiVersion: egressgateway.spidernet.io/v1beta1
    kind: EgressPolicy
    metadata:
      name: test
      namespace: default
    spec:
      appliedTo:
        podSelector:
          matchLabels:
            app: "visitor"
    EOF
    ```

   Descriptions:

   * `spec.egressGatewayName` specifies the name of the EgressGateway group to use.
   * `spec.appliedTo.podSelector` determines which Pods within the cluster this policy should apply to.
   * There are two options for the source IP address of egress traffic in the cluster:
      * You can use the IP address of the gateway nodes. This is suitable for public clouds and traditional networks but has the downside of potential IP changes if a gateway node fails. You can enable this by setting `spec.egressIP.useNodeIP=true`.
      * You can use a dedicated VIP. EgressGateway uses ARP principles for VIP implementation, making it suitable for traditional networks rather than public clouds. The advantage is that the egress source IP remains fixed. If no settings are specified in the EgressPolicy, the default VIP of the egressGatewayName will be used, or you can manually specify `spec.egressIP.ipv4` , which must match the IP pool configured in the EgressGateway.

3. Check the status of the EgressPolicy

    ```shell
    $ kubectl get EgressPolicy -A
    NAMESPACE   NAME   GATEWAY   IPV4           IPV6   EGRESSTUNNEL
    default     test   default   172.22.0.110          egressgateway-worker2
     
    $ kubectl get EgressPolicy test -o yaml
    apiVersion: egressgateway.spidernet.io/v1beta1
    kind: EgressPolicy
    metadata:
      name: test
      namespace: default
    spec:
      appliedTo:
        podSelector:
          matchLabels:
            app: visitor
      egressIP:
        allocatorPolicy: default
        useNodeIP: false
    status:
      eip:
        ipv4: 172.22.0.110
      node: egressgateway-worker2
    ```

   Descriptions:

   * `status.eip` displays the egress IP address used by the group of applications.
   * `status.node` shows which EgressGateway node is responsible for real-time egress traffic forwarding. EgressGateway nodes support high availability. When multiple EgressGateway nodes exist, all EgressPolicy instances will be evenly distributed among them.

4. Check the status of EgressEndpointSlices.

   Each EgressPolicy object has a corresponding EgressEndpointSlices that stores the IP  collection of Pods selected by the EgressPolicy. If your application is unable to access external resources, you can check if the IP addresses in this object are correct.

    ```shell
    $ kubectl get egressendpointslices -A
    NAMESPACE   NAME         AGE
    default     test-kvlp6   18s
    
    $ kubectl get egressendpointslices test-kvlp6 -o yaml
    apiVersion: egressgateway.spidernet.io/v1beta1
    endpoints:
    - ipv4:
      - 172.40.14.195
      node: egressgateway-worker
      ns: default
      pod: visitor-6764bb48cc-29vq9
    kind: EgressEndpointSlice
    metadata:
      name: test-kvlp6
      namespace: default
    ```


## Test Results

1. Deploy the nettools application outside the cluster to simulate an external service. nettools will return the requester's source IP address in the HTTP response.

    ```shell
    docker run -d --net=host ghcr.io/spidernet-io/egressgateway-nettools:latest /usr/bin/nettools-server -protocol web -webPort 8080
    ```

2. Verify the effect of egress traffic in the visitor Pod within the cluster. You should observe that when the visitor accesses the external service, nettools returns a source IP matching the EgressPolicy `.status.eip`.
    ```shell
    $ kubectl get pod
    NAME                       READY   STATUS    RESTARTS   AGE
    visitor-6764bb48cc-29vq9   1/1     Running   0          15m

    $ kubectl exec -it visitor-6764bb48cc-29vq9 bash
    $ curl 10.6.1.92:8080
    Remote IP: 172.22.0.110
    ```
