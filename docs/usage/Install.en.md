# Installing EgressGateway on a Self-Managed Cluster

## Introduction

This guide will demonstrate the quick installation of EgressGateway on a self-managed cluster.

## Requirements

1. You should already have a self-managed Kubernetes cluster with at least 2 nodes.

2. The cluster should have helm tool installed and ready to use.

3. Currently, EgressGateway supports the following CNI (Container Network Interface):

    * "Calico"

        If your cluster is using [Calico](https://www.tigera.io/project-calico/) CNI, please execute the following command.
        This command ensures that the iptables rules of EgressGateway are not overridden by Calico rules; otherwise,
        EgressGateway will not function properly.

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

        > For details about `spec.chainInsertMode`, see [Calico docs](https://projectcalico.docs.tigera.io/reference/resources/felixconfig).

    * "Flannel"

        [Flannel](https://github.com/flannel-io/flannel) CNI does not require any configuration. You can skip this step.

    * "Weave"

        [Weave](https://github.com/flannel-io/flannel) CNI does not require any configuration. You can skip this step.

    * "Spiderpool"

        If your cluster is using [Spiderpool](https://github.com/spidernet-io/spiderpool) with another CNI, follow these steps:

        Add the addresses of external services outside the cluster to the 'hijackCIDR' field in the 'default' object of
        spiderpool.spidercoordinators. This ensures that when Pods access these external services, the traffic goes through
        the host where the Pod is located and matches the EgressGateway rules.

        ```shell
        # "1.1.1.1/32", "2.2.2.2/32" are the addresses of external services. For already running Pods, you need to restart them for these routing rules to take effect within the Pods.
        kubectl patch spidercoordinators default --type='merge' -p '{"spec": {"hijackCIDR": ["1.1.1.1/32", "2.2.2.2/32"]}}'
        ```

## Install EgressGateway

### Add EgressGateway Repository

```shell
helm repo add egressgateway https://spidernet-io.github.io/egressgateway/
helm repo update
```

### Install EgressGateway

1. You can use the following command to quickly install EgressGateway:

    ```shell
    helm install egressgateway egressgateway/egressgateway \
        -n kube-system \
        --set feature.tunnelIpv4Subnet="192.200.0.1/16" \
        --wait --debug
    ```

    In the installation command, please note the following:

    * In the command, you need to provide an IPv4 and IPv6 subnet for the EgressGateway tunnel nodes.
      Make sure this subnet does not conflict with other addresses in the cluster.
    * You can customize the network interface used by the EgressGateway tunnel by using the option
      `--set feature.tunnelDetectMethod="interface=eth0"`. Otherwise, the default route interface is used.
    * If you want to enable IPv6, use the option `--set feature.enableIPv6=true` and set `feature.tunnelIpv6Subnet`.
    * EgressGateway Controller supports high availability. You can set `--set controller.replicas=2` to have two replicas.
    * To enable the return routing rules on the gateway nodes, use `--set feature.enableGatewayReplyRoute=true`.
      This option must be enabled if you want to use Spiderpool with underlay CNI.

2. Confirm that all EgressGateway Pods are running properly.

    ```shell
    $ kubectl get pod -n kube-system | grep egressgateway
    egressgateway-agent-29lt5                  1/1     Running   0          9h
    egressgateway-agent-94n8k                  1/1     Running   0          9h
    egressgateway-agent-klkhf                  1/1     Running   0          9h
    egressgateway-controller-5754f6658-7pn4z   1/1     Running   0          9h
    ```

3. Any feature configurations can be achieved by adjusting the Helm Values of the EgressGateway application.

## Creat an EgressGateway Instance

1. EgressGateway defines a set of nodes as an exit gateway for the cluster. The egress traffic from within the cluster
   will be forwarded through this set of nodes. Therefore, we need to define a set of EgressGateway instances in advance.
   Here is an example:

    ```shell
    cat <<EOF | kubectl apply -f -
    apiVersion: egressgateway.spidernet.io/v1beta1
    kind: EgressGateway
    metadata:
      name: default
    spec:
      ippools:
        ipv4:
        - "10.6.1.60-10.6.1.66"
      nodeSelector:
        selector:
          matchLabels:
            egressgateway: "true"
    EOF
    ```

    In the creation command:

    * In the YAML example above, `spec.ippools.ipv4` defines a set of exit IP addresses for egress traffic.
      You need to adjust it according to the specific environment.
    * The CIDR of `spec.ippools.ipv4` should be the same as the subnet of the egress interface on the gateway nodes
      (usually the default route interface). Otherwise, it may result in inaccessible egress traffic.
    * Use `spec.nodeSelector` of EgressGateway to select a set of nodes as the exit gateways.
      It supports selecting multiple nodes for high availability.

2. Label the exit gateway nodes. You can label multiple nodes. For production environments,
   it is recommended to use 2 nodes. For POC environments, 1 node is sufficient.

    ```shell
    kubectl label node $NodeName egressgateway="true"
    ```

3. Check the status as follows:

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

    In the above output:

    * The `status.nodeList` field has identified the nodes that match `spec.nodeSelector` and shows
      the status of the corresponding EgressTunnel objects.
    * The `spec.ippools.ipv4DefaultEIP` field randomly selects an IP address from `spec.ippools.ipv4` as the default VIP
      for this group of EgressGateways. This default VIP is used when creating EgressPolicy objects for applications.
      If no VIP address is specified, the default VIP will be assigned.

## Creat Applications and Egress Policies

1. Create an application that will be used to test accessing external resources from within a Pod, and label it.

    ```shell
    kubectl create deployment visitor --image nginx
    ```

2. Create an EgressPolicy CR object for the application. An EgressPolicy instance is used to define which Pods'
   egress traffic needs to be forwarded through the EgressGateway nodes, along with other configuration details.
   You can create an example like the following.
   (Note: The EgressPolicy object is tenant-level, so it must be created under the selected application's namespace.)

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

    In the above creation command:

    * `spec.egressGatewayName` specifies which group of EgressGateways to use.
    * `spec.appliedTo.podSelector` specifies which Pods this policy will apply to within the cluster.
    * There are two options for the source IP address of egress traffic in the cluster:
        * You can use the IP address of the gateway nodes. This option is suitable for public cloud and
          traditional network environments. The drawback is that the outgoing source IP may change if
          the gateway nodes fail. Set `spec.egressIP.useNodeIP=true` to enable this option.
        * You can use a dedicated VIP. Since EgressGateway works based on ARP, it is suitable for traditional network
          environments but not for public cloud environments. The advantage is that the outgoing source IP remains permanent
          and fixed. If no setting is specified in the EgressPolicy, it will default to using the default VIP of the
          egressGatewayName. Alternatively, you can manually specify `spec.egressIP.ipv4`, but its IP value must comply
          with the IP pool defined in EgressGateway.

3. Check the status of the EgressPolicy:

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

    In the above output:

    * `status.eip` shows the outbound IP address used by the group of applications when exiting the cluster.
    * `status.node` shows which EgressGateway node is currently responsible for forwarding the egress traffic in real-time.
      Note: EgressGateway nodes support high availability. When multiple EgressGateway nodes exist, all EgressPolicies
      are evenly distributed among different EgressGateway nodes.

4. Check the status of EgressEndpointSlices.

    Each EgressPolicy object has a corresponding EgressEndpointSlices object, which stores the collection of Pod IP addresses
    selected by the EgressPolicy. If an application cannot access external resources, you can check if the IP addresses
    in this object are functioning properly.

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

## Test

1. Deploy the nettools application outside the cluster to simulate an external service.
   The nettools application will return the requester's source IP address in the HTTP response.

    ```shell
    docker run -d --net=host ghcr.io/spidernet-io/egressgateway-nettools:latest /usr/bin/nettools-server -protocol web -webPort 8080
    ```

2. Validate the effect of egress traffic from within the visitor Pod in the cluster. We can see that
   when the visitor accesses the external service, the source IP returned by nettools matches the effect
   of the EgressPolicy's `.status.eip`.

    ```shell
    $ kubectl get pod
    NAME                       READY   STATUS    RESTARTS   AGE
    visitor-6764bb48cc-29vq9   1/1     Running   0          15m

    $ kubectl exec -it visitor-6764bb48cc-29vq9 bash
    $ curl 10.6.1.92:8080
    Remote IP: 10.6.1.60
    ```
