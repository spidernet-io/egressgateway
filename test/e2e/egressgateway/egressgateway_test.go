// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-faker/faker/v4"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
			} else {
				fmt.Println("1")
			}
			if egressConfig.EnableIPv6 {
				badDefaultIPv6 = "fdde:10::1"
				invalidIPv6 = "invalidIPv6"
				singleIpv6Pool = []string{common.RandomIPV6()}
			} else {
				fmt.Println("1")
			}

			GinkgoWriter.Println(singleIpv4Pool, singleIpv6Pool)

			DeferCleanup(func() {
				// delete EgressGateway
				if egw != nil {
					err := common.DeleteObj(ctx, cli, egw)
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})

		GinkgoWriter.Println("85", singleIpv4Pool, singleIpv6Pool)

		// failed to create egressGateway
		DescribeTable("Failed to create egressGateway", func(setUp func(*egressv1.EgressGateway)) {
			var err error
			egw, err = common.CreateGatewayCustom(ctx, cli, setUp)
			Expect(err).To(HaveOccurred())
		},
			Entry("When `Ippools` is invalid", Label("G00001"), func(egw *egressv1.EgressGateway) {
				egw.Spec.Ippools = egressv1.Ippools{IPv4: []string{invalidIPv4}, IPv6: []string{invalidIPv6}}
			}),
			// TODO @bzsuni
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

		// succeeded to create egressGateway
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
			// TODO @bzsuni
			PEntry("when `Ippools` is a IP CIDR", Label("G00008"), func(egw *egressv1.EgressGateway) {
				egw.Spec.Ippools = egressv1.Ippools{IPv4: cidrIpv4Pool, IPv6: cidrIpv6Pool}
				egw.Spec.NodeSelector = egressv1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector}
			}),
		)
	})

	Context("Create egressGateway with empty ippools", Label("G00020", "G00021"), func() {
		var egw *egressv1.EgressGateway
		var egp *egressv1.EgressPolicy
		var egcp *egressv1.EgressClusterPolicy
		var ctx context.Context
		var err error

		var labelSelector *metav1.LabelSelector

		BeforeEach(func() {
			ctx = context.Background()

			labelSelector = &metav1.LabelSelector{MatchLabels: node1.Labels}

			// create gateway with empty ippools
			nodeSelector := egressv1.NodeSelector{
				Selector: labelSelector,
			}
			egw, err = common.CreateGatewayNew(ctx, cli, "egw-"+faker.Word(), egressv1.Ippools{}, nodeSelector)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Succeeded to create egw:\n%s\n", common.GetObjYAML(egw))

			DeferCleanup(func() {
				// delete egp
				if egp != nil {
					GinkgoWriter.Printf("Delete egp: %s\n", egp.Name)
					Expect(common.DeleteObj(ctx, cli, egp)).NotTo(HaveOccurred())
				}

				// delete egcp
				if egcp != nil {
					GinkgoWriter.Printf("Delete egcp: %s\n", egcp.Name)
					Expect(common.DeleteObj(ctx, cli, egcp)).NotTo(HaveOccurred())
				}

				// delete egw
				if egw != nil {
					GinkgoWriter.Printf("Delete egw: %s\n", egw.Name)
					time.Sleep(time.Second)
					Expect(common.DeleteObj(ctx, cli, egw)).NotTo(HaveOccurred())
				}
			})
		})

		// create egressPolicy
		DescribeTable("creaet policy", func(expect bool, setup func(*egressv1.EgressPolicy)) {
			egp, err = common.CreateEgressPolicyCustom(ctx, cli, setup)
			if expect {
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("egp:\n%s\n", common.GetObjYAML(egp)))
			} else {
				Expect(err).To(HaveOccurred(), fmt.Sprintf("egp:\n%s\n", common.GetObjYAML(egp)))
			}
		},
			Entry("should be failed when spec.egressIP.useNodeIP is false", false, func(egp *egressv1.EgressPolicy) {
				egp.Spec.EgressGatewayName = egw.Name
				egp.Spec.EgressIP.UseNodeIP = false
			}),
			// todo @bzsuni wait fixed
			PEntry("should be succeeded when spec.egressIP.useNodeIP is true", true, func(egp *egressv1.EgressPolicy) {
				egp.Spec.EgressGatewayName = egw.Name
				egp.Spec.EgressIP.UseNodeIP = true
			}),
		)

		// create egressClusterPolicy
		DescribeTable("creaet clusterPolicy", func(expect bool, setup func(*egressv1.EgressClusterPolicy)) {
			egcp, err = common.CreateEgressClusterPolicyCustom(ctx, cli, setup)
			if expect {
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("egcp:\n%s\n", common.GetObjYAML(egcp)))
			} else {
				Expect(err).To(HaveOccurred(), fmt.Sprintf("egcp:\n%s\n", common.GetObjYAML(egcp)))
			}
		},
			// todo @bzsuni wait fixed
			PEntry("should be failed when spec.egressIP.useNodeIP is false", false, func(egcp *egressv1.EgressClusterPolicy) {
				egcp.Spec.EgressGatewayName = egw.Name
				egcp.Spec.EgressIP.UseNodeIP = false
			}),
			Entry("should be succeeded when spec.egressIP.useNodeIP is true", true, func(egcp *egressv1.EgressClusterPolicy) {
				egcp.Spec.EgressGatewayName = egw.Name
				egcp.Spec.EgressIP.UseNodeIP = true
			}),
		)
	})

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
			} else {
				fmt.Println("1")
			}
			if egressConfig.EnableIPv6 {
				invalidIPv6 = "invalidIPv6"
				singleIpv6Pool = []string{common.RandomIPV6()}
			} else {
				fmt.Println("1")
			}

			GinkgoWriter.Println(singleIpv4Pool, singleIpv6Pool)

			// create gateway
			egw = createEgressGateway(ctx)
			pool = egw.Spec.Ippools
			v4DefaultEip = pool.Ipv4DefaultEIP
			v6DefaultEip = pool.Ipv6DefaultEIP

			DeferCleanup(func() {
				// delete EgressGateway
				if egw != nil {
					err := common.DeleteObj(ctx, cli, egw)
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})

		GinkgoWriter.Println("85", singleIpv4Pool, singleIpv6Pool)

		DescribeTable("Edit egressGateway", func(expectedErr bool, update func(egw *egressv1.EgressGateway)) {
			// if not expected, error occurred
			GinkgoWriter.Printf("Update EgressGateway: %s\n", egw.Name)
			update(egw)
			err := common.UpdateEgressGateway(ctx, cli, egw)
			if expectedErr {
				if err == nil {
					raw := common.GetObjYAML(egw)
					GinkgoWriter.Println("EgressGateway YAML:")
					GinkgoWriter.Println(raw)
				}
				Expect(err).To(HaveOccurred())
			} else {
				if err != nil {
					raw := common.GetObjYAML(egw)
					GinkgoWriter.Println("EgressGateway YAML:")
					GinkgoWriter.Println(raw)
				}
				Expect(err).NotTo(HaveOccurred())
			}
		},
			Entry("Failed when add invalid `IP` to `Ippools`", Label("G00009"), true, func(egw *egressv1.EgressGateway) {
				raw := common.GetObjYAML(egw)
				GinkgoWriter.Println("EgressGateway YAML, Update before:")
				GinkgoWriter.Println(raw)
				GinkgoWriter.Println("----------------------------------")

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
			PEntry("Failed when delete `IP` that being used", Label("G00010"), true, func(egw *egressv1.EgressGateway) {
				raw := common.GetObjYAML(egw)
				GinkgoWriter.Println("EgressGateway YAML, Update before:")
				GinkgoWriter.Println(raw)

				if egressConfig.EnableIPv4 {
					egw.Spec.Ippools.IPv4 = tools.RemoveValueFromSlice(egw.Spec.Ippools.IPv4, v4DefaultEip)
				}
				if egressConfig.EnableIPv6 {
					egw.Spec.Ippools.IPv6 = tools.RemoveValueFromSlice(egw.Spec.Ippools.IPv6, v6DefaultEip)
				}
			}),
			PEntry("Failed when add different number of ip to `Ippools.IPv4` and `Ippools.IPv6`",
				Label("G00011"), egressConfig.EnableIPv4 && egressConfig.EnableIPv6, func(egw *egressv1.EgressGateway) {
					if egressConfig.EnableIPv4 && egressConfig.EnableIPv6 {
						egw.Spec.Ippools.IPv4 = append(egw.Spec.Ippools.IPv4, singleIpv4Pool...)
					}
				}),
		)

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
			ds, err = common.CreateDaemonSet(ctx, cli, "ds-"+faker.Word(), config.Image)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Create DaemonSet: %s\n", ds.Name)

			// create gateway
			egw = createEgressGateway(ctx)
			v4DefaultEip = egw.Spec.Ippools.Ipv4DefaultEIP
			v6DefaultEip = egw.Spec.Ippools.Ipv6DefaultEIP

			// create egressPolicy
			egp, err = common.CreateEgressPolicyNew(ctx, cli, egressConfig, egw.Name, ds.Labels)
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
				Expect(common.DeleteObj(ctx, cli, egw)).NotTo(HaveOccurred())
			})
		})

		It("Update egressGatewaySpec.NodeSelector", Label("G00014", "G00015", "G00016"), func() {
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
			// todo @bzsuni
			// 	GinkgoWriter.Printf("We expect clusterPolicy: %s update successfully\n", egcp.Name)
			// 	Expect(common.CheckEgressClusterPolicyStatus(f, clusterPolicyName, expectPolicyStatus, time.Second*5)).NotTo(HaveOccurred())

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
			GinkgoWriter.Println("We expect the EgressGatewayStatus is emtpty")
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

	Context("Delete egressGateway", Label("G00017", "G00018"), func() {
		var err error
		var ctx context.Context
		// --- gateway ---
		var egw *egressv1.EgressGateway

		// --- policy ---
		var egp *egressv1.EgressPolicy
		var egcp *egressv1.EgressClusterPolicy

		// label
		var labels map[string]string

		BeforeEach(func() {
			ctx = context.Background()
			labels = map[string]string{"test-kay": ""}

			egw = new(egressv1.EgressGateway)
			egp = new(egressv1.EgressPolicy)
			egcp = new(egressv1.EgressClusterPolicy)

			// create gateway
			egw = createEgressGateway(ctx)

			DeferCleanup(func() {
				// delete egp
				GinkgoWriter.Printf("Delete egp: %s\n", egp.Name)
				Expect(common.DeleteObj(ctx, cli, egp)).NotTo(HaveOccurred())

				// delete egcp
				GinkgoWriter.Printf("Delete egcp: %s\n", egcp.Name)
				Expect(common.DeleteObj(ctx, cli, egcp)).NotTo(HaveOccurred())

				// delete egw
				time.Sleep(time.Second)
				GinkgoWriter.Printf("Delete egw: %s\n", egw.Name)
				Expect(common.DeleteObj(ctx, cli, egw)).NotTo(HaveOccurred())
			})
		})

		It("Failed delete egressGateway when a egressPolicy using it", func() {
			// create egressPolicy
			egp, err = common.CreateEgressPolicyNew(ctx, cli, egressConfig, egw.Name, labels)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Succeeded create EgressPolicy: %s\n", egp.Name)

			// delete egressGateway we expect its failed
			GinkgoWriter.Printf("Delete egressGateway: %s, we expect its failed\n", egw.Name)
			Expect(common.DeleteObj(ctx, cli, egw)).To(HaveOccurred())

			// delete egressPolicy
			GinkgoWriter.Printf("Delete egressPolicy: %s\n", egp.Name)
			Expect(common.DeleteObj(ctx, cli, egp)).NotTo(HaveOccurred())

			// delete egw
			GinkgoWriter.Printf("Delete egressGateway: %s with no policies using it\n", egw.Name)
			Expect(common.DeleteObj(ctx, cli, egw)).NotTo(HaveOccurred())
		})

		It("Failed delete egressGateway when a egressClusterPolicy using it", func() {
			// create egressClusterPolicy
			egcp, err = common.CreateEgressClusterPolicy(ctx, cli, egressConfig, egw.Name, labels)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("Succeeded create EgressClusterPolicy: %s\n", egcp.Name)

			// delete egressGateway we expect its failed
			GinkgoWriter.Printf("Delete egressGateway: %s, we expect its failed\n", egw.Name)
			Expect(common.DeleteObj(ctx, cli, egw)).To(HaveOccurred())

			// delete egressClusterPolicy
			GinkgoWriter.Printf("Delete egressClusterPolicy: %s\n", egcp.Name)
			Expect(common.DeleteObj(ctx, cli, egcp)).NotTo(HaveOccurred())

			// delete egw
			GinkgoWriter.Printf("Delete egressGateway: %s with no policies using it\n", egw.Name)
			Expect(common.DeleteObj(ctx, cli, egw)).NotTo(HaveOccurred())
		})
	})
})

func createEgressGateway(ctx context.Context) (egw *egressv1.EgressGateway) {
	// create gateway
	GinkgoWriter.Println("Create EgressGateway")
	pool, err := common.GenIPPools(ctx, cli, egressConfig, 3, 1)
	Expect(err).NotTo(HaveOccurred())
	nodeSelector := egressv1.NodeSelector{Selector: &metav1.LabelSelector{MatchLabels: node1.Labels}}
	egw, err = common.CreateGatewayNew(ctx, cli, "egw-"+faker.Word(), pool, nodeSelector)
	Expect(err).NotTo(HaveOccurred())

	// get defaultEip
	_, _, err = common.GetGatewayDefaultIP(ctx, cli, egw, egressConfig)
	Expect(err).NotTo(HaveOccurred())
	return
}
