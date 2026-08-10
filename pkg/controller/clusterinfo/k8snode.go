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

func getNodePodCIDRList(node *v1.Node) (podIPv4, podIPv6 []string) {
	if node == nil {
		return
	}

	podCIDRs := node.Spec.PodCIDRs
	if len(podCIDRs) == 0 && node.Spec.PodCIDR != "" {
		podCIDRs = []string{node.Spec.PodCIDR}
	}

	for _, cidr := range podCIDRs {
		if isV4, err := ip.IsIPv4Cidr(cidr); err == nil && isV4 {
			podIPv4 = append(podIPv4, cidr)
			continue
		}
		if isV6, err := ip.IsIPv6Cidr(cidr); err == nil && isV6 {
			podIPv6 = append(podIPv6, cidr)
		}
	}

	return
}
