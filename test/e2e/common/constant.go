// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

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
	POD_IMAGE     = "ghcr.io/spidernet-io/egressgateway-nettools:v1"
	CLIENT_NODE   = "egressgateway-control-plane"

	SERVER_IP_NAME = "SERVER_IP"
	MOD_NAME       = "MOD"
	TCP_PORT_NAME  = "TCP_PORT"
	UDP_PORT_NAME  = "UDP_PORT"
	WEB_PORT_NAME  = "WEB_PORT"
	MOD_VALUE      = "all"
	TCP_PORT_VALUE = "63380"
	UDP_PORT_VALUE = "63381"
	WEB_PORT_VALUE = "63382"

	WEBSOCKET = " WebSocket Server Say hello!"
	UDP       = "UDP Server Say hello!"
	TCP       = "TCP Server Say hello!"

	UDP_CONNECTED = "UDP Client connected!"
	TCP_CONNECTED = "TCP Client connected!"
	WEB_CONNECTED = "WEB Client connected!"
)
