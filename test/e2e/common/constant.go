// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"log"
	"os"
)

// egressgateway
const (
	EGRESSGATEWAY_CHAIN         = "EGRESSGATEWAY-MARK-REQUEST"
	EGRESS_VXLAN_INTERFACE_NAME = "egress.vxlan"

	RANDOM            = "Random"
	AVERAGE_SELECTION = "AverageSelection"
)

// egressgateway configmap
const (
	EGRESSGATEWAY_CONFIGMAP_NAME = "egressgateway"
	EGRESSGATEWAY_CONFIGMAP_KEY  = "conf.yml"
	CALICO                       = "calico"
)

// test
const (
	WEBSOCKET = " WebSocket Server Say hello!"
	UDP       = "UDP Server Say hello!"
	TCP       = "TCP Server Say hello!"

	resetByPeer = "connection reset by peer"

	UDP_CONNECTED = "UDP Client connected!"
	TCP_CONNECTED = "TCP Client connected!"
	WEB_CONNECTED = "WEB Client connected!"
)

// env info key
const (
	IMAGE             = "IMAGE"
	NETTOOLS_SERVER_A = "NETTOOLS_SERVER_A"
	NETTOOLS_SERVER_B = "NETTOOLS_SERVER_B"
	MOD               = "MOD"
	TCP_PORT          = "TCP_PORT"
	UDP_PORT          = "UDP_PORT"
	WEB_PORT          = "WEB_PORT"
	EGRESS_NAMESPACE  = "EGRESS_NAMESPACE"
)

// kubeadm-config
const (
	kubeadmConfig        = "kubeadm-config"
	clusterConfiguration = "ClusterConfiguration"
	serviceSubnet        = "serviceSubnet"
)

// namespace
const (
	kubeSystem = "kube-system"
	NSDefault  = "default"
)

var Env = map[string]string{
	IMAGE:             "",
	NETTOOLS_SERVER_A: "",
	NETTOOLS_SERVER_B: "",
	MOD:               "",
	TCP_PORT:          "",
	UDP_PORT:          "",
	WEB_PORT:          "",
	EGRESS_NAMESPACE:  "",
}

func init() {
	for k := range Env {
		if env := os.Getenv(k); len(env) != 0 {
			Env[k] = env
		} else {
			log.Fatalf("can not found netTools server env: %s\n", k)
		}
	}
}
