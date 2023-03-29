// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressnode_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("Egressnode", func() {
	PIt("check egressnodes", Label("N00001"), func() {
		nodes, err := common.GetAllNodes(f)
		Expect(err).NotTo(HaveOccurred())
		egressNode := new(egressv1.EgressNode)
		Expect(common.GetEgressNode(f, nodes[0], egressNode)).NotTo(HaveOccurred())
		GinkgoWriter.Printf("egressNode: %v\n", egressNode)
	})
})
