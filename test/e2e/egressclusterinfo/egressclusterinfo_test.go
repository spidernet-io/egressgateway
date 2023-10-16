// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-faker/faker/v4"
	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/common"
)

var _ = Describe("EgressClusterInfo", Label("EgressClusterInfo"), Serial, func() {
	var ctx = context.Background()
	var eci *egressv1.EgressClusterInfo

	var calicoIPv4Prefix string
	var calicoIPv6Prefix string

	BeforeEach(func() {
		eci = new(egressv1.EgressClusterInfo)
		eci.Name = "default"
		key := types.NamespacedName{Name: eci.Name}
		err := cli.Get(ctx, key, eci)
		Expect(err).NotTo(HaveOccurred())

		calicoIPv4Prefix = "e2e-v4-" + faker.Word()
		calicoIPv6Prefix = "e2e-v6-" + faker.Word()

		DeferCleanup(func() {
			ctx := context.Background()

			list := &calicov1.IPPoolList{}
			err := cli.List(ctx, list)
			Expect(err).NotTo(HaveOccurred())

			for _, p := range list.Items {
				if strings.Contains(p.Name, calicoIPv4Prefix) ||
					strings.Contains(p.Name, calicoIPv6Prefix) {
					err := common.DeleteObj(ctx, cli, &p)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})
	})

	It("Get and delete EgressClusterInfo", Serial, Label("I00001", "I00007"), func() {
		ctx := context.Background()

		GinkgoWriter.Printf("check eci status synced: %s\n", eci.Name)
		err := common.CheckEgressClusterInfoStatusSynced(ctx, cli, eci)
		Expect(err).NotTo(HaveOccurred())

		GinkgoWriter.Printf("delete EgressClusterInfo, we expect it will be failed")
		err = common.DeleteObj(ctx, cli, eci)

		Expect(err).To(HaveOccurred())
	})

	It("Create or update calico IPPool", Serial, Label("I00006"), func() {
		if egressConfig.EnableIPv6 {
			createOrUpdateCalicoIPPoolAndCheck(
				ctx, cli, eci, calicoIPv6Prefix, "112", common.RandomIPPoolV6Cidr)
		}
		if egressConfig.EnableIPv4 {
			createOrUpdateCalicoIPPoolAndCheck(
				ctx, cli, eci, calicoIPv4Prefix, "24", common.RandomIPPoolV4Cidr)
		}
	})

	DescribeTable("Edit EgressClusterInfo spec", Serial, func(f func()) {
		f()
		err := common.UpdateEgressClusterInfoNew(ctx, cli, eci)
		Expect(err).NotTo(HaveOccurred())
	},
		Entry("Update spec.AutoDetect.ClusterIP to be false", Label("I00002"), func() {
			eci.Spec.AutoDetect.ClusterIP = false
		}),
		Entry("Update spec.AutoDetect.NodeIP to be false", Label("I00003"), func() {
			eci.Spec.AutoDetect.NodeIP = false
		}),
		Entry("Update spec.AutoDetect.PodCidrMode to k8s", Label("I00004"), func() {
			eci.Spec.AutoDetect.PodCidrMode = common.K8s
		}),
		Entry("Update spec.ExtraCIDR", Label("I00005"), func() {
			eci.Spec.ExtraCidr = []string{"10.10.10.1"}
		}),
	)
})

func createOrUpdateCalicoIPPoolAndCheck(
	ctx context.Context, cli client.Client,
	eci *egressv1.EgressClusterInfo,
	namePrefix, cidrPrefix string,
	genRandomCIDR func(string) string) {

	eci.Spec.AutoDetect.PodCidrMode = common.Calico
	err := common.UpdateEgressClusterInfoNew(ctx, cli, eci)
	Expect(err).NotTo(HaveOccurred())

	GinkgoWriter.Println("create calico IPPool")
	pool, err := common.CreateCalicoIPPool(ctx, cli, namePrefix, cidrPrefix, genRandomCIDR)
	Expect(err).NotTo(HaveOccurred())

	err = common.CheckEgressClusterInfoStatusSynced(ctx, cli, eci)
	Expect(err).NotTo(HaveOccurred())

	if egressConfig.EnableIPv4 {
		GinkgoWriter.Println("update calico IPPool CIDR v4")
		err = common.UpdateCalicoIPPoolCIDR(ctx, cli, pool, "24", common.RandomIPPoolV4Cidr)
		Expect(err).NotTo(HaveOccurred())
	}

	if egressConfig.EnableIPv6 && !egressConfig.EnableIPv4 {
		GinkgoWriter.Println("update calico IPPool CIDR v6")
		err = common.UpdateCalicoIPPoolCIDR(ctx, cli, pool, "112", common.RandomIPPoolV6Cidr)

		Expect(err).NotTo(HaveOccurred())
	}

	err = common.CheckEgressClusterInfoStatusSynced(ctx, cli, eci)
	Expect(err).NotTo(HaveOccurred())

	// delete Calico IPPool
	GinkgoWriter.Printf("delete calico IPPool %s\n", pool.Name)
	err = common.DeleteObj(ctx, cli, pool)
	Expect(err).NotTo(HaveOccurred())

	// wait Calico IPPool Deleted
	//

	// check status
	err = common.CheckEgressClusterInfoStatusSynced(ctx, cli, eci)
	Expect(err).NotTo(HaveOccurred())
}
