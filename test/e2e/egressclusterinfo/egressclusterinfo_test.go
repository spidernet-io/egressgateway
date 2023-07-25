// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/client"

	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"

	"github.com/spidernet-io/e2eframework/framework"
	egressv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("EgressClusterInfo", Label("EgressClusterInfo"), func() {
	var eci *egressv1beta1.EgressClusterInfo
	var calicoIPPools []string

	BeforeEach(func() {
		eci = new(egressv1beta1.EgressClusterInfo)
		calicoIPPools = make([]string, 0)

		DeferCleanup(func() {
			GinkgoWriter.Println("defer cleanup...")
			for _, calicoName := range calicoIPPools {
				GinkgoWriter.Printf("delete calico ippool '%s' if exists\n", calicoName)
				pool, err := common.GetCalicoIPPool(f, calicoName)
				Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())
				if pool != nil {
					GinkgoWriter.Printf("calico ippool '%s' already exists, delete it\n", calicoName)
					Expect(client.IgnoreNotFound(common.DeleteCalicoIPPool(f, pool))).NotTo(HaveOccurred())
				}
			}
		})
	})

	It("Get and delete EgressClusterInfo", Serial, Label("I00001", "I00007"), func() {
		// get and check EgressClusterInfo
		GinkgoWriter.Printf("get and check EgressClusterInfo %s\n", egressClusterInfoName)
		common.CheckEgressClusterInfoStatus(f, time.Second*20)

		// get EgressClusterInfo
		Expect(common.GetEgressClusterInfo(f, eci)).NotTo(HaveOccurred())

		// delete EgressClusterInfo
		GinkgoWriter.Printf("delete EgressClusterInfo, we expect it will be failed")
		Expect(f.DeleteResource(eci)).To(HaveOccurred())
	})

	It("Create or update calico ippool", Serial, Label("I00006"), func() {
		// CalicoIPPoolV6
		if enableV6 {
			createOrUpdateCalicoIPPoolAndCheck(f, "test-v6-", "112", &calicoIPPools, common.RandomIPPoolV6Cidr)
		}
		// CalicoIPPoolV4
		if enableV4 {
			createOrUpdateCalicoIPPoolAndCheck(f, "test-v4-", "24", &calicoIPPools, common.RandomIPPoolV4Cidr)
		}
	})

	DescribeTable("Edit egressClusterInfo spec", Serial, func(updateEgci func() error) {
		Expect(updateEgci()).NotTo(HaveOccurred())
		common.CheckEgressClusterInfoStatus(f, time.Second*10)
	},
		Entry("Update spec.autoDetect.clusterIP to be false", Label("I00002"), func() error {
			// get EgressClusterInfo
			Expect(common.GetEgressClusterInfo(f, eci)).NotTo(HaveOccurred())
			eci.Spec.AutoDetect.ClusterIP = false
			return common.UpdateEgressClusterInfo(f, eci, time.Second*10)
		}),
		Entry("Update spec.AutoDetect.NodeIP to be false", Label("I00003"), func() error {
			// get EgressClusterInfo
			Expect(common.GetEgressClusterInfo(f, eci)).NotTo(HaveOccurred())
			eci.Spec.AutoDetect.NodeIP = false
			return common.UpdateEgressClusterInfo(f, eci, time.Second*10)
		}),
		Entry("Update spec.AutoDetect.PodCidrMode to k8s", Label("I00004"), func() error {
			// get EgressClusterInfo
			Expect(common.GetEgressClusterInfo(f, eci)).NotTo(HaveOccurred())
			eci.Spec.AutoDetect.PodCidrMode = common.K8s
			return common.UpdateEgressClusterInfo(f, eci, time.Second*10)
		}),
		Entry("Update spec.ExtraCidr", Label("I00005"), func() error {
			// get EgressClusterInfo
			Expect(common.GetEgressClusterInfo(f, eci)).NotTo(HaveOccurred())
			eci.Spec.ExtraCidr = []string{"10.10.10.1"}
			return common.UpdateEgressClusterInfo(f, eci, time.Second*10)
		}),
	)
})

