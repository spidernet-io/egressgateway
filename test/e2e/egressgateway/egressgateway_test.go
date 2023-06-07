// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway_test

import (
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	egressv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("egressGateway", Label("egressGateway"), func() {
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
		//eg = new(egressv1beta1.EgressGateway)
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

			// delete nodes labels1
			//Expect(common.UnLabelNodes(f, allNodes, labels1)).NotTo(HaveOccurred())
		})
	})

	DescribeTable("Failed to create egressGateway", func(checkCreateEG func() (error, bool)) {
		err, need := checkCreateEG()
		if need {
			Expect(err).To(HaveOccurred())
		}

	},
		Entry("When `Ippools` is invalid", Label("G00001"), func() (error, bool) {
			return common.CreateEgressGateway(f, common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: []string{invalidIPv4}, IPv6: []string{invalidIPv6}}, egressv1beta1.NodeSelector{})), true
		}),
		// todo bzsuni
		PEntry("When `NodeSelector` is empty", Label("G00002"), func() (error, bool) {
			return common.CreateEgressGateway(f, common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: singleIpv4Pool, IPv6: singleIpv6Pool}, egressv1beta1.NodeSelector{})), true
		}),
		Entry("When `defaultEIP` is not in `Ippools`", Label("G00003"), func() (error, bool) {
			return common.CreateEgressGateway(f,
				common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: singleIpv6Pool, IPv6: singleIpv6Pool, Ipv4DefaultEIP: badDefaultIPv4, Ipv6DefaultEIP: badDefaultIPv6},
					egressv1beta1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector})), true
		}),
		Entry("When the number of `Ippools.IPv4` is not same with `Ippools.IPv6` in dual cluster", Label("G00004"), func() (error, bool) {
			if enableV4 && enableV6 {
				return common.CreateEgressGateway(f,
					common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{IPv4: singleIpv4Pool, IPv6: []string{}},
						egressv1beta1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector})), true
			}
			return nil, false
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

	Context("edit egressGateway", func() {
		//var eg *egressv1beta1.EgressGateway
		//
		//BeforeEach(func() {
		//	// generate egressGateway  yaml
		//	common.GenerateEgressGatewayYaml(name,egressv1beta1.Ippools{IPv4: })
		//	// create egressGateway
		//	common.CreateEgressGateway(f)
		//
		//})
		//It("hello", Label("a"), func() {
		//	GinkgoWriter.Printf("a=%v\n", a)
		//})
	})

	//DescribeTable("create egressGateway", func(createEG func() error) {
	//	Expect(createEG()).NotTo(HaveOccurred())
	//},
	//	Entry("single ip in ippools", func() error {
	//		return common.CreateEgressGateway(f, common.GenerateEgressGatewayYaml(name, egressv1beta1.Ippools{Policy: common.RANDOM, IPv4: ipv4Pool, IPv6: ipv6Pool}, egressv1beta1.NodeSelector{Policy: common.AVERAGE_SELECTION, Selector: labelSelector}))
	//	}),
	//)

	//PDescribeTable("create egressgateway", Serial, func(getParams func() *egressGatewayFields) {
	//	// get params
	//	p := getParams()
	//	yaml := common.GenerateEgressGatewayYaml(p.name, p.matchLabels)
	//
	//	if p.labelMatched {
	//		gatewayNodes, notGatewayNodes = labelNodes(allNodes, labels1, labels2)
	//	}
	//
	//	if p.ok {
	//		By("G00001, create egressgateway")
	//		GinkgoWriter.Printf("create egressgateway %s \n", p.name)
	//		Expect(common.CreateEgressGateway(f, yaml)).NotTo(HaveOccurred())
	//		egressGatewayObj, err = common.WaitEgressGatewayUpdatedStatus(f, p.name, gatewayNodes, time.Minute)
	//		Expect(err).NotTo(HaveOccurred())
	//
	//		GinkgoWriter.Printf("succeeded to create egressgateway: %v\n", egressGatewayObj.Name)
	//
	//		if p.labelMatched == false {
	//			// have no matched nodes, we expect the number of gatewayNodes is zero
	//			Expect(gatewayNodes).To(BeEmpty())
	//			Expect(egressGatewayObj.Status.NodeList).To(BeEmpty())
	//
	//			// label node, check if the egressgateway cr upgraded succeeded
	//			GinkgoWriter.Println("label node...")
	//			gatewayNodes, notGatewayNodes = labelNodes(allNodes, labels1, labels2)
	//
	//			// wait egressgateway updated
	//			egressGatewayObj, err = common.WaitEgressGatewayUpdatedStatus(f, p.name, gatewayNodes, time.Minute)
	//			Expect(err).NotTo(HaveOccurred())
	//
	//			// check after labeled nodes
	//			GinkgoWriter.Println("check after labeled node...")
	//			check(egressGatewayObj, gatewayNodes)
	//
	//		} else {
	//			check(egressGatewayObj, gatewayNodes)
	//
	//			// G00002: change egressgateway matchLabels, check if status of the egressgateway cr been upgraded succeeded
	//			By("G00002, edit egressgateway")
	//			GinkgoWriter.Printf("change egressgateway %s matchLabels\n", p.name)
	//			Expect(common.EditEgressGatewayMatchLabels(f, egressGatewayObj, labels2)).NotTo(HaveOccurred())
	//			egressGatewayObj, err = common.WaitEgressGatewayUpdatedMatchLabels(f, p.name, labels2, time.Second*10)
	//			Expect(err).NotTo(HaveOccurred())
	//			Expect(egressGatewayObj).NotTo(BeNil())
	//			GinkgoWriter.Printf("changed egressgateway: %v\n", egressGatewayObj)
	//
	//			gatewayNodes, err = common.GetNodesByMatchLabels(f, labels2)
	//			Expect(err).NotTo(HaveOccurred())
	//			GinkgoWriter.Printf("gatewayNodes: %v\n", gatewayNodes)
	//
	//			notGatewayNodes, err = common.GetUnmatchedNodes(f, gatewayNodes)
	//			Expect(err).NotTo(HaveOccurred())
	//			GinkgoWriter.Printf("notGatewayNodes: %v\n", notGatewayNodes)
	//
	//			// wait egressgateway updated by given timeout
	//			egressGatewayObj, err = common.WaitEgressGatewayUpdatedStatus(f, p.name, gatewayNodes, time.Minute)
	//			Expect(err).NotTo(HaveOccurred())
	//			GinkgoWriter.Printf("egressgatewayObj: %v\n", egressGatewayObj)
	//
	//			// check
	//			check(egressGatewayObj, gatewayNodes)
	//		}
	//
	//		// G00003: delete egressgateway until finish
	//		Expect(common.DeleteEgressGatewayUntilFinish(f, egressGatewayObj, time.Second*20)).NotTo(HaveOccurred())
	//
	//	} else {
	//		Expect(common.CreateEgressGateway(f, yaml)).To(HaveOccurred())
	//	}
	//},
	//	Entry("failed to create egressGateway with name not 'default'", func() *egressGatewayFields {
	//		gatewayFields.name = "badname"
	//		return &gatewayFields
	//	}),
	//	Entry("succeeded to create egressGateway with not matched labelSelector", func() *egressGatewayFields {
	//		gatewayFields.ok = true
	//		return &gatewayFields
	//	}),
	//	Entry("succeeded to create egressGateway with matched labelSelector", func() *egressGatewayFields {
	//		gatewayFields.ok = true
	//		gatewayFields.labelMatched = true
	//		return &gatewayFields
	//	}),
	//)
})

