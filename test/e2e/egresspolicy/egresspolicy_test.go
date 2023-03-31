// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egresspolicy_test

import (
	"fmt"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("Egresspolicy", func() {
	Context("test egressGatewayPolicy", Label("P00001"), func() {
		var (
			egressGatewayName      string
			egressPolicyName       string
			podALabels, podBLabels map[string]string
			deployA, deployB       string
			dst, notDst            []string
			serverIP               string
			eIPs                   []string
			count                  uint64
		)

		BeforeEach(func() {
			atomic.AddUint64(&count, 1)

			egressGatewayName = common.EGRESSAGEWAY_NAME
			egressPolicyName = fmt.Sprintf("test-egresspolicy-%d", count)

			if testV4 {
				serverIpv4b, err := tools.GetContainerIPV4(common.NettoolsServer[common.NETTOOLS_SERVER], time.Second*10)
				Expect(err).NotTo(HaveOccurred())
				serverIP = string(serverIpv4b)
				GinkgoWriter.Printf("serverIP: %v\n", serverIP)
				Expect(serverIP).NotTo(BeEmpty())

				dst = []string{serverIP + "/8"}
				notDst = []string{"1.0.0.0/32"}
				GinkgoWriter.Printf("dst: %v\n", dst)
				GinkgoWriter.Printf("notDst: %v\n", notDst)
			}

			if testV6 {
				serverIpv6b, err := tools.GetContainerIPV6(common.NettoolsServer[common.NETTOOLS_SERVER], time.Second*10)
				Expect(err).NotTo(HaveOccurred())
				serverIP = string(serverIpv6b)
				GinkgoWriter.Printf("serverIP: %v\n", serverIP)
				Expect(serverIP).NotTo(BeEmpty())

				dst = []string{serverIP + "/8"}
				notDst = []string{"ffff::/128"}
				GinkgoWriter.Printf("dst: %v\n", dst)
				GinkgoWriter.Printf("notDst: %v\n", notDst)
				serverIP = "[" + serverIP + "]"
				GinkgoWriter.Printf("serverIP: %v\n", serverIP)
			}

			deployA = fmt.Sprintf("test-poda-%d", count)
			deployB = fmt.Sprintf("test-podb-%d", count)
			podALabels = map[string]string{"app": deployA}
			podBLabels = map[string]string{"app": deployB}

			DeferCleanup(func() {
				// delete egressgateway if its exists
				GinkgoWriter.Println("delete egressgateway if its exists")
				Expect(common.DeleteEgressGatewayIfExists(frame, egressGatewayName, time.Second*10)).NotTo(HaveOccurred())

				// delete egresspolicy if its exists
				GinkgoWriter.Println("delete egresspolicy if its exists")
				Expect(common.DeleteEgressPolicyIfExists(frame, egressPolicyName, time.Second*10)).NotTo(HaveOccurred())

				// delete pod
				GinkgoWriter.Println("delete test pod if its exists")
				Expect(common.DeleteDeployIfExists(frame, deployA, common.POD_NAMESPACE, time.Minute)).NotTo(HaveOccurred())
				Expect(common.DeleteDeployIfExists(frame, deployB, common.POD_NAMESPACE, time.Minute)).NotTo(HaveOccurred())
			})
		})

		It("operate egressgatewaypolicy", Label("P00001", "P00002", "P00003"), func() {
			// get node egressgateway-worker labels
			GinkgoWriter.Printf("get node: %s labels\n", worker)
			workerObj, err := frame.GetNode(worker)
			Expect(err).NotTo(HaveOccurred())
			nodeLabels := workerObj.Labels
			GinkgoWriter.Printf("node %s labels: %v\n", worker, nodeLabels)

			// create egressgateway choice the node egressgateway-worker
			GinkgoWriter.Printf("create egressgateway, choice the node: %s\n", worker)
			gatewayYaml := common.GenerateEgressGatewayYaml(egressGatewayName, nodeLabels)
			err = common.CreateEgressGateway(frame, gatewayYaml)
			Expect(err).NotTo(HaveOccurred())

			// wait egressgateway updated
			GinkgoWriter.Printf("wait egressgateway: %s updated succeeded\n", egressGatewayName)
			gateway, err := common.WaitEgressGatewayUpdatedStatus(frame, egressGatewayName, []string{worker}, time.Second*10)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("egressgateway: %v\n", gateway)

			// check egressgateway status about nodeList
			GinkgoWriter.Printf("check %s status about nodeList\n", egressGatewayName)
			common.CheckEgressGatewayNodeList(frame, gateway, []string{worker})

			// get eIPs
			if testV4 {
				GinkgoWriter.Println("get eIPs v4")
				eIPs, err = common.GetEgressGatewayIPsV4(frame, egressGatewayName)
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Printf("eIPs: %v\n", eIPs)
			}
			if testV6 {
				GinkgoWriter.Println("get eIPs v6")
				eIPs, err = common.GetEgressGatewayIPsV6(frame, egressGatewayName)
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Printf("eIPs: %v\n", eIPs)
			}

			By("P00001, create egressgatewaypolicy")
			// create test client deployA select node control-plane
			GinkgoWriter.Printf("create test client poda in node: %s\n", controlPlane)
			deployAOjb := common.CreateClientPod(frame, deployA, controlPlane, serverIP, int32(1), podALabels, time.Minute)

			// we expect export IP is not eip
			for _, eip := range eIPs {
				GinkgoWriter.Println("we expect export ip is not eip")
				common.CheckEIP(frame, deployAOjb, eip, false, time.Second*30, time.Minute)
			}

			// create egressgatewaypolicy
			GinkgoWriter.Printf("create egressgatewaypolicy: %s\n", egressPolicyName)
			policyYaml := common.GenerateEgressPolicyYaml(egressPolicyName, podALabels, dst)

			err = common.CreateEgressPolicy(frame, policyYaml)
			Expect(err).NotTo(HaveOccurred())

			egressPolicy := &egressv1.EgressGatewayPolicy{}
			Expect(common.GetEgressPolicy(frame, egressPolicyName, egressPolicy)).NotTo(HaveOccurred())
			GinkgoWriter.Printf("egressPolicy: %v\n", egressPolicy)

			// reboot deployA, we expect export ip is eip
			restartPodAndCheckEIP(deployAOjb, eIPs, true)

			// test edit egressgatewaypolicy
			By("P000002, edit egressgatewaypolicy")
			// create test client deployB select node control-plane
			GinkgoWriter.Printf("create test client podb in node: %s\n", controlPlane)
			deployBOjb := common.CreateClientPod(frame, deployB, controlPlane, serverIP, int32(1), podBLabels, time.Minute)

			// we expect export IP is not eip
			for _, eip := range eIPs {
				GinkgoWriter.Println("we expect export ip is not eip")
				common.CheckEIP(frame, deployBOjb, eip, false, time.Second*30, time.Minute)
			}

			// edit egressgateway labelselector podA to podB
			GinkgoWriter.Println("change egressgatewaypolicy podselector")
			// update egressgatewaypolicy
			Expect(common.EditEgressPolicy(frame, egressPolicy, podBLabels, dst)).NotTo(HaveOccurred())

			GinkgoWriter.Println("restart podA, we expect podA export ip is not eip")
			restartPodAndCheckEIP(deployAOjb, eIPs, false)
			GinkgoWriter.Println("restart podB, we expect podB export ip is eip")
			restartPodAndCheckEIP(deployBOjb, eIPs, true)

			// edit egressgateway dst to not matched any pod
			GinkgoWriter.Printf("change egressgatewaypolicy dst to %v\n", notDst)
			// update egressgatewaypolicy
			Expect(common.EditEgressPolicy(frame, egressPolicy, podBLabels, notDst)).NotTo(HaveOccurred())

			GinkgoWriter.Println("restart podB, we expect podB export ip is not eip")
			restartPodAndCheckEIP(deployBOjb, eIPs, false)

			// edit egressgateway dst from notDst to dst
			GinkgoWriter.Printf("change egressgatewaypolicy dst from %v to %v\n", notDst, dst)
			// update egressgatewaypolicy
			Expect(common.EditEgressPolicy(frame, egressPolicy, podBLabels, dst)).NotTo(HaveOccurred())

			GinkgoWriter.Println("restart podB, we expect podB export ip is eip")
			restartPodAndCheckEIP(deployBOjb, eIPs, true)

			By("P00003, delete egressgatewaypolicy")
			Expect(common.DeleteEgressPolicy(frame, egressPolicy)).NotTo(HaveOccurred())

			GinkgoWriter.Println("restart podB, we expect podB export ip is not eip")
			restartPodAndCheckEIP(deployBOjb, eIPs, false)
		})
	})
})

func restartPodAndCheckEIP(deploy *appsv1.Deployment, eIPs []string, expect bool) {
	// reboot deploy, we expect export ip is eip: expect
	podAList, err := frame.GetDeploymentPodList(deploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(podAList).NotTo(BeNil())
	restartedPodAList, err := frame.DeletePodListUntilReady(podAList, time.Minute, delOpts)
	Expect(err).NotTo(HaveOccurred())
	Expect(restartedPodAList).NotTo(BeNil())

	for _, eip := range eIPs {
		common.CheckEIP(frame, deploy, eip, expect, time.Second*30, time.Minute)
	}
}