func createOrUpdateCalicoIPPoolAndCheck(f *framework.Framework, poolNamePre, cidrPrefix string, calicoIPPools *[]string, generateRandomCidr func(_ string) string) {
	// UpdateEgressClusterInfo spec.AutoDetect.PodCidrMode to calico
	eci := new(egressv1beta1.EgressClusterInfo)
	Expect(common.GetEgressClusterInfo(f, eci)).NotTo(HaveOccurred())
	eci.Spec.AutoDetect.PodCidrMode = common.Calico
	Expect(common.UpdateEgressClusterInfo(f, eci, time.Second*10)).NotTo(HaveOccurred())
	eci, err := common.WaitEgressClusterInfoPodCidrAndModeUpdated(f, common.Calico, time.Second*10)
	Expect(err).NotTo(HaveOccurred())
	Expect(eci).NotTo(BeNil())

	// CreateCalicoIPPool
	GinkgoWriter.Println("create calico ippool")
	ipPool := common.CreateCalicoIPPool(f, poolNamePre, cidrPrefix, generateRandomCidr)
	Expect(ipPool).NotTo(BeNil())
	*calicoIPPools = append(*calicoIPPools, ipPool.Name)

	// WaitCalicoIPPoolCreated
	GinkgoWriter.Printf("wait calico ippool %s created\n", ipPool.Name)
	ipPool, err = common.WaitCalicoIPPoolCreated(f, ipPool.Name, time.Second*10)
	Expect(err).NotTo(HaveOccurred())

	// WaitEgressClusterInfoPodCidrUpdated
	GinkgoWriter.Println("WaitEgressClusterInfoPodCidrUpdated after calicoIPPool created")
	eci, err = common.WaitEgressClusterInfoPodCidrAndModeUpdated(f, common.Calico, time.Second*30)
	Expect(err).NotTo(HaveOccurred())
	Expect(eci).NotTo(BeNil())

	// check EgressIgnoreCIDR Fields
	common.CheckEgressClusterInfoStatus(f, time.Second*20)

	// update calicoIPPool
	var updatedPool *calicov1.IPPool
	if enableV4 {
		// UpdateCalicoIPPoolCidr
		GinkgoWriter.Println("UpdateCalicoIPPoolCidr v4")
		updatedPool, err = common.UpdateCalicoIPPoolCidr(f, ipPool, "24", common.RandomIPPoolV4Cidr)
		Expect(err).NotTo(HaveOccurred())
	}

	if enableV6 && !enableV4 {
		// UpdateCalicoIPPoolCidr
		GinkgoWriter.Println("UpdateCalicoIPPoolCidr v6")
		updatedPool, err = common.UpdateCalicoIPPoolCidr(f, ipPool, "112", common.RandomIPPoolV6Cidr)
		Expect(err).NotTo(HaveOccurred())
	}

	Expect(updatedPool).NotTo(BeNil())

	// WaitCalicoIPPoolCidrUpdated
	GinkgoWriter.Println("WaitCalicoIPPoolCidrUpdated")
	Expect(common.WaitCalicoIPPoolCidrUpdated(f, updatedPool, time.Second*3)).NotTo(HaveOccurred())

	// WaitEgressClusterInfoPodCidrUpdated
	GinkgoWriter.Println("WaitEgressClusterInfoPodCidrUpdated after calicoIPPool updated")
	eci, err = common.WaitEgressClusterInfoPodCidrAndModeUpdated(f, common.Calico, time.Second*30)
	Expect(err).NotTo(HaveOccurred())
	Expect(eci).NotTo(BeNil())

	// check EgressIgnoreCIDR Fields
	common.CheckEgressClusterInfoStatus(f, time.Second*20)

	// DeleteCalicoIPPool
	GinkgoWriter.Printf("delete calico ippool %s\n", ipPool.Name)
	Expect(common.DeleteCalicoIPPool(f, ipPool)).NotTo(HaveOccurred())

	// WaitCalicoIPPoolDeleted
	GinkgoWriter.Printf("wait calico ippool %s deleted\n", ipPool.Name)
	Expect(common.WaitCalicoIPPoolDeleted(f, ipPool.Name, time.Second*10)).NotTo(HaveOccurred())

	// WaitEgressClusterInfoPodCidrUpdated
	GinkgoWriter.Printf("wait egressClusterInfo updated after calico ippool '%s' deleted, prevent affecting other cases\n", ipPool.Name)
	_, err = common.WaitEgressClusterInfoPodCidrAndModeUpdated(f, common.Calico, time.Second*5)
	Expect(err).NotTo(HaveOccurred())
}
