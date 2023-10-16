// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package flag

import "flag"

const (
	ProtocolTcp = "tcp"
	ProtocolUdp = "udp"
	ProtocolAll = "all"
	ProtocolWeb = "web"
)

type Config struct {
	Addr, Proto, TcpPort, UdpPort, WebPort *string
	Timeout                                *int
	EgressIP                               *string
	Contain                                *bool
	Batch                                  *bool
}

func ParseClientFlag() Config {
	config := Config{
		Addr:     flag.String("addr", "", "server listen ip addr, default is all local addresses"),
		Proto:    flag.String("protocol", "tcp", "server listen protocol, available options: tcp,udp,web(websocket),all"),
		TcpPort:  flag.String("tcpPort", "8080", "tcp listen port"),
		UdpPort:  flag.String("udpPort", "8081", "udp listen port"),
		WebPort:  flag.String("webPort", "8082", "webSocket listen port"),
		Timeout:  flag.Int("timeout", 10, "command execution seconds time"),
		EgressIP: flag.String("eip", "", "egress IP"),
		Contain:  flag.Bool("contain", true, "contain egressIP"),
		Batch:    flag.Bool("batch", false, "batch mode"),
	}

	flag.Parse()

	return config
}

type ServerConfig struct {
	Addr, Proto, TcpPort, UdpPort, WebPort *string
}

func ParseServerFlag() ServerConfig {
	config := ServerConfig{
		Addr:    flag.String("addr", "", "server listen ip addr, default is all local addresses"),
		Proto:   flag.String("protocol", "tcp", "server listen protocol, available options: tcp,udp,web(websocket),all"),
		TcpPort: flag.String("tcpPort", "8080", "tcp listen port"),
		UdpPort: flag.String("udpPort", "8081", "udp listen port"),
		WebPort: flag.String("webPort", "8082", "webSocket listen port"),
	}

	flag.Parse()

	return config
}
