// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressnode_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("Egressnode", func() {
	It("get and check egressnodes", Label("N00001"), func() {
		// check egressnode status
		common.CheckEgressNodeStatus(f, nodes)
	})
})
