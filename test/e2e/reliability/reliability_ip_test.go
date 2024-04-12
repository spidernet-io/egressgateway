// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package reliability_test

import (
	"context"
	"fmt"
	"time"

	"github.com/go-faker/faker/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("IP Allocation", Label("Reliability_IP"), func() {
	var pods []*corev1.Pod
	var egw *egressv1.EgressGateway
	var p2p map[*corev1.Pod]*egressv1.EgressPolicy
	var egps, newEgps []*egressv1.EgressPolicy
	var IPNum, extraNum int64
	var err error

	var ctx context.Context

	const (
		creationThresholdTime = time.Second * 30
		deletionThresholdTime = time.Second * 30
	)

	BeforeEach(func() {
		ctx = context.Background()
		IPNum = 50
		extraNum = 20

		// create EgressGateway and pods
		egw, pods, err = common.CreateEgressGatewayAndPodsBeforeEach(ctx, cli, egressConfig.EnableIPv4, egressConfig.EnableIPv6, nodeNameList, config.Image, IPNum, 1)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed create egw or pods: %v\n", err))
		GinkgoWriter.Printf("succeeded create egw: %s\n", egw.Name)

		DeferCleanup(func() {
			// delete pods
			for _, pod := range pods {
				Expect(common.DeleteObj(ctx, cli, pod)).NotTo(HaveOccurred())
			}
			// delete policy if exists
			Expect(common.WaitEgressPoliciesDeleted(ctx, cli, egps, time.Minute)).NotTo(HaveOccurred())
			Expect(common.WaitEgressPoliciesDeleted(ctx, cli, newEgps, time.Minute)).NotTo(HaveOccurred())
			// delete egressGateway
			Expect(common.DeleteObj(ctx, cli, egw)).NotTo(HaveOccurred())
		})
	})

	// case R00008 steps:
	// (1) In the beforeEach block, we create a gateway and set up a pool with 100 IPs.
	// (2) Create 120 policies.
	// (3) Delete all policies.
	// (4) Create 120 policies again (start timing before creation).
	// (5) Expect that 100 policies are assigned IP addresses, while 20 policies are not assigned IPs (after creation is complete).
	// (6) Check that the gateway status correctly synchronizes with the status of all policies (after creation is complete, calculate the time spent).
	// (7) Verify that there is a one-to-one correspondence between 100 IPs and 100 policies.
	// (8) Check 100 pod egress IPs; they should match the EIPs used by the policies.
	// (9) Delete all policies (start timing).
	// (10) Verify that the gateway status has synchronized successfully, and all IPs have been released (after deletion is complete, calculate the time spent).
	// (11) Check pod egress IPs again; they should no longer match the previous EIPs.
	It("test IP allocation", Label("R00008", "P00009"), Serial, func() {
		// create egresspolicies
		By("create egressPolicies by gaven pods")
		egps, _, err = common.CreateEgressPoliciesForPods(ctx, cli, egw, pods, egressConfig.EnableIPv4, egressConfig.EnableIPv6, creationThresholdTime)
		Expect(err).NotTo(HaveOccurred())

		By("create extra egressPolicies")
		fakeLabels := map[string]string{
			"app": faker.Word(),
		}
		for i := 0; i < int(extraNum); i++ {
			egp, err := common.CreateEgressPolicyWithEipAllocatorRR(ctx, cli, egw, fakeLabels)
			Expect(err).NotTo(HaveOccurred())
			egps = append(egps, egp)
		}

		// delete egresspolicies
		By("delete all egressPolicies")
		Expect(common.DeleteEgressPolicies(ctx, cli, egps)).NotTo(HaveOccurred())

		creationStart := time.Now()
		// recreate egresspolicies
		By("create egressPolicies by gaven pods")
		newEgps, p2p, err = common.CreateEgressPoliciesForPods(ctx, cli, egw, pods, egressConfig.EnableIPv4, egressConfig.EnableIPv6, creationThresholdTime)
		Expect(err).NotTo(HaveOccurred())

		By("create another extra egressPolicies")
		var extraEgps []*egressv1.EgressPolicy
		for i := 0; i < int(extraNum); i++ {
			egp, err := common.CreateEgressPolicyWithEipAllocatorRR(ctx, cli, egw, fakeLabels)
			Expect(err).NotTo(HaveOccurred())
			extraEgps = append(extraEgps, egp)
			newEgps = append(newEgps, egp)
		}

		// check egressgateway status synced with egresspolicy
		By("check egressgateway status synced with egresspolicies")
		err := common.WaitEGWSyncedWithEGP(cli, egw, egressConfig.EnableIPv4, egressConfig.EnableIPv6, int(IPNum), time.Minute)
		Expect(err).NotTo(HaveOccurred())
		creationTime := time.Since(creationStart)

		// check egessgateway ip number
		if egressConfig.EnableIPv4 {
			Expect(egw.Status.IPUsage.IPv4Free).To(BeZero())
			Expect(egw.Status.IPUsage.IPv4Total).To(Equal(int(IPNum)))
		}

		if egressConfig.EnableIPv6 {
			Expect(egw.Status.IPUsage.IPv6Free).To(BeZero())
			Expect(egw.Status.IPUsage.IPv6Total).To(Equal(int(IPNum)))
		}

		// check eip
		By("check eip of pods")
		Expect(common.CheckPodsEgressIP(ctx, config, p2p, egressConfig.EnableIPv4, egressConfig.EnableIPv6, true)).NotTo(HaveOccurred(), "failed check eip")

		// check extra egresspolicies which should not allocate ip
		for _, egp := range extraEgps {
			Expect(egp.Status.Eip.Ipv4).To(BeEmpty(), fmt.Sprintf("failed check extra egp:\n%s\n", common.GetObjYAML(egp)))
			Expect(egp.Status.Eip.Ipv6).To(BeEmpty(), fmt.Sprintf("failed check extra egp:\n%s\n", common.GetObjYAML(egp)))
		}

		deletionStart := time.Now()
		// delete all policies
		By("delete all egressPolicies")
		Expect(common.WaitEgressPoliciesDeleted(ctx, cli, newEgps, time.Minute)).NotTo(HaveOccurred())

		// check eip after policies deleted
		By("check egressgateway status should be empty")
		Eventually(ctx, func() []egressv1.Eips {
			eips := make([]egressv1.Eips, 0)
			_ = cli.Get(ctx, types.NamespacedName{Namespace: egw.Namespace, Name: egw.Name}, egw)
			for _, eipStatus := range egw.Status.NodeList {
				eips = append(eips, eipStatus.Eips...)
			}
			return eips
		}).WithTimeout(time.Minute*2).WithPolling(time.Second*2).Should(BeEmpty(),
			fmt.Sprintf("failed to wait the egressgateway: %s status to be empty, egressgateway yaml: %v", egw.Name, egw))
		deletionTime := time.Since(deletionStart)

		// check egessgateway ip number
		By("check egressgateway status IPUsage")
		if egressConfig.EnableIPv4 {
			Expect(egw.Status.IPUsage.IPv4Free).To(Equal(int(IPNum)))
			Expect(egw.Status.IPUsage.IPv4Total).To(Equal(int(IPNum)))
		}

		if egressConfig.EnableIPv6 {
			Expect(egw.Status.IPUsage.IPv6Free).To(Equal(int(IPNum)))
			Expect(egw.Status.IPUsage.IPv6Total).To(Equal(int(IPNum)))
		}

		By("check eip of pods")
		Expect(common.CheckPodsEgressIP(ctx, config, p2p, egressConfig.EnableIPv4, egressConfig.EnableIPv6, false)).NotTo(HaveOccurred(), "failed check eip")
		Expect(err).NotTo(HaveOccurred())

		// report the creation time and deletion time
		GinkgoWriter.Printf("IP allocation takes: %s\n", creationTime)
		GinkgoWriter.Printf("IP release takes: %s\n", deletionTime)
	})
})
