// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spidernet-io/e2eframework/framework"
	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("Egressclusterinfo", Label("Egressclusterinfo"), func() {
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

	It("get and delete EgressClusterInfo", Serial, Label("I00001", "I00003"), func() {
		// get EgressClusterInfo
		GinkgoWriter.Printf("get EgressClusterInfo %s\n", egressClusterInfoName)
		Expect(common.GetEgressClusterInfo(f, egressClusterInfoName, eci)).NotTo(HaveOccurred())

		// check EgressIgnoreCIDR Fields
		checkEgressIgnoreCIDRFields(eci)

		// delete EgressClusterInfo
		Expect(f.DeleteResource(eci)).To(HaveOccurred())
	})

	It("create or update calico ippool", Serial, Label("I00002"), func() {
		// CalicoIPPoolV6
		if enableV6 {
			createOrUpdateCalicoIPPoolAndCheck(f, "test-v6-", &calicoIPPools, common.RandomIPPoolV6Cidr, eci)
		}
		// CalicoIPPoolV4
		if enableV4 {
			createOrUpdateCalicoIPPoolAndCheck(f, "test-v4-", &calicoIPPools, common.RandomIPPoolV4Cidr, eci)
		}
	})
})

func checkEgressIgnoreCIDRFields(eci *egressv1beta1.EgressClusterInfo) {
	GinkgoWriter.Println("check EgressIgnoreCIDR Fields")
	var (
		eciNodesIPv4   = make([]string, 0)
		eciNodesIPv6   = make([]string, 0)
		eciClusterIPv4 = make([]string, 0)
		eciClusterIPv6 = make([]string, 0)
		eciPodCidrIPv4 = make([]string, 0)
		eciPodCidrIPv6 = make([]string, 0)
	)
	if ipv4 := eci.Status.EgressIgnoreCIDR.NodeIP.IPv4; ipv4 != nil {
		eciNodesIPv4 = ipv4
	}
	if ipv6 := eci.Status.EgressIgnoreCIDR.NodeIP.IPv6; ipv6 != nil {
		eciNodesIPv6 = ipv6
	}
	if ipv4 := eci.Status.EgressIgnoreCIDR.ClusterIP.IPv4; ipv4 != nil {
		eciClusterIPv4 = ipv4
	}
	if ipv6 := eci.Status.EgressIgnoreCIDR.ClusterIP.IPv6; ipv6 != nil {
		eciClusterIPv6 = ipv6
	}
	if ipv4 := eci.Status.EgressIgnoreCIDR.PodCIDR.IPv4; ipv4 != nil {
		eciPodCidrIPv4 = ipv4
	}
	if ipv6 := eci.Status.EgressIgnoreCIDR.PodCIDR.IPv6; ipv6 != nil {
		eciPodCidrIPv6 = ipv6
	}
	if ignoreNodeIP {
		// get allNodes ip and check
		nodesIPv4, nodesIPv6 := common.GetAllNodesIP(f)

		ok, err := utils.IsSameIPs(eciNodesIPv4, nodesIPv4)
		GinkgoWriter.Printf("eciNodesIPv4: %v, nodesIPv4: %v\n", eciNodesIPv4, nodesIPv4)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		ok, err = utils.IsSameIPs(eciNodesIPv6, nodesIPv6)
		GinkgoWriter.Printf("eciNodesIPv6: %v, nodesIPv6: %v\n", eciNodesIPv6, nodesIPv6)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	}

	if ignoreClusterIP {
		// get service subnet and check
		serviceIpv4s, serviceIpv6s := common.GetClusterIpCidr(f)

		ok, err := utils.IsSameIPCidrs(eciClusterIPv4, serviceIpv4s)
		GinkgoWriter.Printf("eciClusterIPv4: %v, serviceIpv4s: %v\n", eciClusterIPv4, serviceIpv4s)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		ok, err = utils.IsSameIPCidrs(eciClusterIPv6, serviceIpv6s)
		GinkgoWriter.Printf("eciClusterIPv6: %v, serviceIpv6s: %v\n", eciClusterIPv6, serviceIpv6s)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	}

	switch podCidr {
	case common.CALICO:
		// get calico ippool cidr and check
		v4Cidrs, v6Cidrs := common.GetCalicoIPPoolsCidr(f)

		ok, err := utils.IsSameIPCidrs(eciPodCidrIPv4, v4Cidrs)
		GinkgoWriter.Printf("eciPodCidrIPv4: %v, v4Cidrs: %v\n", eciPodCidrIPv4, v4Cidrs)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		ok, err = utils.IsSameIPCidrs(eciPodCidrIPv6, v6Cidrs)
		GinkgoWriter.Printf("eciPodCidrIPv6: %v, v6Cidrs: %v\n", eciPodCidrIPv6, v6Cidrs)
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	}
}

