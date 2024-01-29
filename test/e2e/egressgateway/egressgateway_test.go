// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"

	"github.com/go-faker/faker/v4"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/spidernet-io/egressgateway/pkg/constant"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

var _ = Describe("Operate EgressGateway", Label("EgressGateway"), Ordered, func() {
	Context("Create egressGateway", func() {
		var labels map[string]string
		var gatewayName string
		var egw *egressv1.EgressGateway
		var ctx context.Context

		var (
			badDefaultIPv4, badDefaultIPv6 string
			invalidIPv4, invalidIPv6       string
			singleIpv4Pool, singleIpv6Pool []string
			rangeIpv4Pool, rangeIpv6Pool   []string
			cidrIpv4Pool, cidrIpv6Pool     []string
		)
		var labelSelector *metav1.LabelSelector

		BeforeEach(func() {
			ctx = context.Background()
			egw = new(egressv1.EgressGateway)

			// single Ippools
			singleIpv4Pool, singleIpv6Pool = make([]string, 0), make([]string, 0)
			// range Ippools
			rangeIpv4Pool, rangeIpv6Pool = make([]string, 0), make([]string, 0)
			// cidr Ippools
			cidrIpv4Pool, cidrIpv6Pool = make([]string, 0), make([]string, 0)

			gatewayName = tools.GenerateRandomName("egw")
			labels = map[string]string{gateway: gatewayName}

			labelSelector = &metav1.LabelSelector{MatchLabels: labels}

			if egressConfig.EnableIPv4 {
				badDefaultIPv4 = "11.10.0.1"
				invalidIPv4 = "invalidIPv4"
				singleIpv4Pool = []string{common.RandomIPV4()}
			}
			if egressConfig.EnableIPv6 {
				badDefaultIPv6 = "fdde:10::1"
				invalidIPv6 = "invalidIPv6"
				singleIpv6Pool = []string{common.RandomIPV6()}
			}

			GinkgoWriter.Printf("singleIpv4Pool: %s, singleIpv6Pool: %s\n", singleIpv4Pool, singleIpv6Pool)

			// DeferCleanup(func() {
			// 	// delete EgressGateway
			// 	if egw != nil {
			// 		err := common.DeleteObj(ctx, cli, egw)
			// 		Expect(err).NotTo(HaveOccurred())
			// 	}
			// })
		})

		/*
			This test table assesses various scenarios in which the creation of an egressgateway should fail:

			(1) Creating with an invalid IP pool will result in failure.
			(2) If the NodeSelector is empty, the creation will fail.
			(3) When the specified defaultEIP is not within the IP pool range, the creation will fail.
			(4) In a dual-stack environment, if the number of IP addresses in the IPv4 pool differs from the IPv6 pool, the creation will fail.
		*/
		DescribeTable("Failed to create egressGateway", func(setUp func(*egressv1.EgressGateway)) {
			var err error
			egw, err = common.CreateGatewayCustom(ctx, cli, setUp)
			Expect(err).To(HaveOccurred(), fmt.Sprintf("unexpectedresult, egw yaml:\n%s\n", common.GetObjYAML(egw)))
		},
			Entry("When `Ippools` is invalid", Label("G00001"), func(egw *egressv1.EgressGateway) {
				egw.Spec.Ippools = egressv1.Ippools{IPv4: []string{invalidIPv4}, IPv6: []string{invalidIPv6}}
			}),
			// TODO @bzsuni waiting for the bug to be fixed
			PEntry("When `NodeSelector` is empty", Label("G00002"), func(egw *egressv1.EgressGateway) {
				egw.Spec.Ippools = egressv1.Ippools{IPv4: singleIpv4Pool, IPv6: singleIpv6Pool}
			}),
			Entry("When `defaultEIP` is not in `Ippools`", Label("G00003"), func(egw *egressv1.EgressGateway) {
				egw.Spec.Ippools = egressv1.Ippools{
					IPv4:           singleIpv6Pool,
					IPv6:           singleIpv6Pool,
					Ipv4DefaultEIP: badDefaultIPv4,
					Ipv6DefaultEIP: badDefaultIPv6,
				}
				egw.Spec.NodeSelector = egressv1.NodeSelector{
					Policy:   common.AVERAGE_SELECTION,
					Selector: labelSelector,
				}
			}),
		)

		if egressConfig.EnableIPv4 && egressConfig.EnableIPv6 {
			DescribeTable("Failed to create egressGateway", func(setUp func(*egressv1.EgressGateway)) {
				var err error
				egw, err = common.CreateGatewayCustom(ctx, cli, setUp)
				Expect(err).To(HaveOccurred())
			},
				Entry("When the count of pools.IPv4 differs from pools.IPv6 in a dual cluster.", Label("G00004"),
					func(egw *egressv1.EgressGateway) {
						egw.Spec.Ippools = egressv1.Ippools{IPv4: singleIpv4Pool, IPv6: []string{}}
						egw.Spec.NodeSelector = egressv1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector}
					}))
		}

		/*
			This test table assesses three supported IP pool formats for creating an egressgateway:

			(1) Scenarios with a single individual IP.
			(2) Scenarios with a range of IP addresses.
			(3) Scenarios using CIDR notation.
		*/
		DescribeTable("Succeeded to create egressGateway", func(setUp func(*egressv1.EgressGateway)) {
			var err error
			egw, err = common.CreateGatewayCustom(ctx, cli, setUp)
			Expect(err).NotTo(HaveOccurred())
		},
			Entry("when `Ippools` is a single IP", Label("G00006"), func(egw *egressv1.EgressGateway) {
				egw.Spec.Ippools = egressv1.Ippools{IPv4: singleIpv4Pool, IPv6: singleIpv6Pool}
				egw.Spec.NodeSelector = egressv1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector}
			}),

			Entry("when `Ippools` is a IP range like `a-b`", Label("G00007"), func(egw *egressv1.EgressGateway) {
				egw.Spec.Ippools = egressv1.Ippools{IPv4: rangeIpv4Pool, IPv6: rangeIpv6Pool}
				egw.Spec.NodeSelector = egressv1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector}
			}),
			// TODO @bzsuni waiting for the bug to be fixed
			PEntry("when `Ippools` is a IP CIDR", Label("G00008"), func(egw *egressv1.EgressGateway) {
				egw.Spec.Ippools = egressv1.Ippools{IPv4: cidrIpv4Pool, IPv6: cidrIpv6Pool}
				egw.Spec.NodeSelector = egressv1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector}
			}),
		)
	})

	/*
		This test table assesses several scenarios involving the creation of an egressgateway using an empty IP pool and subsequently creating policies with this gateway:

		(1) When creating an egressPolicy or egressClusterPolicy and specifying useNodeIP as false, the policy creation will fail.
		(2) When creating an egressPolicy or egressClusterPolicy with useNodeIP set to true, the policy creation will succeed,
			and subsequent verification will confirm that the egress IP of pods using the policy matches the specified node's IP.
	*/
	Context("Create egressGateway with empty ippools", Label("G00018", "G00019"), func() {
		var egw *egressv1.EgressGateway
		var egp *egressv1.EgressPolicy
		var egcp *egressv1.EgressClusterPolicy
		var ctx context.Context
		var err error

		var nodeLabelSelector, podLabelSelector *metav1.LabelSelector

		var nodeIPv4, nodeIPv6 string

		var ds *appsv1.DaemonSet

		BeforeEach(func() {
			ctx = context.Background()

			// create DaemonSet
			ds, err = common.CreateDaemonSet(ctx, cli, "ds-"+faker.Word(), config.Image, time.Minute/2)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("succeeded to create DaemonSet: %s\n", ds.Name)
			podLabelSelector = &metav1.LabelSelector{MatchLabels: ds.Labels}

			// get nodeIP
			nodeIPv4, nodeIPv6 = common.GetNodeIP(node1)
			GinkgoWriter.Printf("node: %s, ipv4: %s, ipv6: %s\n", node1.Name, nodeIPv4, nodeIPv6)

			nodeLabelSelector = &metav1.LabelSelector{MatchLabels: node1.Labels}

			// create gateway with empty ippools
			nodeSelector := egressv1.NodeSelector{
				Selector: nodeLabelSelector,
			}
			egw, err = common.CreateGatewayNew(ctx, cli, "egw-"+uuid.NewString(), egressv1.Ippools{}, nodeSelector)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Succeeded to create egw:\n%s\n", common.GetObjYAML(egw))

			DeferCleanup(func() {
				// delete daemonset
				Expect(common.DeleteObj(ctx, cli, ds)).NotTo(HaveOccurred())

				// delete egresspolicy
				if egp != nil {
					GinkgoWriter.Printf("Delete egp: %s\n", egp.Name)
					err = common.WaitEgressPoliciesDeleted(ctx, cli, []*egressv1.EgressPolicy{egp}, time.Second*5)
					Expect(err).NotTo(HaveOccurred())
				}

				// delete egressclusterpolicy
				if egcp != nil {
					GinkgoWriter.Printf("Delete egcp: %s\n", egcp.Name)
					err = common.WaitEgressClusterPoliciesDeleted(ctx, cli, []*egressv1.EgressClusterPolicy{egcp}, time.Second*5)
					Expect(err).NotTo(HaveOccurred())
				}

				// delete egw
				if egw != nil {
					time.Sleep(time.Second * 3)
					GinkgoWriter.Printf("Delete egw: %s\n", egw.Name)
					Expect(common.DeleteObj(ctx, cli, egw)).NotTo(HaveOccurred())
				}
			})
		})

		// create egressPolicy
		DescribeTable("createPolicy", func(expect bool, setup func(*egressv1.EgressPolicy)) {
			egp, err = common.CreateEgressPolicyCustom(ctx, cli, setup)
			if expect {
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
				err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, ds, nodeIPv4, nodeIPv6, true)
				Expect(err).NotTo(HaveOccurred())

			} else {
				Expect(err).To(HaveOccurred(), fmt.Sprintf("egp:\n%s\n", common.GetObjYAML(egp)))
			}
		},
			Entry("should be failed when spec.egressIP.useNodeIP is false", false, func(egp *egressv1.EgressPolicy) {
				egp.Spec.EgressGatewayName = egw.Name
				egp.Spec.EgressIP.UseNodeIP = false
				egp.Spec.AppliedTo.PodSelector = podLabelSelector
			}),
			Entry("should be succeeded when spec.egressIP.useNodeIP is true", true, func(egp *egressv1.EgressPolicy) {
				egp.Spec.EgressGatewayName = egw.Name
				egp.Spec.EgressIP.UseNodeIP = true
				egp.Spec.AppliedTo.PodSelector = podLabelSelector
			}),
		)

		// create egressClusterPolicy
		DescribeTable("createClusterPolicy", func(expect bool, setup func(*egressv1.EgressClusterPolicy)) {
			egcp, err = common.CreateEgressClusterPolicyCustom(ctx, cli, setup)
			if expect {
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
				err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, ds, nodeIPv4, nodeIPv6, true)
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred(), fmt.Sprintf("egcp:\n%s\n", common.GetObjYAML(egcp)))
			}
		},
			Entry("should be failed when spec.egressIP.useNodeIP is false", false, func(egcp *egressv1.EgressClusterPolicy) {
				egcp.Spec.EgressGatewayName = egw.Name
				egcp.Spec.EgressIP.UseNodeIP = false
				egcp.Spec.AppliedTo.PodSelector = podLabelSelector
			}),
			Entry("should be succeeded when spec.egressIP.useNodeIP is true", true, func(egcp *egressv1.EgressClusterPolicy) {
				egcp.Spec.EgressGatewayName = egw.Name
				egcp.Spec.EgressIP.UseNodeIP = true
				egcp.Spec.AppliedTo.PodSelector = podLabelSelector
			}),
		)
	})

	/*
		This test table primarily evaluates the editing of an egress gateway with valid or invalid configurations to determine if the outcomes match the expectations:

		(1) When adding an invalid IP address to the IP pool, it should fail.
		(2) When adding a valid IP address to the IP pool, it should succeed.
		(3) When attempting to delete an IP that is already in use, it should fail.
		(4) In a dual-stack scenario, if a different number of IPs is added to the IPv4 and IPv6 pools, it should fail.
	*/
	Context("Edit egressGateway", func() {
		var ctx context.Context
		var egw *egressv1.EgressGateway
		var v4DefaultEip, v6DefaultEip string
		var pool egressv1.Ippools

		var (
			invalidIPv4, invalidIPv6       string
			singleIpv4Pool, singleIpv6Pool []string
		)

		BeforeEach(func() {
			ctx = context.Background()
			egw = new(egressv1.EgressGateway)

			singleIpv4Pool, singleIpv6Pool = make([]string, 0), make([]string, 0)

			if egressConfig.EnableIPv4 {
				invalidIPv4 = "invalidIPv4"
				singleIpv4Pool = []string{common.RandomIPV4()}
			}
			if egressConfig.EnableIPv6 {
				invalidIPv6 = "invalidIPv6"
				singleIpv6Pool = []string{common.RandomIPV6()}
			}

			GinkgoWriter.Printf("singleIpv4Pool: %s, singleIpv6Pool: %s\n", singleIpv4Pool, singleIpv6Pool)

			// create gateway
			egw = createEgressGateway(ctx)
			pool = egw.Spec.Ippools
			v4DefaultEip = pool.Ipv4DefaultEIP
			v6DefaultEip = pool.Ipv6DefaultEIP

			DeferCleanup(func() {
				// delete EgressGateway
				if egw != nil {
					err := common.DeleteEgressGateway(ctx, cli, egw, time.Minute/2)
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})

		DescribeTable("Edit egressGateway", func(expectedErr bool, update func(egw *egressv1.EgressGateway)) {
			// if not expected, error occurred
			GinkgoWriter.Printf("Update EgressGateway: %s\n", egw.Name)
			update(egw)
			err := common.UpdateEgressGateway(ctx, cli, egw)
			if expectedErr {
				if err == nil {
					raw := common.GetObjYAML(egw)
					GinkgoWriter.Printf("EgressGateway YAML:\n%s\n", raw)
				}
				Expect(err).To(HaveOccurred())
			} else {
				if err != nil {
					raw := common.GetObjYAML(egw)
					GinkgoWriter.Printf("EgressGateway YAML:\n%s\n", raw)
				}
				Expect(err).NotTo(HaveOccurred())
			}
		},
			Entry("Failed when add invalid `IP` to `Ippools`", Label("G00009"), true, func(egw *egressv1.EgressGateway) {
				raw := common.GetObjYAML(egw)
				GinkgoWriter.Printf("EgressGateway YAML, Update before:\n%s\n", raw)

				if egressConfig.EnableIPv4 {
					egw.Spec.Ippools.IPv4 = append(egw.Spec.Ippools.IPv4, invalidIPv4)
				}
				if egressConfig.EnableIPv6 {
					egw.Spec.Ippools.IPv6 = append(egw.Spec.Ippools.IPv6, invalidIPv6)
				}
			}),
			Entry("Succeeded when add valid `IP` to `Ippools`", Label("G00012", "G00013"), false, func(egw *egressv1.EgressGateway) {
				if egressConfig.EnableIPv4 {
					egw.Spec.Ippools.IPv4 = append(egw.Spec.Ippools.IPv4, singleIpv4Pool...)
				}
				if egressConfig.EnableIPv6 {
					egw.Spec.Ippools.IPv6 = append(egw.Spec.Ippools.IPv6, singleIpv6Pool...)
				}
			}),
			Entry("Failed when delete `IP` that being used", Label("G00010"), true, func(egw *egressv1.EgressGateway) {
				raw := common.GetObjYAML(egw)
				GinkgoWriter.Printf("EgressGateway YAML, Update before:\n%s\n", raw)
				if egressConfig.EnableIPv4 {
					egw.Spec.Ippools.IPv4 = tools.RemoveValueFromSlice(egw.Spec.Ippools.IPv4, v4DefaultEip)
				}
				if egressConfig.EnableIPv6 {
					egw.Spec.Ippools.IPv6 = tools.RemoveValueFromSlice(egw.Spec.Ippools.IPv6, v6DefaultEip)
				}
			}),
		)

		if egressConfig.EnableIPv4 && egressConfig.EnableIPv6 {
			It("Edit the egressGateway, it will be failed when add different number of ip to `Ippools.IPv4` and `Ippools.IPv6`", Label("G00011"), func() {
				egw.Spec.Ippools.IPv4 = append(egw.Spec.Ippools.IPv4, singleIpv4Pool...)
				GinkgoWriter.Printf("Update EgressGateway: %s\n", egw.Name)
				err := common.UpdateEgressGateway(ctx, cli, egw)
				Expect(err).To(HaveOccurred(), fmt.Sprintf("EgressGateway YAML:\n%s\n", common.GetObjYAML(egw)))
			})
		}

		/*
			When creating an egressgateway, if defaultEIP is not specified, defaultEIP will be randomly retrieved from the ippool
		*/
		It("`DefaultEip` will be assigned randomly from `Ippools` when the filed is empty", Label("G00005"), func() {
			if egressConfig.EnableIPv4 {
				GinkgoWriter.Printf("Check DefaultEip %s if within range %v\n", v4DefaultEip, pool.IPv4)
				included, err := common.CheckIPIncluded(constant.IPv4, v4DefaultEip, pool.IPv4)
				Expect(err).NotTo(HaveOccurred())
				Expect(included).To(BeTrue())
			}
			if egressConfig.EnableIPv6 {
				GinkgoWriter.Printf("Check DefaultEip %s if within range %v\n", v6DefaultEip, pool.IPv6)
				included, err := common.CheckIPIncluded(constant.IPv6, v6DefaultEip, pool.IPv6)
				Expect(err).NotTo(HaveOccurred())
				Expect(included).To(BeTrue())
			}
		})

	})

	Context("Update egressGateway", func() {
		var ctx context.Context
		// --- gateway ---
		var egw *egressv1.EgressGateway
		var v4DefaultEip, v6DefaultEip string
		var expectGatewayStatus *egressv1.EgressGatewayStatus

		// --- policy ---
		var egp *egressv1.EgressPolicy
		// var egcp *egressv1.EgressClusterPolicy
		var expectPolicyStatus *egressv1.EgressPolicyStatus

		// --- pod ---
		var ds *appsv1.DaemonSet

		BeforeEach(func() {
			ctx = context.Background()
			var err error

			// create ds for eip test
			ds, err = common.CreateDaemonSet(ctx, cli, "ds-"+faker.Word(), config.Image, time.Minute/2)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Create DaemonSet: %s\n", ds.Name)

			// create gateway
			egw = createEgressGateway(ctx)
			v4DefaultEip = egw.Spec.Ippools.Ipv4DefaultEIP
			v6DefaultEip = egw.Spec.Ippools.Ipv6DefaultEIP

			// create egressPolicy
			egp, err = common.CreateEgressPolicyNew(ctx, cli, egressConfig, egw.Name, ds.Labels, "")
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Succeeded create EgressPolicy: %s\n", egp.Name)

			// check egressPolicy status
			GinkgoWriter.Println("CheckEgressPolicyStatusSynced")
			expectPolicyStatus = &egressv1.EgressPolicyStatus{
				Eip: egressv1.Eip{
					Ipv4: v4DefaultEip,
					Ipv6: v6DefaultEip,
				},
				Node: node1.Name,
			}
			Expect(common.CheckEgressPolicyStatusSynced(ctx, cli, egp, expectPolicyStatus, time.Second*5)).NotTo(HaveOccurred(),
				fmt.Sprintf("expect: %v, \nbut: %v\n", expectPolicyStatus, egp.Status))

			// check egressGatewayStatus
			GinkgoWriter.Println("CheckEgressGatewayStatusSynced")
			expectGatewayStatus = &egressv1.EgressGatewayStatus{
				NodeList: []egressv1.EgressIPStatus{
					{
						Name: node1.Name,
						Eips: []egressv1.Eips{
							{IPv4: v4DefaultEip, IPv6: v6DefaultEip, Policies: []egressv1.Policy{
								{Name: egp.Name, Namespace: egp.Namespace},
							}},
						},
						Status: string(egressv1.EgressTunnelReady),
					},
				},
			}
			Expect(common.CheckEgressGatewayStatusSynced(ctx, cli, egw, expectGatewayStatus, time.Second*10)).NotTo(HaveOccurred(),
				fmt.Sprintf("expect: %v, \nbut: %v\n", *expectGatewayStatus, egw.Status))

			// todo @bzsuni
			// // create egressClusterPolicy
			// egcp, err = common.CreateEgressClusterPolicy(ctx, cli, egressConfig, egw.Name, ds.Labels)
			// Expect(err).NotTo(HaveOccurred())
			// GinkgoWriter.Printf("Succeeded create egressClusterPolicy: %s\n", egcp.Name)

			// todo @bzsuni
			// // check egressClusterPolicy status
			// GinkgoWriter.Println("CheckEgressClusterPolicyStatusSynced")
			// Expect(common.CheckEgressClusterPolicyStatusSynced(ctx, cli, egcp, expectPolicyStatus, time.Second*5)).NotTo(HaveOccurred(),
			// 	fmt.Sprintf("expect: %v, \nbut: %v\n", *expectPolicyStatus, egcp.Status))

			// todo @bzsuni
			// // check egressGatewayStatus
			// GinkgoWriter.Println("CheckEgressGatewayStatusSynced")
			// expectGatewayStatus.NodeList[0].Eips[0].Policies = append(expectGatewayStatus.NodeList[0].Eips[0].Policies, egressv1.Policy{Name: egcp.Name})

			// todo @bzsuni
			// // todo @bzsuni bug -- check failed, waiting fixed
			// // Expect(common.CheckEgressGatewayStatusSynced(ctx, cli, egw, expectGatewayStatus, time.Second*5)).NotTo(HaveOccurred(),
			// // 	fmt.Sprintf("expect: %v, \nbut: %v\n", *expectGatewayStatus, egw.Status))

			// check eip in pod
			GinkgoWriter.Printf("Check eip in ds: %s after create policy\n", ds.Name)
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, ds, egp.Status.Eip.Ipv4, egp.Status.Eip.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func() {
				// delete ds
				GinkgoWriter.Printf("Delete ds: %s\n", ds.Name)
				Expect(common.DeleteObj(ctx, cli, ds)).NotTo(HaveOccurred())

				// delete egp
				GinkgoWriter.Printf("Delete egp: %s\n", egp.Name)
				Expect(common.DeleteObj(ctx, cli, egp)).NotTo(HaveOccurred())

				// todo @bzsuni
				// GinkgoWriter.Printf("Delete egcp: %s\n", egcp.Name)
				// Expect(common.DeleteObj(ctx, cli, egcp)).NotTo(HaveOccurred())

				// delete egw
				time.Sleep(time.Second)
				GinkgoWriter.Printf("Delete egw: %s\n", egw.Name)
				Expect(common.DeleteEgressGateway(ctx, cli, egw, time.Minute/2)).NotTo(HaveOccurred())
			})
		})

		/*
			Test editing egressGatewaySpec.NodeSelector and check the synchronization status of gateway and policy, and pod egress IP:

			1. In beforeeach, create an egressGateway, specify NodeSelector as node1, create policy, clusterPolicy and daemonset
			2. Update egressGatewaySpec.NodeSelector from node1 to node2, check status of gateway, policy and clusterPolicy, check pod egress IP
			3. Update egressGatewaySpec.NodeSelector from node2 to not matching any node, check status of gateway, policy and clusterPolicy, check pod egress IP
			4. Update egressGatewaySpec.NodeSelector from not matching any node to node2, check status of gateway, policy and clusterPolicy, check pod egress IP
		*/

		// todo @bzsuni waiting for the bug be fixed
		PIt("Update egressGatewaySpec.NodeSelector", Label("G00014"), func() {
			var err error

			By("Change egressGatewaySpec.NodeSelector form node1 to node2")
			GinkgoWriter.Printf("Before update nodeSelector form node1 to node2, egw: %s\n", common.GetObjYAML(egw))
			egw.Spec.NodeSelector.Selector = metav1.SetAsLabelSelector(node2.Labels)
			Expect(cli.Update(ctx, egw)).NotTo(HaveOccurred())

			// check egressGatewayStatus
			GinkgoWriter.Printf("We expect EgressGatewy: %s update successfully\n", egw.Name)
			expectGatewayStatus.NodeList[0].Name = node2.Name
			Expect(common.CheckEgressGatewayStatusSynced(ctx, cli, egw, expectGatewayStatus, time.Second*5)).NotTo(HaveOccurred(),
				fmt.Sprintf("expect: %v, \nbut: %v\n", expectGatewayStatus, egw.Status))

			// check expectPolicyStatus
			expectPolicyStatus.Node = node2.Name
			// // todo @bzsuni
			// GinkgoWriter.Printf("We expect clusterPolicy: %s update successfully\n", egcp.Name)
			// Expect(common.CheckEgressClusterPolicyStatus(f, clusterPolicyName, expectPolicyStatus, time.Second*5)).NotTo(HaveOccurred())

			GinkgoWriter.Printf("We expect policy: %s update successfully\n", egp.Name)
			Expect(common.CheckEgressPolicyStatusSynced(ctx, cli, egp, expectPolicyStatus, time.Second*5)).NotTo(HaveOccurred(),
				fmt.Sprintf("expect: %v, \nbut: %v\n", expectPolicyStatus, egp.Status))

			// check eip in pod
			GinkgoWriter.Printf("Check eip in ds: %s after update egressGateway nodeSelector from node1 to node2\n", ds.Name)
			err = common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, ds, egp.Status.Eip.Ipv4, egp.Status.Eip.Ipv6, true)
			Expect(err).NotTo(HaveOccurred())

			By("Change egressGatewaySpec.NodeSelector to not match any nodes")
			egw.Spec.NodeSelector.Selector = metav1.SetAsLabelSelector(map[string]string{"not-match": ""})
			Expect(cli.Update(ctx, egw)).NotTo(HaveOccurred())

			// check egressGatewayStatus
			emptyGatewayStatus := &egressv1.EgressGatewayStatus{}
			GinkgoWriter.Println("We expect the EgressGatewayStatus is empty")
			Expect(common.CheckEgressGatewayStatusSynced(ctx, cli, egw, emptyGatewayStatus, time.Second*5)).NotTo(HaveOccurred(),
				fmt.Sprintf("expect: %v, \nbut: %v\n", *emptyGatewayStatus, egw.Status))

			// check expectPolicyStatus
			emptyPolicyStatus := &egressv1.EgressPolicyStatus{}
			GinkgoWriter.Printf("We expect policy: %s update successfully\n", egp.Name)
			Expect(common.CheckEgressPolicyStatusSynced(ctx, cli, egp, emptyPolicyStatus, time.Second*5)).NotTo(HaveOccurred(),
				fmt.Sprintf("expect: %v, \nbut: %v\n", emptyPolicyStatus, egp.Status))

			// todo @bzsuni
			// 	GinkgoWriter.Printf("We expect ClusterPolicy: %s update successfully\n", egcp.Name)
			// Expect(common.CheckEgressClusterPolicyStatus(ctx, cli, egcp, emptyPolicyStatus, time.Second*5)).NotTo(HaveOccurred(),
			// fmt.Sprintf("expect: %v, \nbut: %v\n", emptyPolicyStatus, egcp.Status))

			// check eip in pod
			GinkgoWriter.Printf("Check eip in ds: %s after update egressGateway nodeSelector to not match any nodes\n", ds.Name)
			Expect(common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, ds, v4DefaultEip, v6DefaultEip, false)).NotTo(HaveOccurred())

			By("Change egressGatewaySpec.NodeSelector form notMatchedLabel to nodeBLable")
			egw.Spec.NodeSelector.Selector = metav1.SetAsLabelSelector(node2.Labels)
			Expect(cli.Update(ctx, egw)).NotTo(HaveOccurred())

			// check egressGatewayStatus
			GinkgoWriter.Printf("We expect EgressGatewy: %s update successfully\n", egw.Name)
			Expect(common.CheckEgressGatewayStatusSynced(ctx, cli, egw, expectGatewayStatus, time.Second*5)).NotTo(HaveOccurred(),
				fmt.Sprintf("expect: %v, \nbut: %v\n", expectGatewayStatus, egw.Status))

			// check expectPolicyStatus
			GinkgoWriter.Printf("We expect policy: %s update successfully\n", egp.Name)
			Expect(common.CheckEgressPolicyStatusSynced(ctx, cli, egp, expectPolicyStatus, time.Second*5)).NotTo(HaveOccurred(),
				fmt.Sprintf("expect: %v, \nbut: %v\n", expectPolicyStatus, egp.Status))

			// todo @bzsuni
			// 	GinkgoWriter.Printf("We expect clusterPolicy: %s update successfully\n", egcp.Name)
			// Expect(common.CheckEgressClusterPolicyStatus(ctx, cli, egcp, emptyPolicyStatus, time.Second*5)).NotTo(HaveOccurred(),
			// fmt.Sprintf("expect: %v, \nbut: %v\n", emptyPolicyStatus, egcp.Status))

			// check eip in pod
			GinkgoWriter.Printf("Check eip in ds: %s after update egressGateway nodeSelector form not match any nodes to node2\n", ds.Name)
			Expect(common.CheckDaemonSetEgressIP(ctx, cli, config, egressConfig, ds, egp.Status.Eip.Ipv4, egp.Status.Eip.Ipv6, true)).NotTo(HaveOccurred())
		})
	})

	/*
		Test Case: EgressGateway Finalizer Testing

		1. Create an egress gateway and verify that the finalizer is added.
		2. Create a policy referencing the egress gateway created in the previous step.
		3. Delete the egress gateway and verify that it enters the "deleting" state but is not immediately deleted.
		4. Delete the policy, and verify that the egress gateway is subsequently deleted.
	*/
	Context("Delete egressGateway", func() {
		var ctx context.Context
		// gateway
		var egw *egressv1.EgressGateway

		// policy
		var egp *egressv1.EgressPolicy

		// lalbe
		var label map[string]string

		// error
		var err error

		var gatewayFinalizer = "egressgateway.spidernet.io/egressgateway"

		BeforeEach(func() {
			ctx = context.Background()

			label = map[string]string{"test-finalizer": faker.Word()}

			// create gateway
			egw = createEgressGateway(ctx)

			// check finalizer
			Expect(egw.GetFinalizers()).Should(ContainElement(gatewayFinalizer), "failed to check egressgateway finzalizer")

			// create egressPolicy
			egp, err = common.CreateEgressPolicyNew(ctx, cli, egressConfig, egw.Name, label, "")
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Succeeded create EgressPolicy: %s\n", egp.Name)

			DeferCleanup(func() {
				// delete the egp if it exists
				if egp != nil {
					GinkgoWriter.Printf("Delete egp: %s\n", egp.Name)
					Expect(common.DeleteObj(ctx, cli, egp)).NotTo(HaveOccurred())
				}

				// delete the egw if it exists
				if egw != nil {
					GinkgoWriter.Printf("Delete egw: %s\n", egw.Name)
					Expect(common.DeleteEgressGateway(ctx, cli, egw, time.Minute/2)).NotTo(HaveOccurred())
				}
			})
		})

		It("Test egressgateway finalizer", Label("G00020"), func() {
			// delete gateway
			GinkgoWriter.Printf("delete the egw: %s, we expect it to be in deleting status", egw.Name)
			Expect(common.DeleteObj(ctx, cli, egw)).NotTo(HaveOccurred())
			Consistently(ctx, func() error {
				err = cli.Get(ctx, types.NamespacedName{Name: egw.Name}, egw)
				if err != nil {
					return err
				}
				if egw.DeletionTimestamp.Time.IsZero() {
					return fmt.Errorf("not found deletionTimeStamp")
				}
				return nil
			}).WithTimeout(time.Second * 6).WithPolling(time.Second * 2).Should(Succeed())

			// delete egp
			GinkgoWriter.Printf("delete the egp: %s\n", egp.Name)
			Expect(common.DeleteObj(ctx, cli, egp)).NotTo(HaveOccurred())

			// we expect the egw will be delete after a while
			Eventually(ctx, func() error {
				err = cli.Get(ctx, types.NamespacedName{Name: egw.Name}, egw)
				if errors.IsNotFound(err) {
					return nil
				} else {
					return fmt.Errorf("get the egw: %s that is not our expected", egw.Name)
				}
			}).WithTimeout(time.Second * 6).WithPolling(time.Second * 2).Should(Succeed())
		})
	})
})

func createEgressGateway(ctx context.Context) (egw *egressv1.EgressGateway) {
	// create gateway
	GinkgoWriter.Println("Create EgressGateway")
	pool, err := common.GenIPPools(ctx, cli, egressConfig.EnableIPv4, egressConfig.EnableIPv6, 3, 2)
	Expect(err).NotTo(HaveOccurred())
	nodeSelector := egressv1.NodeSelector{Selector: &metav1.LabelSelector{MatchLabels: node1.Labels}}
	egw, err = common.CreateGatewayNew(ctx, cli, "egw-"+uuid.NewString(), pool, nodeSelector)
	Expect(err).NotTo(HaveOccurred())

	// get defaultEip
	_, _, err = common.GetGatewayDefaultIP(ctx, cli, egw, egressConfig)
	Expect(err).NotTo(HaveOccurred())
	return
}
