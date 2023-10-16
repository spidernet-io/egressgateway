// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-faker/faker/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spidernet-io/egressgateway/pkg/constant"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

var _ = Describe("Operate EgressGateway", Label("EgressGateway"), Ordered, func() {
	var labels map[string]string
	var name string
	var egw *egressv1.EgressGateway
	var ctx = context.Background()

	var (
		badDefaultIPv4, badDefaultIPv6 string
		invalidIPv4, invalidIPv6       string
		singleIpv4Pool, singleIpv6Pool []string
		rangeIpv4Pool, rangeIpv6Pool   []string
		cidrIpv4Pool, cidrIpv6Pool     []string
	)
	var labelSelector *metav1.LabelSelector

	BeforeEach(func() {
		egw = new(egressv1.EgressGateway)

		// single Ippools
		singleIpv4Pool, singleIpv6Pool = make([]string, 0), make([]string, 0)
		// range Ippools
		rangeIpv4Pool, rangeIpv6Pool = make([]string, 0), make([]string, 0)
		// cidr Ippools
		cidrIpv4Pool, cidrIpv6Pool = make([]string, 0), make([]string, 0)

		name = tools.GenerateRandomName("eg")
		labels = map[string]string{gateway: name}

		labelSelector = &metav1.LabelSelector{MatchLabels: labels}

		if egressConfig.EnableIPv4 {
			badDefaultIPv4 = "11.10.0.1"
			invalidIPv4 = "invalidIPv4"
			singleIpv4Pool = []string{common.RandomIPV4()}
			rangeIpv4Pool = []string{common.RandomIPPoolV4Range("10", "12")}
			cidrIpv4Pool = []string{common.RandomIPPoolV4Cidr("24")}
		} else {
			fmt.Println("1")
		}
		if egressConfig.EnableIPv6 {
			badDefaultIPv6 = "fdde:10::1"
			invalidIPv6 = "invalidIPv6"
			singleIpv6Pool = []string{common.RandomIPV6()}
			rangeIpv6Pool = []string{common.RandomIPPoolV6Range("a", "c")}
			cidrIpv6Pool = []string{common.RandomIPPoolV6Cidr("120")}
		} else {
			fmt.Println("1")
		}

		GinkgoWriter.Println(singleIpv4Pool, singleIpv6Pool)

		DeferCleanup(func() {
			ctx := context.Background()
			// delete EgressGateway
			if egw != nil {
				err := common.DeleteObj(ctx, cli, egw)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	GinkgoWriter.Println("85", singleIpv4Pool, singleIpv6Pool)

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

	Context("Edit egressGateway", func() {
		var egw *egressv1.EgressGateway
		var v4DefaultEip, v6DefaultEip string

		BeforeEach(func() {
			ctx := context.Background()
			var err error

			GinkgoWriter.Println("Create EgressGateway")
			nodeSelector := egressv1.NodeSelector{Selector: &metav1.LabelSelector{MatchLabels: node1Label}}
			egw, err = common.CreateGatewayNew(ctx, cli, "egw-"+faker.Word(), egressv1.Ippools{IPv4: rangeIpv4Pool, IPv6: rangeIpv6Pool}, nodeSelector)
			Expect(err).NotTo(HaveOccurred())

			// wait `DefaultEip` updated in egressGateway status
			GinkgoWriter.Println("Wait EgressGateway defaultEgressIP update")

			v4DefaultEip, v6DefaultEip, err = common.GetGatewayDefaultIP(ctx, cli, egw, egressConfig)
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func() {
				err = common.DeleteObj(ctx, cli, egw)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		It("`DefaultEip` will be assigned randomly from `Ippools` when the filed is empty", Label("G00005"), func() {
			if egressConfig.EnableIPv4 {
				GinkgoWriter.Printf("Check DefaultEip %s if within range %v\n", v4DefaultEip, rangeIpv4Pool)
				included, err := common.CheckIPIncluded(constant.IPv4, v4DefaultEip, rangeIpv4Pool)
				Expect(err).NotTo(HaveOccurred())
				Expect(included).To(BeTrue())
			}
			if egressConfig.EnableIPv6 {
				GinkgoWriter.Printf("Check DefaultEip %s if within range %v\n", v6DefaultEip, rangeIpv6Pool)
				included, err := common.CheckIPIncluded(constant.IPv6, v6DefaultEip, rangeIpv6Pool)
				Expect(err).NotTo(HaveOccurred())
				Expect(included).To(BeTrue())
			}
		})

		DescribeTable("Test update egressGateway", func(expectedErr bool, update func(egw *egressv1.EgressGateway)) {
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
	})
})
