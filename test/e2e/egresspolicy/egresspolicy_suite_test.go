// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egresspolicy_test

import (
	"github.com/spidernet-io/e2eframework/framework"
	egressgatewayv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestEgresspolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Egresspolicy Suite")
}

var (
	frame *framework.Framework
	err   error
	c     client.WithWatch

	testV4, testV6                bool
	controlPlane, worker, worker2 string

	delOpts client.DeleteOption
)

var _ = BeforeSuite(func() {
	GinkgoRecover()

	delOpts = client.GracePeriodSeconds(0)

	frame, err = framework.NewFramework(GinkgoT(), []func(scheme *runtime.Scheme) error{egressgatewayv1.AddToScheme})
	Expect(err).NotTo(HaveOccurred(), "failed to NewFramework, details: %w", err)
	c = frame.KClient

	// get ip version of cluster
	v4Enabled, v6Enabled, err := common.GetIPVersion(frame)
	Expect(err).NotTo(HaveOccurred())
	if v4Enabled {
		testV4 = true
	}
	if v6Enabled && !v4Enabled {
		testV6 = true
	}

	// get all nodes
	nodes := frame.Info.KindNodeList
	GinkgoWriter.Printf("nodes: %v\n", nodes)

	for _, node := range nodes {
		GinkgoWriter.Printf("node: %v\n", node)

		switch {
		case strings.HasSuffix(node, "control-plane"):
			controlPlane = node
		case strings.HasSuffix(node, "worker"):
			worker = node
		case strings.HasSuffix(node, "worker2"):
			worker2 = node
		default:
		}
	}
})
