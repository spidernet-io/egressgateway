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
		creationThresholdTime = time.Second * 10
		deletionThresholdTime = time.Second * 10
	)

	BeforeEach(func() {
		ctx = context.Background()
		IPNum = 100
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
			Expect(common.WaitEgressPoliciesDeleted(ctx, cli, egps, time.Second*10)).NotTo(HaveOccurred())
			Expect(common.WaitEgressPoliciesDeleted(ctx, cli, newEgps, time.Second*10)).NotTo(HaveOccurred())
			// delete egressGateway
			Expect(common.DeleteObj(ctx, cli, egw)).NotTo(HaveOccurred())
		})
	})

	// todo @bzsuni wait the bug fixed
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
		egps, _, err = common.CreateEgressPoliciesForPods(ctx, cli, egw, pods, egressConfig.EnableIPv4, egressConfig.EnableIPv6, time.Second*5)
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
		// todo @bzsuni we do not wait all policies delete here
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
		err := common.WaitEGWSyncedWithEGP(cli, egw, egressConfig.EnableIPv4, egressConfig.EnableIPv6, int(IPNum), time.Second*10)
		Expect(err).NotTo(HaveOccurred())
		creationTime := time.Since(creationStart)

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
		Expect(common.WaitEgressPoliciesDeleted(ctx, cli, newEgps, time.Second*10)).NotTo(HaveOccurred())

		// check eip after policies deleted
		By("check egressgateway status should be empty")
		err = common.WaitEGWSyncedWithEGP(cli, egw, egressConfig.EnableIPv4, egressConfig.EnableIPv6, 0, deletionThresholdTime)
		Expect(err).NotTo(HaveOccurred())
		deletionTime := time.Since(deletionStart)

		By("check eip of pods")
		Expect(common.CheckPodsEgressIP(ctx, config, p2p, egressConfig.EnableIPv4, egressConfig.EnableIPv6, false)).NotTo(HaveOccurred(), "failed check eip")
		Expect(err).NotTo(HaveOccurred())

		// report the creation time and deletion time
		GinkgoWriter.Printf("IP allocation takes: %s\n", creationTime)
		GinkgoWriter.Printf("IP release takes: %s\n", deletionTime)
	})
})