// some egressgateway fields and assertions used to verify
type egressGatewayFields struct {
	name        string
	matchLabels map[string]string

	// expect assertion result
	ok, labelMatched bool
}

func labelNodes(allNodes []string, labels1, labels2 map[string]string) (gatewayNodes, notGatewayNodes []string) {
	// label nodes[0]
	Expect(err).NotTo(HaveOccurred())
	gatewayNodes = []string{allNodes[0]}
	anotherNodes := []string{allNodes[1]}
	GinkgoWriter.Printf("gatewayNodes: %v\n", gatewayNodes)
	Expect(common.LabelNodes(f, gatewayNodes, labels1)).NotTo(HaveOccurred())
	Expect(common.LabelNodes(f, anotherNodes, labels2)).NotTo(HaveOccurred())

	notGatewayNodes, err = common.GetUnmatchedNodes(f, gatewayNodes)
	GinkgoWriter.Printf("notGatewayNodes: %v\n", notGatewayNodes)
	Expect(err).NotTo(HaveOccurred())
	return
}

//func check(egressGateway *egressv1beta1.EgressGateway, gatewayNodes []string) {
//	// check egressgateway status.nodelist
//	GinkgoWriter.Printf("egressGatewayObj.Status.NodeList: %v\n", egressGateway.Status.NodeList)
//	common.CheckEgressGatewayNodeList(f, egressGateway, gatewayNodes)
//}
