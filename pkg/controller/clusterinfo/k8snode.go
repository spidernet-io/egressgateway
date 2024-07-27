// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package clusterinfo

import (
	v1 "k8s.io/api/core/v1"

	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
)

// GetNodeIPList get node ip list
func getNodeIPList(node *v1.Node) (nodeIPv4, nodeIPv6 []string) {
	if node == nil {
		return
	}
	for _, addresses := range node.Status.Addresses {
		if addresses.Type == v1.NodeInternalIP {
			if isV4, _ := ip.IsIPv4(addresses.Address); isV4 {
				nodeIPv4 = append(nodeIPv4, addresses.Address)
			}
			if isV6, _ := ip.IsIPv6(addresses.Address); isV6 {
				nodeIPv6 = append(nodeIPv6, addresses.Address)
			}
		}
	}
	return
}
