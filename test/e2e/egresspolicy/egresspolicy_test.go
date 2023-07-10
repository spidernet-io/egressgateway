// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egresspolicy_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressgatewayv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

var _ = Describe("EgressPolicy", func() {
	Context("Test egressGatewayPolicy", Label("EgressPolicy"), func() {
		var (
			// gateway
			egressGatewayName string
			IPPools           egressgatewayv1beta1.Ippools
			nodeSelector      egressgatewayv1beta1.NodeSelector

			// policy
			egressPolicyName    string
			emptyEgressIP       egressgatewayv1beta1.EgressIP
			egressPolicy        *egressgatewayv1beta1.EgressPolicy
			egressClusterPolicy *egressgatewayv1beta1.EgressClusterPolicy

			unmatchedIPv4, unmatchedIPv6 string
			unmatchedDesSubnet           []string

			// pod
			dsNameA, dsNameB     string
			podAList, podBList   *corev1.PodList
			podALabel, podBLabel map[string]string
		)

		BeforeEach(func() {
			// gateway
			egressGatewayName = tools.GenerateRandomName("eg")
			IPPools = common.GenerateRangeEgressGatewayIPPools(f, 3)
			nodeSelector = egressgatewayv1beta1.NodeSelector{Selector: &v1.LabelSelector{MatchLabels: nodeObjs[0].Labels}}

			GinkgoWriter.Printf("Create egressGateway: %s\n", egressGatewayName)
			egressGatewayYaml := common.GenerateEgressGatewayYaml(egressGatewayName, IPPools, nodeSelector)
			Expect(common.CreateEgressGateway(f, egressGatewayYaml)).NotTo(HaveOccurred(), "egressGatewayYaml: ", egressGatewayYaml)

			// daemonSet A and daemonSet B
			dsNameA = tools.GenerateRandomName("dsa")
			dsNameB = tools.GenerateRandomName("dsb")
			podALabel = map[string]string{"app": dsNameA}
			podBLabel = map[string]string{"app": dsNameB}

			GinkgoWriter.Printf("Create dsA: %s until ready\n", dsNameA)
			dsObj := common.GenerateDSYaml(dsNameA, podALabel)
			podAList, err = common.CreateDSUntilReady(f, dsObj, time.Second*30)
			Expect(err).NotTo(HaveOccurred())
			Expect(podAList).NotTo(BeNil())

			GinkgoWriter.Printf("Create dsB: %s until ready\n", dsNameB)
			dsObj1 := common.GenerateDSYaml(dsNameB, podBLabel)
			podBList, err = common.CreateDSUntilReady(f, dsObj1, time.Second*30)
			Expect(err).NotTo(HaveOccurred())
			Expect(podBList).NotTo(BeNil())

			// policy
			egressPolicyName = tools.GenerateRandomName("policy")
			emptyEgressIP = egressgatewayv1beta1.EgressIP{}
			egressPolicy = new(egressgatewayv1beta1.EgressPolicy)
			egressClusterPolicy = new(egressgatewayv1beta1.EgressClusterPolicy)

			if v4Enabled {
				unmatchedIPv4 = common.RandomIPPoolV4Cidr("32")
				unmatchedDesSubnet = append(unmatchedDesSubnet, unmatchedIPv4)
			}
			if v6Enabled {
				unmatchedIPv6 = common.RandomIPPoolV6Cidr("128")
				unmatchedDesSubnet = append(unmatchedDesSubnet, unmatchedIPv6)
			}

			DeferCleanup(func() {
				// delete pod if its exists
				GinkgoWriter.Println("delete test podA if its exists")
				Expect(common.DeleteDSIfExists(f, dsNameA, common.NSDefault, time.Minute)).NotTo(HaveOccurred())
				GinkgoWriter.Println("delete test podB if its exists")
				Expect(common.DeleteDSIfExists(f, dsNameB, common.NSDefault, time.Minute)).NotTo(HaveOccurred())

				// delete egressPolicy if its exists
				GinkgoWriter.Println("delete egressPolicy if its exists")
				Expect(common.DeleteEgressPolicyIfExists(f, egressPolicyName, common.NSDefault, egressPolicy, time.Second*10)).NotTo(HaveOccurred())
				GinkgoWriter.Println("delete egressClusterPolicy if its exists")
				Expect(common.DeleteEgressPolicyIfExists(f, egressPolicyName, "", egressClusterPolicy, time.Second*10)).NotTo(HaveOccurred())

				// delete egressGateway if its exists
				GinkgoWriter.Println("delete egressGateway if its exists")
				Expect(common.DeleteEgressGatewayIfExists(f, egressGatewayName, time.Second*10)).NotTo(HaveOccurred())
			})
		})

		DescribeTable("Test policy", Serial, func(isGlobal bool, createPolicy func() error) {
			By("case P00008: create policy with empty `EgressIP`")
			// createPolicy
			GinkgoWriter.Printf("Create egressPolicy: %s\n", egressPolicyName)
			Expect(createPolicy()).NotTo(HaveOccurred())

			var v4Eip, v6Eip string
			if isGlobal {
				// WaitEgressClusterPolicyEipUpdated
				GinkgoWriter.Printf("WaitEgressClusterPolicyEipUpdated...\n")
				v4Eip, v6Eip, err = common.WaitEgressClusterPolicyEipUpdated(f, egressPolicyName, "", "", v4Enabled, v6Enabled, time.Second*3)
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Printf("v4Eip: %s, v6Eip: %s\n", v4Eip, v6Eip)
			} else {
				// WaitEgressPolicyEipUpdated
				GinkgoWriter.Printf("WaitEgressPolicyEipUpdated...\n")
				v4Eip, v6Eip, err = common.WaitEgressPolicyEipUpdated(f, egressPolicyName, common.NSDefault, "", "", v4Enabled, v6Enabled, time.Second*3)
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Printf("v4Eip: %s, v6Eip: %s\n", v4Eip, v6Eip)
			}

			// check eip in podA
			GinkgoWriter.Printf("Check export ip in dsA: %s pods\n", dsNameA)
			checkEip(podAList, v4Eip, v6Eip, true, time.Second*20)

			// update policy matched podA to match podB
			By("case P00014: update policy matched podA to match podB")
			if isGlobal {
				// update egressClusterPolicy
				updatePolicy(egressPolicyName, egressClusterPolicy, podBLabel, nil)
			} else {
				// update egressPolicy
				updatePolicy(egressPolicyName, egressPolicy, podBLabel, nil)
			}

			// check eip in podB, we expect pods export ip is eip
			GinkgoWriter.Printf("Check export ip in dsB: %s pods\n", dsNameB)
			checkEip(podBList, v4Eip, v6Eip, true, time.Second*20)

			// check eip in podA, we expect pods export ip is not eip
			GinkgoWriter.Printf("Check export ip in dsA: %s pods\n", dsNameA)
			checkEip(podAList, v4Eip, v6Eip, false, time.Second*20)

			By("case P00013: update policy to unmatched `DestSubnet`")
			// update policy `DestSubnet`, set it does not match with external ip
			if isGlobal {
				// update egressPolicy
				updatePolicy(egressPolicyName, egressClusterPolicy, nil, unmatchedDesSubnet)
			} else {
				// update egressClusterPolicy
				updatePolicy(egressPolicyName, egressPolicy, nil, unmatchedDesSubnet)
			}

			// check eip in podB, we expect pods export ip is not eip
			GinkgoWriter.Printf("Check export ip in dsB: %s pods\n", dsNameB)
			checkEip(podBList, v4Eip, v6Eip, false, time.Second*20)

			// update policy `DestSubnet`, set it match with external ip
			if isGlobal {
				updatePolicy(egressPolicyName, egressClusterPolicy, nil, dst)
			} else {
				updatePolicy(egressPolicyName, egressPolicy, nil, dst)
			}

			// check eip in podB, we expect pods export ip is eip
			GinkgoWriter.Printf("Check export ip in dsB: %s pods\n", dsNameB)
			checkEip(podBList, v4Eip, v6Eip, true, time.Second*20)

			// delete policy
			By("case P00019: delete policy, we expect pod's export ip is not eip")
			if isGlobal {
				GinkgoWriter.Printf("Delete policy: %s\n", egressPolicyName)
				Expect(common.DeleteEgressPolicy(f, egressClusterPolicy)).NotTo(HaveOccurred())
				checkEip(podBList, v4Eip, v6Eip, false, time.Second*20)
			} else {
				GinkgoWriter.Printf("Delete policy: %s\n", egressPolicyName)
				Expect(common.DeleteEgressPolicy(f, egressPolicy)).NotTo(HaveOccurred())
				checkEip(podBList, v4Eip, v6Eip, false, time.Second*20)
			}
		},
			PEntry("When global-level", true, func() error {
				GinkgoWriter.Printf("GenerateEgressClusterPolicyYaml: %s\n", egressPolicyName)
				return common.CreateEgressPolicy(f, common.GenerateEgressClusterPolicyYaml(egressPolicyName, egressGatewayName, emptyEgressIP, podALabel, nil, dst))
			}),
			Entry("When namespace-level", false, func() error {
				GinkgoWriter.Printf("GenerateEgressPolicyYaml: %s\n", egressPolicyName)
				return common.CreateEgressPolicy(f, common.GenerateEgressPolicyYaml(egressPolicyName, egressGatewayName, common.NSDefault, emptyEgressIP, podALabel, nil, dst))
			}),
		)
	})
})

