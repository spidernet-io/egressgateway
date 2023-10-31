# 自建集群安装 EgressGateway

## 介绍

本文将演示在一个自建集群上快速安装 EgressGateway。

## 要求

1. 已经具备一个自建好的 Kubernetes 集群，至少有 2 个节点。

2. 集群准备好 helm 工具。

3. 目前 EgressGateway 支持如下 CNI。

    * "Calico"

        如果您的集群使用 [Calico](https://www.tigera.io/project-calico/) CNI，请执行如下命令，
        该命令确保 EgressGateway 的 iptables 规则不会被 Calico 规则覆盖，否则 EgressGateway 将不能工作。
      
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

        > 有关 `spec.chainInsertMode` 的含义可参考 [Calico 文档](https://projectcalico.docs.tigera.io/reference/resources/felixconfig)。

    * "Flannel"

        [Flannel](https://github.com/flannel-io/flannel) CNI 不需要任何配置，您可以跳过此步骤。

    * "Weave"

        [Weave](https://github.com/flannel-io/flannel) CNI 不需要任何配置，您可以跳过此步骤。

    * "Spiderpool"

        如果您的集群使用 [Spiderpool](https://github.com/spidernet-io/spiderpool) 搭配其他CNI，需要进行如下操作。

        将集群外的服务地址添加到 spiderpool.spidercoordinators 的 'default' 对象的 'hijackCIDR' 中，使 Pod 访问这些外部服务时，流量先经过 Pod 所在的主机，从而被 EgressGateway 规则匹配。

        ```shell
        # "1.1.1.1/32", "2.2.2.2/32" 为外部服务地址。对于已经运行的 Pod，需要重启 Pod，这些路由规则才会在 Pod 中生效。
        kubectl patch spidercoordinators default  --type='merge' -p '{"spec": {"hijackCIDR": ["1.1.1.1/32", "2.2.2.2/32"]}}'
        ```

## 安装 EgressGateway

### 添加 EgressGateway 仓库

```shell
helm repo add egressgateway https://spidernet-io.github.io/egressgateway/
helm repo update
```

### 安装 EgressGateway

1. 可使用如下命令快速安装 EgressGateway

    ```shell
    helm install egressgateway egressgateway/egressgateway \
		  -n kube-system \
			--set feature.tunnelIpv4Subnet="192.200.0.1/16" \
			--wait --debug
    ```

   在安装命令中，有如下注意点：

   * 安装命令中，需要提供用于 EgressGateway 隧道节点的 IPv4 和 IPv6 网段，要求该网段和集群内的其他地址不冲突。
   * 可使用选项 `--set feature.tunnelDetectMethod="interface=eth0"` 来定制 EgressGateway 隧道的承载网卡，否则，默认使用默认路由的网卡。
   * 如果希望使用 IPv6 ，可使用选项 `--set feature.enableIPv6=true` 开启，并设置 `feature.tunnelIpv6Subnet`。
   * EgressGateway Controller 支持高可用，可通过 `--set controller.replicas=2` 设置。
   * 开启网关节点上的返回路由规则，可通过设置 `--set feature.enableGatewayReplyRoute=true` 开启，如果要搭配 Spiderpool 支持 underlay CNI，则必须开启该选项。

2. 确认所有的 EgressGateway Pod 运行正常。

    ```shell
    $ kubectl get pod -n kube-system | grep egressgateway
    egressgateway-agent-29lt5                  1/1     Running   0          9h
    egressgateway-agent-94n8k                  1/1     Running   0          9h
    egressgateway-agent-klkhf                  1/1     Running   0          9h
    egressgateway-controller-5754f6658-7pn4z   1/1     Running   0          9h
    ```

3. 任何功能配置，可通过调整 EgressGateway 应用的 Helm Values 来实现。

## 创建 EgressGateway 实例

1. EgressGateway 定义了一组节点作为集群的出口网关，集群内的 egress 流量将会通过这组节点转发而出集群。因此，我们需要预先定义一组 EgressGateway，例子如下：

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

    创建命令中：

    * 如上 YAML 例子中，`spec.ippools.ipv4` 定义了一组 egress 的 出口 IP 地址，需要根据具体环境的实际情况调整，
    * 其中，`spec.ippools.ipv4` 的 CIDR 应该是与网关节点上的出口网卡（一般情况下是默认路由的网卡）的子网相同，否则，极有可能导致 egress 访问不通。
    * 通过 EgressGateway 的 `spec.nodeSelector` 来 select 一组节点作为出口网关，它支持 select 多个节点来实现高可用。

2. 给出口网关节点打上 label，可以给多个 node 打上 label，作为生成环境，建议 2 个节点，作为 POC 环境， 建议 1 个节点即可

    ```shell
    kubectl label node $NodeName egressgateway="true"
    ```

3. 查看状态如下

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

    在如上输出中：

    * `status.nodeList` 字段已经识别到了符合 `spec.nodeSelector` 的节点及该节点对应的 EgressTunnel 对象的状态
    * `spec.ippools.ipv4DefaultEIP` 字段会从 `spec.ippools.ipv4` 中随机选择一个 IP 地址作为该组 EgressGateway 的默认 VIP，
      它的作用是：当为应用创建 EgressPolicy 对象时，如果未指定 VIP 地址，则默认分配使用该默认 VIP

## 创建应用和出口策略

1. 创建一个应用，它将用于测试 POD 访问集群外部用途，并给它打上 label。

    ```shell
    kubectl create deployment visitor --image nginx
    ```

2. 为应用创建 EgressPolicy CR 对象。
   EgressPolicy 实例用于定义哪些 Pod 的出口流量要经过 EgressGateway 节点转发，以及其它的配置细节。
   可创建如下例子，当匹配的 Pod 访问任意集群外部的地址（任意不是 Node IP、CNI Pod CIDR、ClusterIP 的地址）时，
   都会被 EgressGateway Node 转发。注意的是，EgressPolicy 对象是租户级别的，因此，它务必创建在 selected 应用的租户下。

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

    如上创建命令中：

    * `spec.egressGatewayName` 指定了使用哪一组 EgressGateway 的名字。
    * `spec.appliedTo.podSelector` 指定了本策略生效在集群内的哪些 Pod。
    * 集群的 egress 流量的源 IP 地址有两种选择：
        * 可使用网关节点的 IP。它可适用于公有云和传统网络等环境，缺点是，随着网关节点的故障，出口源 IP 可能会发生变化。
          可设置 `spec.egressIP.useNodeIP=true` 来生效。
        * 可使用独立的 VIP，因为 EgressGateway 是基于 ARP 原理生效 VIP，所以它适用于传统网络，而不适用于公有云等环境。
          它的优点是，出口源 IP 永久是固定的。在 EgressPolicy 中不做任何设置，则默认使用 egressGatewayName 的缺省 VIP，
          或者可单独手动指定 `spec.egressIP.ipv4`，其 IP 值务必符合 EgressGateway 中的 IP 池。

3. 查看 EgressPolicy 的状态

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

    如上输出中：

    * `status.eip` 展示了该组应用出集群时使用的出口 IP 地址。
    * `status.node` 展示了哪一个 EgressGateway 的节点在实时的负责出口流量的转发。
      注：EgressGateway 节点支持高可用，当存在多个 EgressGateway 节点时，所有的 EgressPolicy 会均摊到不同的 EgressGateway 节点上实施。

4. 查看 EgressEndpointSlices 的状态

    每个 EgressPolicy 对象，都有一个对应的 EgressEndpointSlices 对象，其中存储了 EgressPolicy 选择的 Pod IP 地址集合。
    当应用无法出口访问时，可排查该对象中的 IP 地址是否正常。

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

## 测试效果

1. 可在集群外部署应用 nettools，用于模拟一个集群外部的服务，nettools 会在 http 回复中返回请求者的源 IP 地址。

    ```shell
    docker run -d --net=host ghcr.io/spidernet-io/egressgateway-nettools:latest /usr/bin/nettools-server -protocol web -webPort 8080
    ```

2. 在集群内部的 visitor Pod 中，验证出口流量的效果，我们可以看到 visitor 访问外部服务，
   nettools 返回的源 IP 符合了 EgressPolicy `.status.eip` 的效果。

    ```shell
    $ kubectl get pod
    NAME                       READY   STATUS    RESTARTS   AGE
    visitor-6764bb48cc-29vq9   1/1     Running   0          15m

    $ kubectl exec -it visitor-6764bb48cc-29vq9 bash
    $ curl 10.6.1.92:8080
    Remote IP: 10.6.1.60
    ```
