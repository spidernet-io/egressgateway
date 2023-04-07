// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressnode_test

import (
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("Egressnode", func() {
	It("get and check egressnodes", Label("N00001"), func() {
		// check egressnode status
		common.CheckEgressNodeStatus(f, nodes)
	})

	Context("edit node parameters about ip or mac address", Serial, Label("N00002"), func() {
		var (
			node                       string
			physicalInterface          string
			testInterface, testMacAddr string
		)

		BeforeEach(func() {
			testInterface = "testNic-" + tools.GetRandomNum(100)
			testMacAddr = tools.GetRandomMac()
			node = nodes[0]
			physicalInterface, err = common.GetKindNodeDefaultInterface(node, time.Second*20)
			Expect(err).NotTo(HaveOccurred(), "failed to GetKindNodeDefaultInterface, details: %v\n", err)

			DeferCleanup(func() {

			})
		})

		PIt("change interface-name of the node, egressnode cr status.physicalInterface should be same with the changed interface-name", func() {
			_, err = common.SetKindNodeInterface(node, physicalInterface, testInterface, time.Second*30)
			Expect(err).NotTo(HaveOccurred())
			Expect(common.WaitEgressNodePhysicalInterfaceUpgraded(f, node, testInterface, time.Minute)).NotTo(HaveOccurred())
		})
		PIt("change egress.vxlan ip of the node, the ip should restore its original value based on the egressnode cr status.tunnelMac after a while", func() {
			_, err = common.SetKindNodeMacAddrByGivenInterface(node, common.EGRESS_VXLAN_INTERFACE_NAME, testMacAddr, time.Second*30)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
