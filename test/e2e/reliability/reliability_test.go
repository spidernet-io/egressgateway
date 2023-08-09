// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package reliability_test

import (
	"context"

	"github.com/go-faker/faker/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("Reliability", func() {
	Context("Test reliability", Label("Reliability"), func() {
		var (
			ctx       context.Context
			egw       *egressv1.EgressGateway
			daemonSet *appsv1.DaemonSet
			policy    *egressv1.EgressPolicy
		)

		BeforeEach(func() {
			var err error

			egNodes := nodeNameList[:2]
			labels := map[string]string{"eg-reliability": "true"}
			selector := egressv1.NodeSelector{Selector: &v1.LabelSelector{MatchLabels: labels}}

			err = common.LabelNodes(ctx, cli, egNodes, labels)
			Expect(err).NotTo(HaveOccurred())

			pool, err := common.GenIPPools(ctx, cli, egressConfig, 3, 2)
			Expect(err).NotTo(HaveOccurred())

			egw, err = common.CreateGatewayNew(ctx, cli, "egw-"+faker.Word(), pool, selector)
			Expect(err).NotTo(HaveOccurred())

			// check default eip
			v4DefaultEip, v6DefaultEip, err := common.GetGatewayDefaultIP(ctx, cli, egw, egressConfig)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("v4DefaultEip: %s, v6DefaultEip: %s\n", v4DefaultEip, v6DefaultEip)

			// daemonSet
			daemonSet, err = common.CreateDaemonSet(ctx, cli, "ds-reliability-"+faker.Word(), config.Image)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Create DaemonSet: %s\n", daemonSet.Name)

			// policy
			policy, err = common.CreateEgressPolicyNew(ctx, cli, egressConfig, egw.Name, daemonSet.Labels)
			Expect(err).NotTo(HaveOccurred())

			// check eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, daemonSet,
				policy.Status.Eip.Ipv4, policy.Status.Eip.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// restore the cluster to avoid affecting the execution of other use cases
			GinkgoWriter.Println("PowerOnNodesUntilClusterReady")
			// Expect(common.PowerOnNodesUntilClusterReady(f, nodes, time.Second*30)).NotTo(HaveOccurred())

			err := common.DeleteObj(ctx, cli, daemonSet)
			Expect(err).NotTo(HaveOccurred())

			err = common.DeleteObj(ctx, cli, policy)
			Expect(err).NotTo(HaveOccurred())

			err = common.DeleteObj(ctx, cli, egw)
			Expect(err).NotTo(HaveOccurred())

			// un label nodes
			//GinkgoWriter.Println("UnLabelNodes")
			//Expect(common.UnLabelNodes(f, egNodes, nodeLabel)).NotTo(HaveOccurred())
		})

		// TODO @bzsuni
		PIt("Test EIP drift after then eip-node shut down", Serial, Label("R00005"), func() {
			//// shut down the eip node
			//GinkgoWriter.Printf("Shut down node: %s\n", nodeNameA)
			//Expect(common.PowerOffNodeUntilNotReady(f, nodeNameA, time.Minute)).NotTo(HaveOccurred())
			//
			//// check if eip drift after node shut down
			//GinkgoWriter.Println("Check if eip drift after node shut down")
			//bs := tools.SubtractionSlice(workers, []string{nodeNameA})
			//Expect(bs).NotTo(BeEmpty())
			//nodeNameB = bs[0]
			//Expect(nodeNameB).NotTo(BeEmpty())
			//GinkgoWriter.Printf("We expect the eip will drift to node: %s\n", nodeNameB)
			//Expect(common.WaitEipToExpectNode(f, nodeNameB, egressPolicy, time.Second*10)).NotTo(HaveOccurred())
			//
			//// check the running pod's export IP is eip
			//GinkgoWriter.Println("Check the eip in running pods after shut down the eip node")
			//list, err := common.ListNodesPod(f, podLabel, tools.SubtractionSlice(nodes, []string{nodeNameA}))
			//Expect(err).NotTo(HaveOccurred())
			//checkEip(list, v4Eip, v6Eip, true, 3, time.Second*5)
			//
			//// power on the node and wait cluster ready
			//GinkgoWriter.Println("PowerOnNodesUntilClusterReady")
			//Expect(common.PowerOnNodesUntilClusterReady(f, nodes, time.Second*30)).NotTo(HaveOccurred())
		})
	})
})
