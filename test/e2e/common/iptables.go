// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"fmt"
	"time"

	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

// CheckEgressGatewayChain check if chain EGRESSGATEWAY-MARK-REQUEST created
func CheckEgressGatewayChain(nodesName []string, duration time.Duration) bool {
	command := fmt.Sprintf("iptables -L %s -t mangle", EGRESSGATEWAY_CHAIN)
	for _, nodeName := range nodesName {
		if _, err := tools.ExecInKindNode(nodeName, command, duration); err != nil {
			return false
		}
	}
	return true
}
