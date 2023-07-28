// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/e2eframework/framework"
	egressgatewayv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

func TestEgressgateway(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Egressgateway Suite")
}

const gateway = "gateway"

var (
	f                  *framework.Framework
	err                error
	c                  client.WithWatch
	allNodes           []string
	nodeObjs           []*v1.Node
	enableV4, enableV6 bool
)

var _ = BeforeSuite(func() {
	GinkgoRecover()

	f, err = framework.NewFramework(GinkgoT(), []func(scheme *runtime.Scheme) error{egressgatewayv1beta1.AddToScheme})
	Expect(err).NotTo(HaveOccurred(), "failed to NewFramework, details: %w", err)
	c = f.KClient

	// all nodes
	allNodes = f.Info.KindNodeList
	Expect(allNodes).NotTo(BeEmpty())

	for i, node := range allNodes {
		GinkgoWriter.Printf("%dTh node: %s\n", i, node)
		getNode, err := f.GetNode(node)
		Expect(err).NotTo(HaveOccurred())
		nodeObjs = append(nodeObjs, getNode)
	}
	Expect(len(nodeObjs) > 2).To(BeTrue(), "test case needs at lest 3 nodes")

	enableV4, enableV6, err = common.GetIPVersion(f)
	Expect(err).NotTo(HaveOccurred())
	GinkgoWriter.Printf("enableV4: %v, enableV6: %v\n", enableV4, enableV6)
})
