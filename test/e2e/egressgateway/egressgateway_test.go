// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("Egressgateway", func() {
	var gatewayFields egressGatewayFields
	var egressGatewayObj *egressv1.EgressGateway
	var labels, anotherLabels map[string]string
	var name string
	var allNodes, notGatewayNodes, gatewayNodes []string

	BeforeEach(func() {
		labels = map[string]string{"egress": "true"}
		anotherLabels = map[string]string{"egress": "true1"}
		name = common.EGRESSAGEWAY_NAME
		allNodes = []string{}
		notGatewayNodes = []string{}
		gatewayNodes = []string{}

		gatewayFields = egressGatewayFields{
			name:        name,
			matchLabels: labels,

			labelMatched: false,
			ok:           false,
		}

		egressGatewayObj = &egressv1.EgressGateway{}

		DeferCleanup(func() {
			// delete egressgateway if its exists
			Expect(common.DeleteEgressGatewayIfExists(f, gatewayFields.name, time.Second*10)).NotTo(HaveOccurred())

			// delete nodes labels
			Expect(common.UnLabelNodes(f, allNodes, labels)).NotTo(HaveOccurred())
		})
	})

	DescribeTable("create egressgateway", Serial, Label("G00001", "G00002", "G00003"), func(getParams func() *egressGatewayFields) {
		// get params
		p := getParams()
		yaml := common.GenerateEgressGatewayYaml(p.name, p.matchLabels)

		allNodes, err = common.GetAllNodes(f)
		GinkgoWriter.Printf("allNodes: %v\n", allNodes)

		if p.labelMatched {
			gatewayNodes, notGatewayNodes = labelNodes(allNodes, labels, anotherLabels)
		}

		if p.ok {
			By("G00001, create egressgateway")
			GinkgoWriter.Printf("create egressgateway %s \n", p.name)
			Expect(common.CreateEgressGateway(f, yaml)).NotTo(HaveOccurred())
			egressGatewayObj, err = common.WaitEgressGatewayUpdatedStatus(f, p.name, gatewayNodes, time.Minute)
			Expect(err).NotTo(HaveOccurred())

			GinkgoWriter.Printf("succeeded to create egressgateway: %v\n", egressGatewayObj.Name)

			if p.labelMatched == false {
				// have no matched nodes, we expect the number of gatewayNodes is zero
				Expect(gatewayNodes).To(BeEmpty())
				Expect(egressGatewayObj.Status.NodeList).To(BeEmpty())

				// label node
				GinkgoWriter.Println("label node...")
				gatewayNodes, notGatewayNodes = labelNodes(allNodes, labels, anotherLabels)

				// wait egressgateway updated
				egressGatewayObj, err = common.WaitEgressGatewayUpdatedStatus(f, p.name, gatewayNodes, time.Minute)
				Expect(err).NotTo(HaveOccurred())

				// check after labeled nodes
				GinkgoWriter.Println("check after labeled node...")
				check(egressGatewayObj, gatewayNodes)

			} else {
				check(egressGatewayObj, gatewayNodes)

				// G00002: change egressgateway matchLabels
				By("G00002, edit egressgateway")
				GinkgoWriter.Printf("change egressgateway %s matchLabels\n", p.name)
				Expect(common.EditEgressGatewayMatchLabels(f, egressGatewayObj, anotherLabels)).NotTo(HaveOccurred())
				egressGatewayObj, err = common.WaitEgressGatewayUpdatedMatchLabels(f, p.name, anotherLabels, time.Second*10)
				Expect(err).NotTo(HaveOccurred())
				Expect(egressGatewayObj).NotTo(BeNil())
				GinkgoWriter.Printf("changed egressgateway: %v\n", egressGatewayObj)

				gatewayNodes, err = common.GetNodesByMatchLabels(f, anotherLabels)
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Printf("gatewayNodes: %v\n", gatewayNodes)

				notGatewayNodes, err = common.GetUnmatchedNodes(f, gatewayNodes)
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Printf("notGatewayNodes: %v\n", notGatewayNodes)

				// wait egressgateway updated by given timeout
				egressGatewayObj, err = common.WaitEgressGatewayUpdatedStatus(f, p.name, gatewayNodes, time.Minute)
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Printf("egressgatewayObj: %v\n", egressGatewayObj)

				// check
				check(egressGatewayObj, gatewayNodes)
			}

			// G00003: delete egressgateway until finish
			Expect(common.DeleteEgressGatewayUntilFinish(f, egressGatewayObj, time.Second*20)).NotTo(HaveOccurred())

			// check iptables chain after egressgateway deleted
			GinkgoWriter.Println("check iptables chain after delete egressgateway...")
			Expect(common.CheckEgressGatewayChain(allNodes, time.Second*10)).To(BeFalse())
		} else {
			Expect(common.CreateEgressGateway(f, yaml)).To(HaveOccurred())
		}
	},
		PEntry("failed to create egressGateway with name not 'default'", func() *egressGatewayFields {
			gatewayFields.name = "badname"
			return &gatewayFields
		}),
		PEntry("succeeded to create egressGateway with not matched labelSelector", func() *egressGatewayFields {
			gatewayFields.ok = true
			return &gatewayFields
		}),
		PEntry("succeeded to create egressGateway with matched labelSelector", func() *egressGatewayFields {
			gatewayFields.ok = true
			gatewayFields.labelMatched = true
			return &gatewayFields
		}),
	)
})

// some egressgateway fields and assertions used to verify
type egressGatewayFields struct {
	name        string
	matchLabels map[string]string

	// expect assertion result
	ok, labelMatched bool
}

func labelNodes(allNodes []string, labels, anotherLabels map[string]string) (gatewayNodes, notGatewayNodes []string) {
	// label nodes[0]
	Expect(err).NotTo(HaveOccurred())
	gatewayNodes = []string{allNodes[0]}
	anotherNodes := []string{allNodes[1]}
	GinkgoWriter.Printf("gatewayNodes: %v\n", gatewayNodes)
	Expect(common.LabelNodes(f, gatewayNodes, labels)).NotTo(HaveOccurred())
	Expect(common.LabelNodes(f, anotherNodes, anotherLabels)).NotTo(HaveOccurred())

	notGatewayNodes, err = common.GetUnmatchedNodes(f, gatewayNodes)
	GinkgoWriter.Printf("notGatewayNodes: %v\n", notGatewayNodes)
	Expect(err).NotTo(HaveOccurred())
	return
}

func check(egressGateway *egressv1.EgressGateway, gatewayNodes []string) {
	// check egressgateway status.nodelist
	GinkgoWriter.Printf("egressGatewayObj.Status.NodeList: %v\n", egressGateway.Status.NodeList)
	common.CheckEgressGatewayNodeList(f, egressGateway, gatewayNodes)
}
