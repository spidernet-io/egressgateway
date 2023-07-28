// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo_test

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Egressclusterinfo", Label("Egressclusterinfo"), func() {
	//var eci *egressv1beta1.EgressClusterInfo
	//var calicoIPPools []string
	//
	//BeforeEach(func() {
	//	eci = new(egressv1beta1.EgressClusterInfo)
	//	calicoIPPools = make([]string, 0)
	//
	//	DeferCleanup(func() {
	//		GinkgoWriter.Println("defer cleanup...")
	//		for _, calicoName := range calicoIPPools {
	//			GinkgoWriter.Printf("delete calico ippool '%s' if exists\n", calicoName)
	//			pool, err := common.GetCalicoIPPool(f, calicoName)
	//			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())
	//			if pool != nil {
	//				GinkgoWriter.Printf("calico ippool '%s' already exists, delete it\n", calicoName)
	//				Expect(client.IgnoreNotFound(common.DeleteCalicoIPPool(f, pool))).NotTo(HaveOccurred())
	//			}
	//		}
	//	})
	//})
	//
	//It("get and delete EgressClusterInfo", Serial, Label("I00001", "I00003"), func() {
	//	// get and check EgressClusterInfo
	//	GinkgoWriter.Printf("get and check EgressClusterInfo %s\n", egressClusterInfoName)
	//	common.CheckEgressIgnoreCIDRFields(f, time.Second*20)
	//
	//	// get EgressClusterInfo
	//	Expect(common.GetEgressClusterInfo(f, egressClusterInfoName, eci)).NotTo(HaveOccurred())
	//
	//	// delete EgressClusterInfo
	//	GinkgoWriter.Printf("delete EgressClusterInfo, we expect it will be failed")
	//	Expect(f.DeleteResource(eci)).To(HaveOccurred())
	//})
	//
	//It("create or update calico ippool", Serial, Label("I00002"), func() {
	//	// CalicoIPPoolV6
	//	if enableV6 {
	//		createOrUpdateCalicoIPPoolAndCheck(f, "test-v6-", "112", &calicoIPPools, common.RandomIPPoolV6Cidr)
	//	}
	//	// CalicoIPPoolV4
	//	if enableV4 {
	//		createOrUpdateCalicoIPPoolAndCheck(f, "test-v4-", "24", &calicoIPPools, common.RandomIPPoolV4Cidr)
	//	}
	//})
})

//func createOrUpdateCalicoIPPoolAndCheck(f *framework.Framework, poolNamePre, cidrPrefix string, calicoIPPools *[]string, generateRandomCidr func(_ string) string) {
//	// CreateCalicoIPPool
//	GinkgoWriter.Println("create calico ippool")
//	ipPool := common.CreateCalicoIPPool(f, poolNamePre, cidrPrefix, generateRandomCidr)
//	Expect(ipPool).NotTo(BeNil())
//	*calicoIPPools = append(*calicoIPPools, ipPool.Name)
//
//	// WaitCalicoIPPoolCreated
//	GinkgoWriter.Printf("wait calico ippool %s created\n", ipPool.Name)
//	ipPool, err = common.WaitCalicoIPPoolCreated(f, ipPool.Name, time.Second*10)
//	Expect(err).NotTo(HaveOccurred())
//
//	// WaitEgressClusterInfoPodCidrUpdated
//	GinkgoWriter.Println("WaitEgressClusterInfoPodCidrUpdated after calicoIPPool created")
//	eci, err := common.WaitEgressClusterInfoPodCidrUpdated(f, common.CALICO, time.Second*30)
//	Expect(err).NotTo(HaveOccurred())
//	Expect(eci).NotTo(BeNil())
//
//	// check EgressIgnoreCIDR Fields
//	common.CheckEgressIgnoreCIDRFields(f, time.Second*20)
//
//	// update calicoIPPool
//	var updatedPool *calicov1.IPPool
//	if enableV4 {
//		// UpdateCalicoIPPoolCidr
//		GinkgoWriter.Println("UpdateCalicoIPPoolCidr v4")
//		updatedPool, err = common.UpdateCalicoIPPoolCidr(f, ipPool, "24", common.RandomIPPoolV4Cidr)
//		Expect(err).NotTo(HaveOccurred())
//	}
//
//	if enableV6 && !enableV4 {
//		// UpdateCalicoIPPoolCidr
//		GinkgoWriter.Println("UpdateCalicoIPPoolCidr v6")
//		updatedPool, err = common.UpdateCalicoIPPoolCidr(f, ipPool, "112", common.RandomIPPoolV6Cidr)
//		Expect(err).NotTo(HaveOccurred())
//	}
//
//	Expect(updatedPool).NotTo(BeNil())
//
//	// WaitCalicoIPPoolCidrUpdated
//	GinkgoWriter.Println("WaitCalicoIPPoolCidrUpdated")
//	Expect(common.WaitCalicoIPPoolCidrUpdated(f, updatedPool, time.Second*3)).NotTo(HaveOccurred())
//
//	// WaitEgressClusterInfoPodCidrUpdated
//	GinkgoWriter.Println("WaitEgressClusterInfoPodCidrUpdated after calicoIPPool updated")
//	eci, err = common.WaitEgressClusterInfoPodCidrUpdated(f, common.CALICO, time.Second*30)
//	Expect(err).NotTo(HaveOccurred())
//	Expect(eci).NotTo(BeNil())
//
//	// check EgressIgnoreCIDR Fields
//	common.CheckEgressIgnoreCIDRFields(f, time.Second*20)
//
//	// DeleteCalicoIPPool
//	GinkgoWriter.Printf("delete calico ippool %s\n", ipPool.Name)
//	Expect(common.DeleteCalicoIPPool(f, ipPool)).NotTo(HaveOccurred())
//
//	// WaitCalicoIPPoolDeleted
//	GinkgoWriter.Printf("wait calico ippool %s deleted\n", ipPool.Name)
//	Expect(common.WaitCalicoIPPoolDeleted(f, ipPool.Name, time.Second*10)).NotTo(HaveOccurred())
//
//	// WaitEgressClusterInfoPodCidrUpdated
//	GinkgoWriter.Printf("wait egressClusterInfo updated after calico ippool '%s' deleted, prevent affecting other cases\n", ipPool.Name)
//	_, err = common.WaitEgressClusterInfoPodCidrUpdated(f, common.CALICO, time.Second*5)
//	Expect(err).NotTo(HaveOccurred())
//}
