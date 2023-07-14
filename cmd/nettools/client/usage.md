# nettools usage

## 说明
此工具是在 egressGateway e2e 中，用来测试 `Eip` 是否生效的工具。
此工具可以测试3种连接模式：tcp、udp 及 websocket

## 使用

### 帮助
```shell
nettools-client -h

Usage of nettools-client:
  -addr string
        server listen ip addr, default is all local addresses
  -protocol string
        server listen protocol, available options: tcp,udp,web(websocket),all (default "tcp")
  -tcpPort string
        tcp listen port (default "8080")
  -timeout int
        command execution seconds time (default 10)
  -udpPort string
        udp listen port (default "8081")
  -webPort string
        webSocket listen port (default "8082")

```

### 测试某一种连接 例如 websocket （其他连接模式类似）
使用 `nettools-client` -protocol=web -addr=<访问的IP地址> -webPort=<访问的端口> -timeout=<命令执行秒数>
```shell
nettools-client -protocol=web -addr=[fc00:f853:ccd:e793::5] -webPort=63382 -timeout=6
```
输出内容如下：
```shell
Usage of nettools-client:
  -addr string
        server listen ip addr, default is all local addresses
  -protocol string
        server listen protocol, available options: tcp,udp,web(websocket),all (default "tcp")
  -tcpPort string
        tcp listen port (default "8080")
  -timeout int
        command execution seconds time (default 10)
  -udpPort string
        udp listen port (default "8081")
  -webPort string
        webSocket listen port (default "8082")
2023/07/18 10:03:46 trying to connect websocket:  ws://[fc00:f853:ccd:e793::5]:63382/
[fd40::5676:c03:52b7:d55c:cf80]:54280 : WEB Client connected!
2023-07-18 10:03:46.068212658 +0000 UTC m=+716.616780517 clientIP=[fc00:f853:ccd:e793:a::6]:54280 WebSocket Server Say hello!

2023-07-18 10:03:48.070632278 +0000 UTC m=+718.619200144 clientIP=[fc00:f853:ccd:e793:a::6]:54280 WebSocket Server Say hello!

2023-07-18 10:03:50.07135121 +0000 UTC m=+720.619919080 clientIP=[fc00:f853:ccd:e793:a::6]:54280 WebSocket Server Say hello!

2023-07-18 10:03:52.072931034 +0000 UTC m=+722.621498928 clientIP=[fc00:f853:ccd:e793:a::6]:54280 WebSocket Server Say hello!
```

### 测试所有连接模式 
使用 `nettools-client` -protocol=all -addr=<访问的IP地址> -webPort=<访问的 websocket 端口> -tcpPort=<访问的 tcp 端口> -udpPort=<访问的 udp 端口> -timeout=<命令执行秒数>
```shell
nettools-client -protocol=all -addr=172.18.0.5 -webPort=63382 -tcpPort=63380 -udpPort=63381 -timeout=10
```
输出内容如下：
```shell
Usage of nettools-client:
  -addr string
        server listen ip addr, default is all local addresses
  -protocol string
        server listen protocol, available options: tcp,udp,web(websocket),all (default "tcp")
  -tcpPort string
        tcp listen port (default "8080")
  -timeout int
        command execution seconds time (default 10)
  -udpPort string
        udp listen port (default "8081")
  -webPort string
        webSocket listen port (default "8082")
2023/07/18 10:05:37 trying to connect websocket:  ws://172.18.0.5:63382/
2023/07/18 10:05:37 trying to connect udpServer:  172.18.0.5:63381
172.40.14.193:48964 : UDP Client connected!
2023/07/18 10:05:37 trying to connect tcpServer:  172.18.0.5:63380
172.40.14.193:53626 : TCP Client connected!
2023-07-18 10:05:37.698029937 +0000 UTC m=+828.246597810 clientIP=172.18.1.5:48964 UDP Server Say hello!

172.40.14.193:45580 : WEB Client connected!
2023-07-18 10:05:37.699658473 +0000 UTC m=+828.248226326 clientIP=172.18.1.5:45580 WebSocket Server Say hello!

2023-07-18 10:05:39.700306168 +0000 UTC m=+830.248874035 clientIP=172.18.1.5:53626 TCP Server Say hello!

2023-07-18 10:05:39.703592023 +0000 UTC m=+830.252159889 clientIP=172.18.1.5:45580 WebSocket Server Say hello!

2023-07-18 10:05:39.703820604 +0000 UTC m=+830.252388458 clientIP=172.18.1.5:48964 UDP Server Say hello!

2023-07-18 10:05:41.704771349 +0000 UTC m=+832.253339222 clientIP=172.18.1.5:48964 UDP Server Say hello!

2023-07-18 10:05:41.704848423 +0000 UTC m=+832.253416304 clientIP=172.18.1.5:45580 WebSocket Server Say hello!

2023-07-18 10:05:43.703968463 +0000 UTC m=+834.252536315 clientIP=172.18.1.5:53626 TCP Server Say hello!

2023-07-18 10:05:43.705657959 +0000 UTC m=+834.254225845 clientIP=172.18.1.5:45580 WebSocket Server Say hello!

2023-07-18 10:05:43.705928269 +0000 UTC m=+834.254496196 clientIP=172.18.1.5:48964 UDP Server Say hello!
```





