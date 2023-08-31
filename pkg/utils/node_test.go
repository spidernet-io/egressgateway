// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/spidernet-io/egressgateway/pkg/utils"
)

func TestGetNodeIP(t *testing.T) {
	node := new(v1.Node)
	address := make([]v1.NodeAddress, 0, 2)
	address = append(
		address,
		v1.NodeAddress{
			Type:    v1.NodeInternalIP,
			Address: "127.0.0.1",
		},
		v1.NodeAddress{
			Type:    v1.NodeInternalIP,
			Address: "::1",
		},
	)
	node.Status.Addresses = address
	utils.GetNodeIP(node)
}
