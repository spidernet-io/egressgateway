// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	v1 "k8s.io/api/core/v1"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
)

func IsNodeReady(node *v1.Node) bool {
	for i := range node.Status.Conditions {
		cond := &node.Status.Conditions[i]
		// - NodeReady = ConditionTrue
		// - NodeOutOfDisk = ConditionFalse
		// - NodeNetworkUnavailable = ConditionFalse
		if cond.Type == v1.NodeReady && cond.Status != v1.ConditionTrue {
			return false
		}
	}
	// nodes that are marked unscheduled
	return !(node.Spec.Unschedulable)
}

func IsNodeVxlanReady(node *egressv1.EgressNode, enableIPv4, enableIPv6 bool) bool {
	if enableIPv4 {
		if node.Status.VxlanIPv4IP == "" {
			return false
		}
		if node.Status.TunnelMac == "" {
			return false
		}
		if node.Status.PhysicalInterfaceIPv4 == "" {
			return false
		}
	}
	if enableIPv6 {
		if node.Status.VxlanIPv6IP == "" {
			return false
		}
		if node.Status.PhysicalInterfaceIPv6 == "" {
			return false
		}
	}
	if node.Status.PhysicalInterface == "" {
		return false
	}
	return true
}
