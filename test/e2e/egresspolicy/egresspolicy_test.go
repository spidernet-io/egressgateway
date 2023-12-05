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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

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
				// todo @bzsuni waiting finalizer-feature to be done
				time.Sleep(time.Second * 3)
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
			// delete the policy if it is existed
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
			Entry("should fail when the policy is set with invalid `EgressIP`", Label("P00001"), true, func(egp *egressv1.EgressPolicy) {
				egp.Spec.EgressGatewayName = egw.Name
				egp.Spec.AppliedTo.PodSubnet = []string{"10.10.0.0/16"}
				if egressConfig.EnableIPv4 {
					egp.Spec.EgressIP.IPv4 = "fddd:10::2"
				}
				if egressConfig.EnableIPv6 {
					egp.Spec.EgressIP.IPv6 = "10.10.10.2"
				}
			}),
			Entry("should fail when the `Spec.EgressIP` of the policy is not within the IP range of the ippools in the gateway used by the policy", Label("P00004"), true,
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
			Entry("should fail when Spec.AppliedTo is empty", Label("P00005"), true,
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
			Entry("should fail when the cluster-policy is set with invalid `EgressIP`", Label("P00001"), true, func(egcp *egressv1.EgressClusterPolicy) {
				egcp.Spec.EgressGatewayName = egw.Name
				egcp.Spec.AppliedTo.PodSubnet = &[]string{"10.10.0.0/16"}
				if egressConfig.EnableIPv4 {
					egcp.Spec.EgressIP.IPv4 = "fddd:10::2"
				}
				if egressConfig.EnableIPv6 {
					egcp.Spec.EgressIP.IPv6 = "10.10.10.2"
				}
			}),
			Entry("should fail when the `Spec.EgressIP` of the cluster-policy is not within the IP range of the ippools in the gateway used by the policy", Label("P00004"), true,
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
			Entry("should fail when Spec.AppliedTo is empty", Label("P00005"), true,
				func(egcp *egressv1.EgressClusterPolicy) {
					egcp.Spec.EgressGatewayName = egw.Name
					egcp.Spec.AppliedTo = egressv1.ClusterAppliedTo{}
				}),
			Entry("should fail when the cluster-policy set with both Spec.AppliedTo.PodSubnet and Spec.AppliedTo.PodSelector", Label("P00006"), true,
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

		/*
			This test case tests some validations after updating the gateway when EgressIP.UseNodeIP is set to true when creating a policy. The test steps are as follows:
			1. Create a gateway and specify the nodeSelector as node1
			2. Create a policy and set EgressIP.UseNodeIP to true
			3. Validate the status of the gateway and policy, verify the pod's egress IP should be the IP of node1
			4. Update the gateway to change the match from node1 to node2
			5. Validate the status of the gateway and policy, verify the pod's egress IP should be the IP of node2
		*/
		Context("Create policy with setting EgressIP.UseNodeIP to be true", Label("P00015", "P00016"), func() {
			var egw *egressv1.EgressGateway
			var egp *egressv1.EgressPolicy
			var egcp *egressv1.EgressClusterPolicy
			var ctx context.Context
			var err error

			var podLabelSelector *metav1.LabelSelector
			var node2Selector egressv1.NodeSelector

			var node1IPv4, node1IPv6 string
			var node2IPv4, node2IPv6 string

			var ds *appsv1.DaemonSet

			BeforeEach(func() {
				ctx = context.Background()

				// create DaemonSet
				ds, err = common.CreateDaemonSet(ctx, cli, "ds-"+uuid.NewString(), config.Image)
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Printf("succeeded to create DaemonSet: %s\n", ds.Name)
				podLabelSelector = &metav1.LabelSelector{MatchLabels: ds.Labels}

				// get nodeIP
				node1IPv4, node1IPv6 = common.GetNodeIP(node1)
				GinkgoWriter.Printf("node: %s, ipv4: %s, ipv6: %s\n", node1.Name, node1IPv4, node1IPv6)

				node2IPv4, node2IPv6 = common.GetNodeIP(node2)
				GinkgoWriter.Printf("node: %s, ipv4: %s, ipv6: %s\n", node2.Name, node2IPv4, node2IPv6)

				node1LabelSelector := &metav1.LabelSelector{MatchLabels: node1.Labels}
				node2LabelSelector := &metav1.LabelSelector{MatchLabels: node2.Labels}

				node1Selector := egressv1.NodeSelector{
					Selector: node1LabelSelector,
				}
				node2Selector = egressv1.NodeSelector{
					Selector: node2LabelSelector,
				}

				// create gateway with empty ippools
				egw, err = common.CreateGatewayNew(ctx, cli, "egw-"+uuid.NewString(), egressv1.Ippools{}, node1Selector)
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Printf("Succeeded to create egw:\n%s\n", common.GetObjYAML(egw))

				DeferCleanup(func() {
					// delete daemonset
					Expect(common.DeleteObj(ctx, cli, ds)).NotTo(HaveOccurred())

					// delete EgressPolicy
					if egp != nil {
						GinkgoWriter.Printf("Delete egp: %s\n", egp.Name)
						err = common.WaitEgressPoliciesDeleted(ctx, cli, []*egressv1.EgressPolicy{egp}, time.Second*5)
						Expect(err).NotTo(HaveOccurred())
					}

					// delete EgressClusterPolicy
					if egcp != nil {
						GinkgoWriter.Printf("Delete egcp: %s\n", egcp.Name)
						err = common.WaitEgressClusterPoliciesDeleted(ctx, cli, []*egressv1.EgressClusterPolicy{egcp}, time.Second*5)
						Expect(err).NotTo(HaveOccurred())
					}

					// delete egw
					if egw != nil {
						GinkgoWriter.Printf("Delete egw: %s\n", egw.Name)
						Expect(common.DeleteObj(ctx, cli, egw)).NotTo(HaveOccurred())
					}
				})
			})

			It("namespace-level policy", func() {
				egp, err = common.CreateEgressPolicyCustom(ctx, cli,
					func(egp *egressv1.EgressPolicy) {
						egp.Spec.EgressGatewayName = egw.Name
						egp.Spec.EgressIP.UseNodeIP = true
						egp.Spec.AppliedTo.PodSelector = podLabelSelector
					})

				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("egp:\n%s\n", common.GetObjYAML(egp)))
				// check if the egressgateway synced successfully
				expectStatus := &egressv1.EgressGatewayStatus{
					NodeList: []egressv1.EgressIPStatus{
						{
							Name: node1.Name,
							Eips: []egressv1.Eips{
								{Policies: []egressv1.Policy{
									{Name: egp.Name, Namespace: egp.Namespace},
								}},
							},
							Status: string(egressv1.EgressTunnelReady),
						},
					},
				}
				err = common.CheckEgressGatewayStatusSynced(ctx, cli, egw, expectStatus, time.Second*3)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed CheckEgressGatewayStatusSynced, egressgateway:\n%s\n", common.GetObjYAML(egw)))
				// check the pod export IP
				err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, ds, node1IPv4, node1IPv6, true)
				Expect(err).NotTo(HaveOccurred())

				// update the `NodeSelector` of the gateway to change the match from node1 to node2
				GinkgoWriter.Printf("update the gateway: %s, to change the match from node: %s to node: %s\n", egw.Name, node1.Name, node2.Name)
				egw.Spec.NodeSelector = node2Selector
				// todo @bzsuni waiting for the bug to be fixed
				Expect(cli.Update(ctx, egw)).NotTo(HaveOccurred(), fmt.Sprintf("failed to update gateway:\n%s\n", common.GetObjYAML(egw)))
				// check if the egressgateway synced successfully
				expectStatus = &egressv1.EgressGatewayStatus{
					NodeList: []egressv1.EgressIPStatus{
						{
							Name: node2.Name,
							Eips: []egressv1.Eips{
								{Policies: []egressv1.Policy{
									{Name: egp.Name, Namespace: egp.Namespace},
								}},
							},
							Status: string(egressv1.EgressTunnelReady),
						},
					},
				}
				err = common.CheckEgressGatewayStatusSynced(ctx, cli, egw, expectStatus, time.Second*3)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed CheckEgressGatewayStatusSynced, egressgateway:\n%s\n", common.GetObjYAML(egw)))
				// check the pod export IP
				err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, ds, node2IPv4, node2IPv6, true)
				Expect(err).NotTo(HaveOccurred())
			})

			It("cluster-level policy", func() {
				egcp, err = common.CreateEgressClusterPolicyCustom(ctx, cli,
					func(egcp *egressv1.EgressClusterPolicy) {
						egcp.Spec.EgressGatewayName = egw.Name
						egcp.Spec.EgressIP.UseNodeIP = true
						egcp.Spec.AppliedTo.PodSelector = podLabelSelector
					})
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("egcp:\n%s\n", common.GetObjYAML(egcp)))
				// check if the egressgateway synced successfully
				expectStatus := &egressv1.EgressGatewayStatus{
					NodeList: []egressv1.EgressIPStatus{
						{
							Name: node1.Name,
							Eips: []egressv1.Eips{
								{Policies: []egressv1.Policy{
									{Name: egcp.Name},
								}},
							},
							Status: string(egressv1.EgressTunnelReady),
						},
					},
				}
				err = common.CheckEgressGatewayStatusSynced(ctx, cli, egw, expectStatus, time.Second*3)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed CheckEgressGatewayStatusSynced, egressgateway:\n%s\n", common.GetObjYAML(egw)))
				// check the pod export IP
				err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, ds, node1IPv4, node1IPv6, true)
				Expect(err).NotTo(HaveOccurred())

				// update the `NodeSelector` of the gateway to change the match from node1 to node2
				GinkgoWriter.Printf("update the gateway: %s, to change the match from node: %s to node: %s\n", egw.Name, node1.Name, node2.Name)
				egw.Spec.NodeSelector = node2Selector
				// todo @bzsuni waiting for the bug to be fixed
				Expect(cli.Update(ctx, egw)).NotTo(HaveOccurred(), fmt.Sprintf("failed to update gateway:\n%s\n", common.GetObjYAML(egw)))
				// check if the egressgateway synced successfully
				expectStatus = &egressv1.EgressGatewayStatus{
					NodeList: []egressv1.EgressIPStatus{
						{
							Name: node2.Name,
							Eips: []egressv1.Eips{
								{Policies: []egressv1.Policy{
									{Name: egcp.Name},
								}},
							},
							Status: string(egressv1.EgressTunnelReady),
						},
					},
				}
				err = common.CheckEgressGatewayStatusSynced(ctx, cli, egw, expectStatus, time.Second*3)
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed CheckEgressGatewayStatusSynced, egressgateway:\n%s\n", common.GetObjYAML(egw)))
				// check the pod export IP
				err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, ds, node2IPv4, node2IPv6, true)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	/*
		This test case is used to verify that the policy does not allow editing of the fields Spec.EgressIP.IP and Spec.EgressGatewayName
		We expect that when these two fields are edited, the request will be rejected
	*/
	Context("Edit policy", Label("P00018", "P00019"), func() {
		var egw1 *egressv1.EgressGateway
		var egp *egressv1.EgressPolicy
		var egcp *egressv1.EgressClusterPolicy
		var ctx context.Context
		var err error
		var pool egressv1.Ippools

		BeforeEach(func() {
			ctx = context.Background()

			// create EgressGateway
			if egressConfig.EnableIPv4 {
				pool.IPv4 = []string{"10.10.10.1", "10.10.10.2"}
			}
			if egressConfig.EnableIPv6 {
				pool.IPv6 = []string{"fddd:10::1", "fddd:10::2"}
			}

			nodeSelector := egressv1.NodeSelector{Selector: &metav1.LabelSelector{MatchLabels: nodeLabel}}

			egw1, err = common.CreateGatewayNew(ctx, cli, "egw1-"+uuid.NewString(), pool, nodeSelector)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Create EgressGateway: %s\n", egw1.Name)

			DeferCleanup(func() {

				// delete EgressPolicy
				if egp != nil {
					GinkgoWriter.Printf("Delete egp: %s\n", egp.Name)
					err = common.WaitEgressPoliciesDeleted(ctx, cli, []*egressv1.EgressPolicy{egp}, time.Second*5)
					Expect(err).NotTo(HaveOccurred())
				}

				// delete EgressClusterPolicy
				if egcp != nil {
					GinkgoWriter.Printf("Delete egcp: %s\n", egcp.Name)
					err = common.WaitEgressClusterPoliciesDeleted(ctx, cli, []*egressv1.EgressClusterPolicy{egcp}, time.Second*5)
					Expect(err).NotTo(HaveOccurred())
				}

				// delete egw
				if egw1 != nil {
					// todo @bzsuni waiting for the finalizer-feature to be done
					time.Sleep(time.Second * 2)
					GinkgoWriter.Printf("Delete egw: %s\n", egw1.Name)
					Expect(common.DeleteObj(ctx, cli, egw1)).NotTo(HaveOccurred())
				}
			})
		})

		It("namespace-level policy", func() {
			// create EgressPolicy
			egp, err = common.CreateEgressPolicyCustom(ctx, cli,
				func(egp *egressv1.EgressPolicy) {
					egp.Spec.EgressGatewayName = egw1.Name

					newEgressGateway := new(egressv1.EgressGateway)
					err := cli.Get(ctx, types.NamespacedName{Name: egw1.Name}, newEgressGateway)
					Expect(err).NotTo(HaveOccurred())

					if egressConfig.EnableIPv4 {
						egp.Spec.EgressIP.IPv4 = newEgressGateway.Spec.Ippools.Ipv4DefaultEIP
					}
					if egressConfig.EnableIPv6 {
						egp.Spec.EgressIP.IPv6 = newEgressGateway.Spec.Ippools.Ipv6DefaultEIP
					}
					egp.Spec.AppliedTo.PodSubnet = []string{"10.10.0.0/18"}
				})
			GinkgoWriter.Printf("the policy yaml:\n%s\n", common.GetObjYAML(egp))
			Expect(err).NotTo(HaveOccurred())

			cpEgp := egp.DeepCopy()
			// edit policy Spec.EgressIP.IPv4 and Spec.EgressIP.IPv6
			if egressConfig.EnableIPv4 {
				for _, val := range pool.IPv4 {
					if val != egp.Spec.EgressIP.IPv4 {
						egp.Spec.EgressIP.IPv4 = val
						break
					}
				}
			}
			if egressConfig.EnableIPv6 {
				for _, val := range pool.IPv6 {
					if val != egp.Spec.EgressIP.IPv6 {
						egp.Spec.EgressIP.IPv6 = val
						break
					}
				}
			}
			// update policy EgressIP.IPv4 or EgressIP.IPv6
			Expect(cli.Update(ctx, egp)).To(HaveOccurred())

			// edit policy Spec.
			cpEgp.Spec.EgressGatewayName = egw.Name
			// update policy
			Expect(cli.Update(ctx, cpEgp)).To(HaveOccurred())
		})

		// todo @bzsuni waiting for the bug to be fixed
		PIt("cluster-level policy", func() {
			// create EgressClusterPolicy
			egcp, err = common.CreateEgressClusterPolicyCustom(ctx, cli,
				func(egcp *egressv1.EgressClusterPolicy) {
					egcp.Spec.EgressGatewayName = egw1.Name
					if egressConfig.EnableIPv4 {
						egcp.Spec.EgressIP.IPv4 = pool.IPv4[0]
					}
					if egressConfig.EnableIPv6 {
						egcp.Spec.EgressIP.IPv6 = pool.IPv6[0]
					}
					egcp.Spec.AppliedTo.PodSubnet = &[]string{"10.10.0.0/18"}
				})
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("the cluster policy yaml:\n%s\n", common.GetObjYAML(egcp))

			cpEgcp := egcp.DeepCopy()
			// edit policy Spec.EgressIP.IPv4 and Spec.EgressIP.IPv6
			if egressConfig.EnableIPv4 {
				egcp.Spec.EgressIP.IPv4 = pool.IPv4[1]
			}
			if egressConfig.EnableIPv6 {
				egcp.Spec.EgressIP.IPv6 = pool.IPv6[1]
			}
			// update policy EgressIP.IPv4 or EgressIP.IPv6
			Expect(cli.Update(ctx, egcp)).To(HaveOccurred())

			// edit policy Spec.
			cpEgcp.Spec.EgressGatewayName = egw.Name
			// update policy
			Expect(cli.Update(ctx, cpEgcp)).To(HaveOccurred())
		})
	})

	/*
	   namespace-level policy only takes effect in its specified namespace
	    1. Create namespace test-ns
	    2. Create pods with the same name in default and test-ns namespaces respectively
	    3. Create a policy in default namespace, with PodSelector matching the labels of the above pods
	    4. Check the egress IP of the pod in default namespace should be the eip of the policy
	    5. Check the egress IP of the pod in test-ns namespace should NOT be the eip of the policy
	*/
	Context("namespace-level policy", Label("P00021"), func() {
		var ctx context.Context
		var testNs *corev1.Namespace
		var podName string
		var podObj, podObjNs *corev1.Pod
		var podLabel map[string]string
		var err error
		var egp *egressv1.EgressPolicy

		BeforeEach(func() {
			ctx = context.Background()
			podName = "pod-" + uuid.NewString()
			podLabel = map[string]string{"app": podName}

			DeferCleanup(func() {
				// delete ns
				if testNs != nil {
					Expect(common.DeleteObj(ctx, cli, testNs)).NotTo(HaveOccurred())
					Eventually(ctx, func(ctx context.Context) bool {
						e := cli.Get(ctx, types.NamespacedName{Name: testNs.Name}, testNs)
						return errors.IsNotFound(e)
					}).WithTimeout(time.Second * 10).WithPolling(time.Second).Should(BeTrue())
				}
				// delete pods
				if podObj != nil {
					Expect(common.DeleteObj(ctx, cli, podObj)).NotTo(HaveOccurred())
				}

				// delete EgressPolicy
				if egp != nil {
					// Expect(common.DeleteObj(ctx, cli, egp)).NotTo(HaveOccurred())
					err = common.WaitEgressPoliciesDeleted(ctx, cli, []*egressv1.EgressPolicy{egp}, time.Second*5)
					Expect(err).NotTo(HaveOccurred())
					time.Sleep(time.Second * 2)
				}
			})
		})

		It("test the scope of policy", func() {
			// create ns test-ns
			testNs = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			}
			Expect(cli.Create(ctx, testNs)).NotTo(HaveOccurred())

			// create a pod in default namespace
			podObj, err = common.CreatePodCustom(ctx, cli, podName, config.Image, func(pod *corev1.Pod) {
				pod.Namespace = "default"
				pod.Labels = podLabel
			})
			Expect(err).NotTo(HaveOccurred())

			// create a name-same pod in namespace test-ns
			podObjNs, err = common.CreatePodCustom(ctx, cli, podName, config.Image, func(pod *corev1.Pod) {
				pod.Namespace = testNs.Name
				pod.Labels = podLabel
			})
			Expect(err).NotTo(HaveOccurred())

			// waiting for the pod to be created
			Expect(common.WaitPodRunning(ctx, cli, podObj, time.Second*10)).NotTo(HaveOccurred())

			// create a policy in default namespace
			egp, err = common.CreateEgressPolicyNew(ctx, cli, egressConfig, egw.Name, podLabel)
			Expect(err).NotTo(HaveOccurred())
			err = common.WaitEgressPolicyStatusReady(ctx, cli, egp, egressConfig.EnableIPv4, egressConfig.EnableIPv6, time.Second*3)
			Expect(err).NotTo(HaveOccurred())

			// check the eip of the pod in default namespace
			if egressConfig.EnableIPv4 {
				err = common.CheckPodEgressIP(ctx, config, *podObj, egp.Status.Eip.Ipv4, config.ServerAIPv4, true)
				Expect(err).NotTo(HaveOccurred())
			}
			if egressConfig.EnableIPv6 {
				err = common.CheckPodEgressIP(ctx, config, *podObj, egp.Status.Eip.Ipv6, config.ServerAIPv6, true)
				Expect(err).NotTo(HaveOccurred())
			}

			// check the eip of the pod in the namespace `test-ns`
			if egressConfig.EnableIPv4 {
				err = common.CheckPodEgressIP(ctx, config, *podObjNs, egp.Status.Eip.Ipv4, config.ServerAIPv4, false)
				Expect(err).NotTo(HaveOccurred())
			}
			if egressConfig.EnableIPv6 {
				err = common.CheckPodEgressIP(ctx, config, *podObjNs, egp.Status.Eip.Ipv6, config.ServerAIPv6, false)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})
