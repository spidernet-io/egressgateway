// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway_test

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spidernet-io/egressgateway/pkg/constant"
	egressv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

var ErrNotNeed = errors.New("not need this case")
var _ = Describe("Operate egressGateway", Label("egressGateway"), func() {
	var labels map[string]string
	var name string
	var (
		badDefaultIPv4, badDefaultIPv6 string
		invalidIPv4, invalidIPv6       string
		singleIpv4Pool, singleIpv6Pool []string
		rangeIpv4Pool, rangeIpv6Pool   []string
		cidrIpv4Pool, cidrIpv6Pool     []string
	)
	var labelSelector *metav1.LabelSelector

	//var notGatewayNodes, gatewayNodes []string

	BeforeEach(func() {
		// single Ippools
		singleIpv4Pool, singleIpv6Pool = make([]string, 0), make([]string, 0)
		// range Ippools
		rangeIpv4Pool, rangeIpv6Pool = make([]string, 0), make([]string, 0)
		// cidr Ippools
		cidrIpv4Pool, cidrIpv6Pool = make([]string, 0), make([]string, 0)

		name = tools.GenerateRandomName("eg")
		labels = map[string]string{gateway: name}

		labelSelector = &metav1.LabelSelector{MatchLabels: labels}

		if enableV4 {
			badDefaultIPv4 = "11.10.0.1"
			invalidIPv4 = "invalidIPv4"
			singleIpv4Pool = []string{common.RandomIPV4()}
			rangeIpv4Pool = []string{common.RandomIPPoolV4Range("10", "12")}
			cidrIpv4Pool = []string{common.RandomIPPoolV4Cidr("24")}
		}
		if enableV6 {
			badDefaultIPv6 = "fdde:10::1"
			invalidIPv6 = "invalidIPv6"
			singleIpv6Pool = []string{common.RandomIPV6()}
			rangeIpv6Pool = []string{common.RandomIPPoolV6Range("a", "c")}
			cidrIpv6Pool = []string{common.RandomIPPoolV6Cidr("120")}
		}

		DeferCleanup(func() {
			// delete egressgateway if its exists
			Expect(common.DeleteEgressGatewayIfExists(f, name, time.Second*10)).NotTo(HaveOccurred())
		})
	})

	DescribeTable("Failed to create egressGateway", func(checkCreateEG func() error) {
		Expect(checkCreateEG()).To(HaveOccurred())
	},
		Entry("When `Ippools` is invalid", Label("G00001"), func() error {
			return common.CreateEgressGateway(f, common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: []string{invalidIPv4}, IPv6: []string{invalidIPv6}}, egressv1beta1.NodeSelector{}))
		}),
		// todo bzsuni
		PEntry("When `NodeSelector` is empty", Label("G00002"), func() error {
			return common.CreateEgressGateway(f, common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: singleIpv4Pool, IPv6: singleIpv6Pool}, egressv1beta1.NodeSelector{}))
		}),
		Entry("When `defaultEIP` is not in `Ippools`", Label("G00003"), func() error {
			return common.CreateEgressGateway(f,
				common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: singleIpv6Pool, IPv6: singleIpv6Pool, Ipv4DefaultEIP: badDefaultIPv4, Ipv6DefaultEIP: badDefaultIPv6},
					egressv1beta1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector}))
		}),
		Entry("When the number of `Ippools.IPv4` is not same with `Ippools.IPv6` in dual cluster", Label("G00004"), func() error {
			if enableV4 && enableV6 {
				return common.CreateEgressGateway(f,
					common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: singleIpv4Pool, IPv6: []string{}},
						egressv1beta1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector}))
			}
			return ErrNotNeed
		}),
	)

	DescribeTable("Succeeded to create egressGateway", func(createEG func() error) {
		Expect(createEG()).NotTo(HaveOccurred())
	},
		Entry("when `Ippools` is a single IP", Label("G00006"), func() error {
			GinkgoWriter.Printf("singleIpv4Pool: %v, singleIpv6Pool: %v\n", singleIpv4Pool, singleIpv6Pool)
			return common.CreateEgressGateway(f,
				common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: singleIpv4Pool, IPv6: singleIpv6Pool},
					egressv1beta1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector}))
		}),
		Entry("when `Ippools` is a IP range like `a-b`", Label("G00007"), func() error {
			GinkgoWriter.Printf("rangeIpv4Pool: %v, rangeIpv6Pool: %v\n", rangeIpv4Pool, rangeIpv6Pool)
			return common.CreateEgressGateway(f,
				common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: rangeIpv4Pool, IPv6: rangeIpv6Pool},
					egressv1beta1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector}))
		}),
		// todo bzsuni
		PEntry("when `Ippools` is a IP cidr", Label("G00008"), func() error {
			GinkgoWriter.Printf("cidrIpv4Pool: %v, cidrIpv6Pool: %v\n", cidrIpv4Pool, cidrIpv6Pool)
			return common.CreateEgressGateway(f,
				common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: cidrIpv4Pool, IPv6: cidrIpv6Pool},
					egressv1beta1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector}))
		}),
	)

	Context("Edit egressGateway", func() {
		var eg *egressv1beta1.EgressGateway
		var v4DefaultEip, v6DefaultEip string
		var nodeA, nodeB *v1.Node
		var nodeAName, nodeBName string
		var nodeALabel, nodeBLabel *metav1.LabelSelector

		BeforeEach(func() {
			// node
			nodeA = nodeObjs[0]
			nodeB = nodeObjs[1]
			nodeAName = nodeA.Name
			nodeBName = nodeB.Name
			nodeALabel = &metav1.LabelSelector{MatchLabels: nodeA.Labels}
			nodeBLabel = &metav1.LabelSelector{MatchLabels: nodeB.Labels}
			GinkgoWriter.Printf("nodeA: %s, labels: %s\n", nodeAName, common.YamlMarshal(nodeALabel))
			GinkgoWriter.Printf("nodeB: %s, labels: %s\n", nodeBName, common.YamlMarshal(nodeBLabel))

			// generate egressGateway  yaml
			GinkgoWriter.Println("GenerateEgressGatewayYaml")
			eg = common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: rangeIpv4Pool, IPv6: rangeIpv6Pool}, egressv1beta1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: nodeALabel})

			// create egressGateway
			GinkgoWriter.Println("CreateEgressGateway")
			Expect(common.CreateEgressGateway(f, eg)).NotTo(HaveOccurred())

			// wait `DefaultEip` updated in egressGateway status
			GinkgoWriter.Println("WaitEgressGatewayDefaultEIPUpdated")
			v4DefaultEip, v6DefaultEip, err = common.WaitEgressGatewayDefaultEIPUpdated(f, name, enableV4, enableV6, time.Second*10)
			Expect(err).NotTo(HaveOccurred())

			DeferCleanup(func() {

			})
		})

		It("`DefaultEip` will be assigned randomly from `Ippools` when the filed is empty", Label("G00005"), func() {
			if enableV4 {
				GinkgoWriter.Printf("Check DefaultEip %s if within range %v\n", v4DefaultEip, rangeIpv4Pool)
				included, err := common.CheckIPIncluded(constant.IPv4, v4DefaultEip, rangeIpv4Pool)
				Expect(err).NotTo(HaveOccurred())
				Expect(included).To(BeTrue())
			}
			if enableV6 {
				GinkgoWriter.Printf("Check DefaultEip %s if within range %v\n", v6DefaultEip, rangeIpv6Pool)
				included, err := common.CheckIPIncluded(constant.IPv6, v6DefaultEip, rangeIpv6Pool)
				Expect(err).NotTo(HaveOccurred())
				Expect(included).To(BeTrue())
			}
		})

		DescribeTable("Test update egressGateway", func(expected bool, updateEG func() error) {
			// if not expected, error occurred
			if !expected {
				Expect(updateEG()).To(HaveOccurred())
			} else {
				// if expected, error not occurred
				Expect(updateEG()).NotTo(HaveOccurred())
			}
		},
			Entry("Failed when add invalid `IP` to `Ippools`", Label("G00009"), false, func() error {
				if enableV4 {
					eg.Spec.Ippools.IPv4 = append(eg.Spec.Ippools.IPv4, invalidIPv4)
				}
				if enableV6 {
					eg.Spec.Ippools.IPv6 = append(eg.Spec.Ippools.IPv6, invalidIPv6)
				}
				GinkgoWriter.Printf("UpdateEgressGateway: %s\n", eg.Name)
				return common.UpdateEgressGateway(f, eg, time.Second*10)
			}),
			Entry("Succeeded when add valid `IP` to `Ippools`", Label("G00012", "G00013"), true, func() error {
				if enableV4 {
					eg.Spec.Ippools.IPv4 = append(eg.Spec.Ippools.IPv4, singleIpv4Pool...)
				}
				if enableV6 {
					eg.Spec.Ippools.IPv6 = append(eg.Spec.Ippools.IPv6, singleIpv6Pool...)
				}
				GinkgoWriter.Printf("UpdateEgressGateway: %s\n", eg.Name)
				return common.UpdateEgressGateway(f, eg, time.Second*10)
			}),
			Entry("Failed when delete `IP` that being used", Label("G00010"), false, func() error {
				if enableV4 {
					eg.Spec.Ippools.IPv4 = tools.RemoveValueFromSlice(eg.Spec.Ippools.IPv4, v4DefaultEip)
				}
				if enableV6 {
					eg.Spec.Ippools.IPv6 = tools.RemoveValueFromSlice(eg.Spec.Ippools.IPv6, v6DefaultEip)
				}
				GinkgoWriter.Printf("UpdateEgressGateway: %s\n", eg.Name)
				return common.UpdateEgressGateway(f, eg, time.Second*10)
			}),
			Entry("Failed when add different number of ip to `Ippools.IPv4` and `Ippools.IPv6`", Label("G00011"), false, func() error {
				if enableV4 && enableV6 {
					eg.Spec.Ippools.IPv4 = append(eg.Spec.Ippools.IPv4, singleIpv4Pool...)
					GinkgoWriter.Printf("UpdateEgressGateway: %s\n", eg.Name)
					return common.UpdateEgressGateway(f, eg, time.Second*10)
				}
				return ErrNotNeed
			}),
		)
	})

})
