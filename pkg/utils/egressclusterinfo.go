// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	corev1 "k8s.io/api/core/v1"
)

func GetNodeIP(node *corev1.Node) (nodeIPv4, nodeIPv6 string) {
	for _, addresses := range node.Status.Addresses {
		if addresses.Type == corev1.NodeInternalIP {
			if isV4, _ := IsIPv4(addresses.Address); isV4 {
				nodeIPv4 = addresses.Address
			}
			if isV6, _ := IsIPv6(addresses.Address); isV6 {
				nodeIPv6 = addresses.Address
			}
		}
	}
	return
}
