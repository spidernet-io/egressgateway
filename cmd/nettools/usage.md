# nettools usage

此工具用来验证 Egress IP 是否生效，可以快速测试 TCP、UDP 及 Web HTTP/WebSocket 等多种模式。

## nettools-server 使用

nettools-server 可以部署在集群外部的任意机器上，用做 Egress IP 测试的目标服务器。

```shell
docker run -d --net=host ghcr.io/spidernet-io/egressgateway-nettools:latest /usr/bin/nettools-server 
```

nettools-server 支持更多的可选参数。例如 `-protocol` 协议，可以通过 `-protocol web,tcp,udp` 来启动一种或多种协议的测试服务。`-tcpPort`, `-udpPort`, `-webPort` 支持定义服务的端口，以下为默认端口。

```shell
docker run -d --net=host ghcr.io/spidernet-io/egressgateway-nettools:latest /usr/bin/nettools-server \
  -protocol all \
  -tcpPort=8080 \
  -udpPort=8081 \
  -webPort=8082
```

如果您启动了 web 协议的服务，通过下面命令可以显示您的 Remote IP。 

```shell
curl SERVER_IP:8082
```

您可以阅读下面章节，使用 nettools-client 工具进行测试。

## nettools-client 使用

### 用于 EgressGateway 安装后的验证

当您在集群安装了 EgressGateway 时，会测试运行是否符合预期。可以通过启动一组 Deployment，并为 Deployment 配置相应的 EgressPolicy 规则。

以下为一个可以快速启动的 Deployment。

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: egress-demo
  namespace: default
spec:
  selector:
    matchLabels:
      app: egress-demo
  template:
    metadata:
      labels:
        app: egress-demo
    spec:
      containers:
        - command:
            - sleep
            - infinity
          image: ghcr.io/spidernet-io/egressgateway-nettools:latest
          imagePullPolicy: IfNotPresent
          name: egress-demo
```

然后通过 `kubectl exec` 进入相应的 Pod，对 nettools-server 发起访问即可测试 Egress。

```shell
nettools-client` -protocol=all -addr=[NETTOOLS_SERVER_ADDR>] -webPort=[WEB_PORT] -tcpPort=[TCP_PORT] -udpPort=[UDP_PORT]
```

将上述占位符替换掉就得到下面一个测试测试命令。

```shell
nettools-client -protocol=all -addr=172.18.0.5 -webPort=63382 -tcpPort=63380 -udpPort=63381 -timeout=10
```

您可以参考下面的可选参数进行更多的自定义测试。

```shell

```shell
nettools-client -h
  -addr string
        Server listen addr, default is all local addresses
  -protocol string
        Server listen protocol, available options: tcp,udp,web,all (default "tcp")
  -tcpPort string
        TCP listen port (default "8080")
  -udpPort string
        UDP listen port (default "8081")
  -webPort string
        WEB listen port (default "8082")
  -timeout int
        command execution seconds time (default 10)        
```

### 用于 CI 自动化测试

nettools-client 以下可选参数主要用于 EgressGateway 项目的自动化测试。

```shell
nettools-client -h
  -batch
        batch mode for CI (default false)
  -contain
        contain egressIP (default true)
  -eip string
        egress IP
```

使用 `-batch` 参数可以启用批处理模式，默认为 false 。使用 `-contain` 参数可以控制是否包含出口 IP，默认为真 `true`。使用 `-eip` 参数可以指定 Egress IP。如果命令行工具的执行结果不符合预期，退出码为 1，否则为 0。
