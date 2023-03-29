// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0
package nettools

import (
	"flag"
)

const (
	PROTOCOL_TCP = "tcp"
	PROTOCOL_UDP = "udp"
	PROTOCOL_ALL = "all"
	PROTOCOL_WEB = "web"
)

type Config struct {
	Addr, Proto, TcpPort, UdpPort, WebPort *string
}

func ParseFlag() Config {
	config := Config{
		Addr:    flag.String("addr", "", "server listen ip addr, default is all local addresses"),
		Proto:   flag.String("protocol", "tcp", "server listen protocol, available options: tcp,udp,web(websocket),all"),
		TcpPort: flag.String("tcpPort", "8080", "tcp listen port"),
		UdpPort: flag.String("udpPort", "8081", "udp listen port"),
		WebPort: flag.String("webPort", "8082", "webSocket listen port"),
	}

	flag.Parse()
	flag.Usage()

	return config
}
