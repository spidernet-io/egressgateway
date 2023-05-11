// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEgressgateway(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Egressgateway Suite")
}

//
//var (
//	f        *framework.Framework
//	err      error
//	c        client.WithWatch
//	allNodes []string
//)
//
//var _ = BeforeSuite(func() {
//	GinkgoRecover()
//
//	f, err = framework.NewFramework(GinkgoT(), []func(scheme *runtime.Scheme) error{egressgatewayv1.AddToScheme})
//	Expect(err).NotTo(HaveOccurred(), "failed to NewFramework, details: %w", err)
//	c = f.KClient
//	allNodes = f.Info.KindNodeList
//	Expect(allNodes).NotTo(BeEmpty())
//	for _, node := range allNodes {
//		getNode, err := f.GetNode(node)
//		Expect(err).NotTo(HaveOccurred())
//		GinkgoWriter.Printf("node: %v, nodeLabel: %v\n", getNode, getNode.Labels)
//	}
//})
