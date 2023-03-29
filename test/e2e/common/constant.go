// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"log"
	"os"
)

// egressgateway
const (
	EGRESSGATEWAY_CHAIN               = "EGRESSGATEWAY-MARK-REQUEST"
	EGRESSGATEWAY_CONFIGMAP_NAMESPACE = "kube-system"
	EGRESSGATEWAY_CONFIGMAP_NAME      = "egressgateway"
	EGRESSGATEWAY_CONFIGMAP_KEY       = "conf.yml"
	EGRESS_VXLAN_INTERFACE_NAME       = "egress.vxlan"
	EGRESSAGEWAY_NAME                 = "default"
)

// test
const (
	POD_NAMESPACE = "default"

	WEBSOCKET = " WebSocket Server Say hello!"
	UDP       = "UDP Server Say hello!"
	TCP       = "TCP Server Say hello!"

	UDP_CONNECTED = "UDP Client connected!"
	TCP_CONNECTED = "TCP Client connected!"
	WEB_CONNECTED = "WEB Client connected!"
)

// nettools sever env info key
const (
	IMAGE           = "IMAGE"
	NETTOOLS_SERVER = "NETTOOLS_SERVER"
	MOD             = "MOD"
	TCP_PORT        = "TCP_PORT"
	UDP_PORT        = "UDP_PORT"
	WEB_PORT        = "WEB_PORT"
)

var NettoolsServer = map[string]string{
	IMAGE:           "",
	NETTOOLS_SERVER: "",
	MOD:             "",
	TCP_PORT:        "",
	UDP_PORT:        "",
	WEB_PORT:        "",
}

func init() {
	for k := range NettoolsServer {
		if env := os.Getenv(k); len(env) != 0 {
			NettoolsServer[k] = env
		} else {
			log.Fatalf("can not found netTools server env: %s\n", k)
		}
	}
}
