// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egresspolicy_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/e2eframework/framework"
	egressgatewayv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

func TestEgresspolicy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Egresspolicy Suite")
}

var (
	f   *framework.Framework
	err error
	c   client.WithWatch

	v4Enabled, v6Enabled bool
	nodes                []string
	nodeObjs             []*v1.Node

	serverIPv4A, serverIPv6A string
	serverIPv4B, serverIPv6B string
	dst                      []string

	delOpts client.DeleteOption
)

var _ = BeforeSuite(func() {
	GinkgoRecover()

	delOpts = client.GracePeriodSeconds(0)

	f, err = framework.NewFramework(GinkgoT(), []func(scheme *runtime.Scheme) error{egressgatewayv1beta1.AddToScheme})
	Expect(err).NotTo(HaveOccurred(), "failed to NewFramework, details: %w", err)
	c = f.KClient

	// IP version of cluster
	v4Enabled, v6Enabled, err = common.GetIPVersion(f)
	Expect(err).NotTo(HaveOccurred())
	GinkgoWriter.Printf("v4Enabled: %v, v6Enabled: %v\n", v4Enabled, v6Enabled)

	// all nodes
	nodes = f.Info.KindNodeList
	GinkgoWriter.Printf("nodes: %v\n", nodes)

	for i, node := range nodes {
		GinkgoWriter.Printf("%dTh node: %s\n", i, node)
		getNode, err := f.GetNode(node)
		Expect(err).NotTo(HaveOccurred())
		nodeObjs = append(nodeObjs, getNode)
	}
	Expect(len(nodeObjs) > 2).To(BeTrue(), "test case needs at lest 3 nodes")

	// net-tool server
	dst = make([]string, 0)
	if v4Enabled {
		serverIpv4a, err := tools.GetContainerIPV4(common.Env[common.NETTOOLS_SERVER_A], time.Second*10)
		Expect(err).NotTo(HaveOccurred())
		serverIPv4A = string(serverIpv4a)
		GinkgoWriter.Printf("serverIPv4a: %v\n", serverIPv4A)
		Expect(serverIPv4A).NotTo(BeEmpty())

		dst = append(dst, serverIPv4A+"/8")
		GinkgoWriter.Printf("dst: %v\n", dst)

		serverIpv4b, err := tools.GetContainerIPV4(common.Env[common.NETTOOLS_SERVER_B], time.Second*10)
		Expect(err).NotTo(HaveOccurred())
		serverIPv4B = string(serverIpv4b)
		GinkgoWriter.Printf("serverIPv4b: %v\n", serverIPv4B)
		Expect(serverIPv4B).NotTo(BeEmpty())
	}

	if v6Enabled {
		serverIpv6a, err := tools.GetContainerIPV6(common.Env[common.NETTOOLS_SERVER_A], time.Second*10)
		Expect(err).NotTo(HaveOccurred())
		serverIPv6A = string(serverIpv6a)
		Expect(serverIPv6A).NotTo(BeEmpty())

		serverIpv6b, err := tools.GetContainerIPV6(common.Env[common.NETTOOLS_SERVER_B], time.Second*10)
		Expect(err).NotTo(HaveOccurred())
		serverIPv6B = string(serverIpv6b)
		Expect(serverIPv6B).NotTo(BeEmpty())

		dst = append(dst, serverIPv6A+"/64")
		GinkgoWriter.Printf("dst: %v\n", dst)

		serverIPv6A = "[" + serverIPv6A + "]"
		GinkgoWriter.Printf("serverIPv6a: %v\n", serverIPv6A)

		serverIPv6B = "[" + serverIPv6B + "]"
		GinkgoWriter.Printf("serverIPv6b: %v\n", serverIPv6B)
	}
})
