// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egresspolicy_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"

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

		egw, err = common.CreateGatewayNew(ctx, cli, "egw-"+uuid.NewString(), pool, nodeSelector)
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

	/*
		These test cases mainly test some limiting checks when creating policies and cluster-policies to see if they meet expectations. It mainly includes the following checks:

		1. Using an illegal egressIP to create a policy will fail.
		2. When the manually specified egressIP of the policy is not in the IP pool range of the gateway used by this policy, the creation will fail.
		3. When Spec.AppliedTo of the policy is empty, the creation will fail.
		4. When the policy specifies both Spec.AppliedTo.PodSubnet and Spec.AppliedTo.PodSelector at the same time, the creation will fail.
		5. When Spec.EgressIP.UseNodeIP of the policy is true, but an egressIP is also specified at the same time, the creation will fail.
	*/
	Context("Creation test", func() {
		ctx := context.Background()
		var egp *egressv1.EgressPolicy
		var egcp *egressv1.EgressClusterPolicy
		var err error

		AfterEach(func() {
			// delete the policy if it is exist
			if egp != nil {
				err = common.WaitEgressPoliciesDeleted(ctx, cli, []*egressv1.EgressPolicy{egp}, time.Second*5)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		DescribeTable("namespaced policy", func(expectErr bool, setUp func(egp *egressv1.EgressPolicy)) {
			egp, err = common.CreateEgressPolicyCustom(ctx, cli, setUp)
			if expectErr {
				Expect(err).To(HaveOccurred(), fmt.Sprintf("egressPolicy yaml:\n%s\n", common.GetObjYAML(egp)))
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
			// todo @bzsuni waiting for the bug be fixed
			PEntry("should fail when the policy is set with invalid `EgressIP`", Label("P00001"), true, func(egp *egressv1.EgressPolicy) {
				egp.Spec.EgressGatewayName = egw.Name
				egp.Spec.AppliedTo.PodSubnet = []string{"10.10.0.0/16"}
				if egressConfig.EnableIPv4 {
					egp.Spec.EgressIP.IPv4 = "fddd:10::2"
				}
				if egressConfig.EnableIPv6 {
					egp.Spec.EgressIP.IPv6 = "10.10.10.2"
				}
			}),
			// todo @bzsuni waiting for the bug be fixed
			PEntry("should fail when the `Spec.EgressIP` of the policy is not within the IP range of the ippools in the gateway used by the policy", Label("P00004"), true,
				func(egp *egressv1.EgressPolicy) {
					egp.Spec.EgressGatewayName = egw.Name
					egp.Spec.AppliedTo.PodSubnet = []string{"10.10.0.0/16"}
					if egressConfig.EnableIPv4 {
						egp.Spec.EgressIP.IPv4 = "10.10.10.2"
					}
					if egressConfig.EnableIPv6 {
						egp.Spec.EgressIP.IPv6 = "fddd:10::2"
					}
				}),
			// todo @bzsuni waiting for the bug be fixed
			PEntry("should fail when Spec.AppliedTo is empty", Label("P00005"), true,
				func(egp *egressv1.EgressPolicy) {
					egp.Spec.EgressGatewayName = egw.Name
					egp.Spec.AppliedTo = egressv1.AppliedTo{}
				}),
			Entry("should fail when the policy set with both Spec.AppliedTo.PodSubnet and Spec.AppliedTo.PodSelector", Label("P00006"), true,
				func(egp *egressv1.EgressPolicy) {
					egp.Spec.EgressGatewayName = egw.Name
					egp.Spec.AppliedTo.PodSubnet = []string{"10.10.0.0/16"}
					egp.Spec.AppliedTo.PodSelector = &metav1.LabelSelector{MatchLabels: map[string]string{"a": "b"}}
				}),
			Entry("should fail when the `Spec.EgressIP.UseNodeIP` of the policy is set to true and the Spec.EgressIP is not empty", Label("P00017"), true,
				func(egp *egressv1.EgressPolicy) {
					egp.Spec.EgressGatewayName = egw.Name
					egp.Spec.AppliedTo.PodSubnet = []string{"10.10.0.0/16"}
					egp.Spec.EgressIP.UseNodeIP = true
					if egressConfig.EnableIPv4 {
						egp.Spec.EgressIP.IPv4 = egw.Spec.Ippools.Ipv4DefaultEIP
					}
					if egressConfig.EnableIPv6 {
						egp.Spec.EgressIP.IPv6 = egw.Spec.Ippools.Ipv6DefaultEIP
					}
				}),
		)

		DescribeTable("cluster policy", func(expectErr bool, setUp func(egp *egressv1.EgressClusterPolicy)) {
			egcp, err = common.CreateEgressClusterPolicyCustom(ctx, cli, setUp)
			if expectErr {
				Expect(err).To(HaveOccurred(), fmt.Sprintf("egressClusterPolicy yaml:\n%s\n", common.GetObjYAML(egcp)))
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
			// todo @bzsuni waiting for the bug be fixed
			PEntry("should fail when the cluster-policy is set with invalid `EgressIP`", Label("P00001"), true, func(egcp *egressv1.EgressClusterPolicy) {
				egcp.Spec.EgressGatewayName = egw.Name
				egcp.Spec.AppliedTo.PodSubnet = &[]string{"10.10.0.0/16"}
				if egressConfig.EnableIPv4 {
					egcp.Spec.EgressIP.IPv4 = "fddd:10::2"
				}
				if egressConfig.EnableIPv6 {
					egcp.Spec.EgressIP.IPv6 = "10.10.10.2"
				}
			}),
			// todo @bzsuni waiting for the bug be fixed
			PEntry("should fail when the `Spec.EgressIP` of the cluster-policy is not within the IP range of the ippools in the gateway used by the policy", Label("P00004"), true,
				func(egcp *egressv1.EgressClusterPolicy) {
					egcp.Spec.EgressGatewayName = egw.Name
					egcp.Spec.AppliedTo.PodSubnet = &[]string{"10.10.0.0/16"}
					if egressConfig.EnableIPv4 {
						egcp.Spec.EgressIP.IPv4 = "10.10.10.2"
					}
					if egressConfig.EnableIPv6 {
						egcp.Spec.EgressIP.IPv6 = "fddd:10::2"
					}
				}),

			// todo @bzsuni waiting for the bug be fixed
			PEntry("should fail when Spec.AppliedTo is empty", Label("P00005"), true,
				func(egcp *egressv1.EgressClusterPolicy) {
					egcp.Spec.EgressGatewayName = egw.Name
					egcp.Spec.AppliedTo = egressv1.ClusterAppliedTo{}
				}),
			// todo @bzsuni waiting for the bug be fixed
			PEntry("should fail when the cluster-policy set with both Spec.AppliedTo.PodSubnet and Spec.AppliedTo.PodSelector", Label("P00006"), true,
				func(egcp *egressv1.EgressClusterPolicy) {
					egcp.Spec.EgressGatewayName = egw.Name
					egcp.Spec.AppliedTo.PodSubnet = &[]string{"10.10.0.0/16"}
					egcp.Spec.AppliedTo.PodSelector = &metav1.LabelSelector{MatchLabels: map[string]string{"a": "b"}}
				}),
			// todo @bzsuni waiting for the bug be fixed
			PEntry("should fail when the `Spec.EgressIP.UseNodeIP` of the cluster-policy is set to true and the Spec.EgressIP is not empty", Label("P00017"), true,
				func(egcp *egressv1.EgressClusterPolicy) {
					egcp.Spec.EgressGatewayName = egw.Name
					egcp.Spec.AppliedTo.PodSubnet = &[]string{"10.10.0.0/16"}
					egcp.Spec.EgressIP.UseNodeIP = true
					if egressConfig.EnableIPv4 {
						egcp.Spec.EgressIP.IPv4 = egw.Spec.Ippools.Ipv4DefaultEIP
					}
					if egressConfig.EnableIPv6 {
						egcp.Spec.EgressIP.IPv6 = egw.Spec.Ippools.Ipv6DefaultEIP
					}
				}),
		)
	})
})
