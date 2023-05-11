// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressnode_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEgressnode(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Egressnode Suite")
}

//
//var (
//	f     *framework.Framework
//	err   error
//	nodes []string
//)
//
//var _ = BeforeSuite(func() {
//	GinkgoRecover()
//
//	f, err = framework.NewFramework(GinkgoT(), []func(scheme *runtime.Scheme) error{egressgatewayv1.AddToScheme})
//	Expect(err).NotTo(HaveOccurred(), "failed to NewFramework, details: %w", err)
//	nodes = f.Info.KindNodeList
//	Expect(nodes).NotTo(BeEmpty())
//})