func createOrUpdateCalicoIPPoolAndCheck(f *framework.Framework, poolNamePre string, calicoIPPools *[]string, generateRandomCidr func() string, eci *egressv1beta1.EgressClusterInfo) {
	// get EgressClusterInfo
	GinkgoWriter.Printf("get EgressClusterInfo %s\n", egressClusterInfoName)
	Expect(common.GetEgressClusterInfo(f, egressClusterInfoName, eci)).NotTo(HaveOccurred())

	// CreateCalicoIPPool
	GinkgoWriter.Println("create calico ippool")
	ipPool := common.CreateCalicoIPPool(f, poolNamePre, generateRandomCidr)
	Expect(ipPool).NotTo(BeNil())
	*calicoIPPools = append(*calicoIPPools, ipPool.Name)

	// WaitCalicoIPPoolCreated
	GinkgoWriter.Printf("wait calico ippool %s created\n", ipPool.Name)
	ipPool, err = common.WaitCalicoIPPoolCreated(f, ipPool.Name, time.Second*10)
	Expect(err).NotTo(HaveOccurred())

	// WaitEgressClusterInfoPodCidrUpdated
	GinkgoWriter.Println("WaitEgressClusterInfoPodCidrUpdated after calicoIPPool created")
	eci, err = common.WaitEgressClusterInfoPodCidrUpdated(f, eci, common.CALICO, time.Second*30)
	Expect(err).NotTo(HaveOccurred())
	Expect(eci).NotTo(BeNil())

	// check EgressIgnoreCIDR Fields
	checkEgressIgnoreCIDRFields(eci)

	// update calicoIPPool
	var updatedPool *calicov1.IPPool
	if enableV4 {
		// UpdateCalicoIPPoolCidr
		GinkgoWriter.Println("UpdateCalicoIPPoolCidr v4")
		updatedPool, err = common.UpdateCalicoIPPoolCidr(f, ipPool, common.RandomIPPoolV4Cidr)
		Expect(err).NotTo(HaveOccurred())
	}

	if enableV6 && !enableV4 {
		// UpdateCalicoIPPoolCidr
		GinkgoWriter.Println("UpdateCalicoIPPoolCidr v6")
		updatedPool, err = common.UpdateCalicoIPPoolCidr(f, ipPool, common.RandomIPPoolV6Cidr)
		Expect(err).NotTo(HaveOccurred())
	}

	Expect(updatedPool).NotTo(BeNil())

	// WaitCalicoIPPoolCidrUpdated
	GinkgoWriter.Println("WaitCalicoIPPoolCidrUpdated")
	Expect(common.WaitCalicoIPPoolCidrUpdated(f, updatedPool, time.Second*3)).NotTo(HaveOccurred())

	// WaitEgressClusterInfoPodCidrUpdated
	GinkgoWriter.Println("WaitEgressClusterInfoPodCidrUpdated after calicoIPPool updated")
	eci, err = common.WaitEgressClusterInfoPodCidrUpdated(f, eci, common.CALICO, time.Second*30)
	Expect(err).NotTo(HaveOccurred())
	Expect(eci).NotTo(BeNil())

	// check EgressIgnoreCIDR Fields
	checkEgressIgnoreCIDRFields(eci)

	// DeleteCalicoIPPool
	GinkgoWriter.Printf("delete calico ippool %s\n", ipPool.Name)
	Expect(common.DeleteCalicoIPPool(f, ipPool)).NotTo(HaveOccurred())

	// WaitCalicoIPPoolDeleted
	GinkgoWriter.Printf("wait calico ippool %s deleted\n", ipPool.Name)
	Expect(common.WaitCalicoIPPoolDeleted(f, ipPool.Name, time.Second*10)).NotTo(HaveOccurred())

	// WaitEgressClusterInfoPodCidrUpdated
	GinkgoWriter.Printf("wait egressClusterInfo updated after calico ippool '%s' deleted, prevent affecting other cases\n", ipPool.Name)
	_, err = common.WaitEgressClusterInfoPodCidrUpdated(f, eci, common.CALICO, time.Second*5)
}
