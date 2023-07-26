// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package reliability_test

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

func TestReliability(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reliability Suite")
}

var (
	f   *framework.Framework
	err error
	c   client.WithWatch

	v4Enabled, v6Enabled    bool
	nodes, masters, workers []string
	nodeObjs                []*v1.Node

	serverIPv4, serverIPv6 string
	dst                    []string

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

	masters, err = common.GetNodesByMatchLabels(f, common.ControlPlaneLabel)
	Expect(err).NotTo(HaveOccurred())
	Expect(masters).NotTo(BeEmpty())
	GinkgoWriter.Printf("masters: %v\n", masters)

	workers, err = common.GetUnmatchedNodes(f, masters)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(workers) > 0).To(BeTrue(), "worker nodes number is less then 2")
	GinkgoWriter.Printf("workers: %v\n", workers)

	// net-tool server
	dst = make([]string, 0)
	if v4Enabled {
		serverIpv4b, err := tools.GetContainerIPV4(common.Env[common.NETTOOLS_SERVER_A], time.Second*10)
		Expect(err).NotTo(HaveOccurred())
		serverIPv4 = string(serverIpv4b)
		GinkgoWriter.Printf("serverIPv4: %v\n", serverIPv4)
		Expect(serverIPv4).NotTo(BeEmpty())

		dst = append(dst, serverIPv4+"/8")
		GinkgoWriter.Printf("dst: %v\n", dst)
	}

	if v6Enabled {
		serverIpv6b, err := tools.GetContainerIPV6(common.Env[common.NETTOOLS_SERVER_A], time.Second*10)
		Expect(err).NotTo(HaveOccurred())
		serverIPv6 = string(serverIpv6b)
		Expect(serverIPv6).NotTo(BeEmpty())

		dst = append(dst, serverIPv6+"/64")
		GinkgoWriter.Printf("dst: %v\n", dst)

		serverIPv6 = "[" + serverIPv6 + "]"
		GinkgoWriter.Printf("serverIPv6: %v\n", serverIPv6)
	}
})