func updatePolicy(policyName string, obj client.Object, label map[string]string, dst []string) {
	GinkgoWriter.Printf("Update policy...\n")
	// Get policy
	Expect(common.GetEgressPolicy(f, policyName, common.NSDefault, obj)).NotTo(HaveOccurred())
	// Edit policy
	p, pOk := obj.(*egressgatewayv1beta1.EgressPolicy)
	if pOk {
		Expect(common.EditEgressPolicy(f, p, label, dst)).NotTo(HaveOccurred())
	}
	cp, cpOk := obj.(*egressgatewayv1beta1.EgressClusterPolicy)
	if cpOk {
		Expect(common.EditEgressClusterPolicy(f, cp, label, dst)).NotTo(HaveOccurred())
	}
}

func checkEip(podList *corev1.PodList, v4Eip, v6Eip string, expect bool, timeout time.Duration) {
	for i, pod := range podList.Items {
		GinkgoWriter.Printf("checking in %dth pod: %s\n", i, pod.Name)
		if v4Enabled {
			Expect(v4Eip).NotTo(BeEmpty())
			Expect(common.CheckEIPinClientPod(f, &pod, v4Eip, serverIPv4, expect, timeout)).NotTo(HaveOccurred())
		}
		if v6Enabled {
			Expect(v6Eip).NotTo(BeEmpty())
			Expect(common.CheckEIPinClientPod(f, &pod, v6Eip, serverIPv6, expect, timeout)).NotTo(HaveOccurred())
		}
	}
}
