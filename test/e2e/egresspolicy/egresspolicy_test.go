// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egresspolicy_test

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-faker/faker/v4"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("EgressPolicy", Ordered, func() {
	var egw *egressv1.EgressGateway

	BeforeAll(func() {
		ctx := context.Background()

		// create EgressGateway
		pool, err := common.GenIPPools(ctx, cli, egressConfig.EnableIPv4, egressConfig.EnableIPv6, 3, 1)
		Expect(err).NotTo(HaveOccurred())
		nodeSelector := egressv1.NodeSelector{Selector: &metav1.LabelSelector{MatchLabels: nodeLabel}}

		egw, err = common.CreateGatewayNew(ctx, cli, "egw-"+strings.ToLower(faker.FirstName())+faker.Word(), pool, nodeSelector)
		Expect(err).NotTo(HaveOccurred())
		GinkgoWriter.Printf("Create EgressGateway: %s\n", egw.Name)

		DeferCleanup(func() {
			// delete EgressGateway
			if egw != nil {
				err = common.DeleteObj(ctx, cli, egw)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	Context("Test EgressPolicy", Label("EgressPolicy", "P00007", "P00008", "P00013", "P00014", "P00019"), func() {
		var (
			dsA *appsv1.DaemonSet
			dsB *appsv1.DaemonSet

			policy        *egressv1.EgressPolicy
			clusterPolicy *egressv1.EgressClusterPolicy
		)

		BeforeEach(func() {
			ctx := context.Background()
			var err error
			// create DaemonSet-A DaemonSet-B for A/B test
			dsA, err = common.CreateDaemonSet(ctx, cli, "ds-a-"+faker.Word(), config.Image)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Create DaemonSet A: %s\n", dsA.Name)

			dsB, err = common.CreateDaemonSet(ctx, cli, "ds-b-"+faker.Word(), config.Image)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Create DaemonSet B: %s\n", dsB.Name)

			DeferCleanup(func() {
				// delete DaemonSet-A DaemonSet-B
				ctx := context.Background()
				err := common.DeleteObj(ctx, cli, dsA)
				Expect(err).NotTo(HaveOccurred())
				err = common.DeleteObj(ctx, cli, dsB)
				Expect(err).NotTo(HaveOccurred())

				// delete policy
				err = common.DeleteObj(ctx, cli, policy)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("test namespaced policy", func() {
			var err error
			ctx := context.Background()

			// P00008
			By("case P00008: create policy with empty `EgressIP`")

			policy, err = common.CreateEgressPolicyNew(ctx, cli, egressConfig, egw.Name, dsA.Labels)
			Expect(err).NotTo(HaveOccurred())

			GinkgoWriter.Printf("Create EgressPolicy: %s\n", policy.Name)
			time.Sleep(time.Second * 2)
			e := policy.Status.Eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsA, e.Ipv4, e.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())

			// P00011
			By("case P00011: update policy to empty `DestSubnet`")
			e = policy.Status.Eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsB, e.Ipv4, e.Ipv6, false)
			Expect(err).NotTo(HaveOccurred())

			// P00014
			By("case P00014: update policy matched dsA to match dsB")
			policy.Spec.AppliedTo.PodSelector.MatchLabels = dsB.Spec.Template.Labels
			err = cli.Update(ctx, policy)
			Expect(err).NotTo(HaveOccurred())

			// check dsB
			time.Sleep(time.Second * 2)
			e = policy.Status.Eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsB, e.Ipv4, e.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())

			// check dsA
			time.Sleep(time.Second * 2)
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsA, e.Ipv4, e.Ipv6, false)
			Expect(err).NotTo(HaveOccurred())

			// P00013
			By("case P00013: update policy to unmatched `DestSubnet`")

			policy.Spec.DestSubnet = []string{"1.1.1.1/32"}
			err = cli.Update(ctx, policy)
			Expect(err).NotTo(HaveOccurred())

			// check dsB
			time.Sleep(time.Second * 2)
			e = policy.Status.Eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsB, e.Ipv4, e.Ipv6, false)
			Expect(err).NotTo(HaveOccurred())

			// P00019
			By("case P00019: delete policy, we expect the egress address not egressIP")
			err = common.DeleteObj(ctx, cli, policy)
			Expect(err).NotTo(HaveOccurred())

			// check dsB
			time.Sleep(time.Second * 2)
			e = policy.Status.Eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsB, e.Ipv4, e.Ipv6, false)
			Expect(err).NotTo(HaveOccurred())
		})

		It("test cluster policy", func() {
			var err error
			ctx := context.Background()

			// P00008
			By("case P00008: create policy with empty `EgressIP`")

			clusterPolicy, err = common.CreateEgressClusterPolicy(ctx, cli, egressConfig, egw.Name, dsA.Labels)
			Expect(err).NotTo(HaveOccurred())

			GinkgoWriter.Printf("Create EgressClusterPolicy: %s\n", clusterPolicy.Name)
			time.Sleep(time.Second * 2)
			e := clusterPolicy.Status.Eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsA, e.Ipv4, e.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())

			// P00011
			By("case P00011: update policy to empty `DestSubnet`")
			e = clusterPolicy.Status.Eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsB, e.Ipv4, e.Ipv6, false)
			Expect(err).NotTo(HaveOccurred())

			// P00014
			By("case P00014: update policy matched dsA to match dsB")
			clusterPolicy.Spec.AppliedTo.PodSelector.MatchLabels = dsB.Spec.Template.Labels
			err = cli.Update(ctx, clusterPolicy)
			Expect(err).NotTo(HaveOccurred())

			// check dsB
			time.Sleep(time.Second * 2)
			e = clusterPolicy.Status.Eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsB, e.Ipv4, e.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())

			// check dsA
			time.Sleep(time.Second * 2)
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsA, e.Ipv4, e.Ipv6, false)
			Expect(err).NotTo(HaveOccurred())

			// P00013
			By("case P00013: update policy to unmatched `DestSubnet`")

			clusterPolicy.Spec.DestSubnet = []string{"1.1.1.1/32"}
			err = cli.Update(ctx, clusterPolicy)
			Expect(err).NotTo(HaveOccurred())

			// check dsB
			time.Sleep(time.Second * 2)
			e = clusterPolicy.Status.Eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsB, e.Ipv4, e.Ipv6, false)
			Expect(err).NotTo(HaveOccurred())

			// P00019
			By("case P00019: delete policy, we expect the egress address not egressIP")
			err = common.DeleteObj(ctx, cli, clusterPolicy)
			Expect(err).NotTo(HaveOccurred())

			// check dsB
			time.Sleep(time.Second * 2)
			e = clusterPolicy.Status.Eip
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, dsB, e.Ipv4, e.Ipv6, false)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
