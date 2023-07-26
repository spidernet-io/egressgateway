// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package reliability_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	egressgatewayv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

var _ = Describe("Reliability", func() {
	Context("Test reliability", Label("Reliability"), func() {
		var (
			// gateway
			egressGatewayName    string
			IPPools              egressgatewayv1beta1.Ippools
			nodeSelector         egressgatewayv1beta1.NodeSelector
			egNodes              []string
			nodeNameA, nodeNameB string

			// policy
			egressPolicyName string
			emptyEgressIP    egressgatewayv1beta1.EgressIP
			egressPolicy     *egressgatewayv1beta1.EgressPolicy

			// pod
			dsName              string
			podList             *corev1.PodList
			podLabel, nodeLabel map[string]string

			// eip
			v4Eip, v6Eip string
		)

		BeforeEach(func() {
			// label nodes
			egNodes = workers[:2]
			nodeLabel = map[string]string{"eg": "true"}
			Expect(common.LabelNodes(f, egNodes, nodeLabel)).NotTo(HaveOccurred())

			nodesByMatchLabels, err := common.GetNodesByMatchLabels(f, nodeLabel)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(nodesByMatchLabels)).To(Equal(2))

			// gateway
			egressGatewayName = tools.GenerateRandomName("eg")
			IPPools = common.GenerateRangeEgressGatewayIPPools(f, 3)
			nodeSelector = egressgatewayv1beta1.NodeSelector{Selector: &v1.LabelSelector{MatchLabels: nodeLabel}}

			GinkgoWriter.Printf("Create egressGateway: %s\n", egressGatewayName)
			egressGatewayYaml := common.GenerateEgressGatewayYaml(egressGatewayName, IPPools, nodeSelector)
			Expect(err).NotTo(HaveOccurred())
			Expect(common.CreateEgressGateway(f, egressGatewayYaml)).NotTo(HaveOccurred(), "egressGatewayYaml: ", common.YamlMarshal(egressGatewayYaml))

			GinkgoWriter.Println("WaitEgressGatewayDefaultEIPUpdated")
			v4DefaultEip, v6DefaultEip, err := common.WaitEgressGatewayDefaultEIPUpdated(f, egressGatewayName, v4Enabled, v6Enabled, time.Second*5)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("v4DefaultEip: %s, v6DefaultEip: %s\n", v4DefaultEip, v6DefaultEip)

			// daemonSet
			dsName = tools.GenerateRandomName("ds")
			podLabel = map[string]string{"app": dsName}

			GinkgoWriter.Printf("Create ds: %s until ready\n", dsName)
			dsObj := common.GenerateDSYaml(dsName, podLabel)
			podList, err = common.CreateDSUntilReady(f, dsObj, time.Second*30)
			Expect(err).NotTo(HaveOccurred())
			Expect(podList).NotTo(BeNil())

			// policy
			egressPolicyName = tools.GenerateRandomName("policy")
			emptyEgressIP = egressgatewayv1beta1.EgressIP{}
			egressPolicy = new(egressgatewayv1beta1.EgressPolicy)

			GinkgoWriter.Printf("Create egressPolicy: %s\n", egressPolicyName)
			egressPolicyYaml := common.GenerateEgressPolicyYaml(egressPolicyName, egressGatewayName, common.NSDefault, emptyEgressIP, podLabel, nil, dst)
			Expect(common.CreateEgressPolicy(f, egressPolicyYaml)).NotTo(HaveOccurred(), "egressPolicyYaml: ", common.YamlMarshal(egressPolicyYaml))

			// wait policy status about eip updated
			v4Eip, v6Eip, err = common.WaitEgressPolicyEipUpdated(f, egressPolicyName, common.NSDefault, v4DefaultEip, v6DefaultEip, v4Enabled, v6Enabled, time.Second*3)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("v4Eip: %s, v6Eip: %s\n", v4Eip, v6Eip)

			Expect(common.GetEgressPolicy(f, egressPolicyName, common.NSDefault, egressPolicy)).NotTo(HaveOccurred())

			// wait gateway status about policy updated
			nodeNameA, _, _, err = common.WaitEgressGatewayStatusUpdated(f, egressPolicy, time.Second*10)
			Expect(err).NotTo(HaveOccurred())
			Expect(nodeNameA).NotTo(BeEmpty())
			GinkgoWriter.Printf("The node where eip takes effect is %s\n", nodeNameA)

			// check eip
			GinkgoWriter.Println("Check eip in pod")
			checkEip(podList, v4Eip, v6Eip, true, 3, time.Second*5)
		})

		AfterEach(func() {
			// restore the cluster to avoid affecting the execution of other use cases
			GinkgoWriter.Println("PowerOnNodesUntilClusterReady")
			Expect(common.PowerOnNodesUntilClusterReady(f, nodes, time.Second*30)).NotTo(HaveOccurred())

			// delete pod if its exists
			GinkgoWriter.Println("Delete test pod if its exists")
			Expect(common.DeleteDSIfExists(f, dsName, common.NSDefault, time.Minute)).NotTo(HaveOccurred())

			// delete egressPolicy if its exists
			GinkgoWriter.Println("Delete egressPolicy if its exists")
			Expect(common.DeleteEgressPolicyIfExists(f, egressPolicyName, common.NSDefault, egressPolicy, time.Second*10)).NotTo(HaveOccurred())

			// delete egressGateway if its exists
			GinkgoWriter.Println("Delete egressGateway if its exists")
			Expect(common.DeleteEgressGatewayIfExists(f, egressGatewayName, time.Second*10)).NotTo(HaveOccurred())

			// un label nodes
			GinkgoWriter.Println("UnLabelNodes")
			Expect(common.UnLabelNodes(f, egNodes, nodeLabel)).NotTo(HaveOccurred())
		})

		// todo bzsuni
		PIt("Test EIP drift after then eip-node shut down", Serial, Label("R00005"), func() {
			// shut down the eip node
			GinkgoWriter.Printf("Shut down node: %s\n", nodeNameA)
			Expect(common.PowerOffNodeUntilNotReady(f, nodeNameA, time.Minute)).NotTo(HaveOccurred())

			// check if eip drift after node shut down
			GinkgoWriter.Println("Check if eip drift after node shut down")
			bs := tools.SubtractionSlice(workers, []string{nodeNameA})
			Expect(bs).NotTo(BeEmpty())
			nodeNameB = bs[0]
			Expect(nodeNameB).NotTo(BeEmpty())
			GinkgoWriter.Printf("We expect the eip will drift to node: %s\n", nodeNameB)
			Expect(common.WaitEipToExpectNode(f, nodeNameB, egressPolicy, time.Second*10)).NotTo(HaveOccurred())

			// check the running pod's export IP is eip
			GinkgoWriter.Println("Check the eip in running pods after shut down the eip node")
			list, err := common.ListNodesPod(f, podLabel, tools.SubtractionSlice(nodes, []string{nodeNameA}))
			Expect(err).NotTo(HaveOccurred())
			checkEip(list, v4Eip, v6Eip, true, 3, time.Second*5)

			// power on the node and wait cluster ready
			GinkgoWriter.Println("PowerOnNodesUntilClusterReady")
			Expect(common.PowerOnNodesUntilClusterReady(f, nodes, time.Second*30)).NotTo(HaveOccurred())
		})
	})
})

func checkEip(podList *corev1.PodList, v4Eip, v6Eip string, expect bool, retry int, timeout time.Duration) {
	for i, pod := range podList.Items {
		GinkgoWriter.Printf("Checking in %dth pod: %s\n", i, pod.Name)
		if v4Enabled {
			Expect(v4Eip).NotTo(BeEmpty())
			Expect(common.CheckEipInClientPod(f, &pod, v4Eip, serverIPv4, expect, retry, timeout)).NotTo(HaveOccurred())
		}
		if v6Enabled {
			Expect(v6Eip).NotTo(BeEmpty())
			Expect(common.CheckEipInClientPod(f, &pod, v6Eip, serverIPv6, expect, retry, timeout)).NotTo(HaveOccurred())
		}
	}
}
