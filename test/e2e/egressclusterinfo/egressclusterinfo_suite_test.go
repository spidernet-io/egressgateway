// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo_test

import (
	"github.com/spidernet-io/egressgateway/test/e2e/common"
	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spidernet-io/e2eframework/framework"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressgatewayv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
)

func TestEgressclusterinfo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Egressclusterinfo Suite")
}

const egressClusterInfoName = "default"

var (
	f                             *framework.Framework
	err                           error
	c                             client.WithWatch
	allNodes                      []string
	ignoreClusterIP, ignoreNodeIP bool
	podCidr                       string
	enableV4, enableV6            bool
)

var _ = BeforeSuite(func() {
	GinkgoRecover()

	f, err = framework.NewFramework(GinkgoT(), []func(scheme *runtime.Scheme) error{egressgatewayv1beta1.AddToScheme, calicov1.AddToScheme})
	Expect(err).NotTo(HaveOccurred(), "failed to NewFramework, details: %w", err)
	c = f.KClient

	// allNode
	allNodes = f.Info.KindNodeList
	Expect(allNodes).NotTo(BeEmpty())

	// get GetEgressIgnoreCIDR
	ignoreCidr, err := common.GetEgressIgnoreCIDR(f)
	Expect(err).NotTo(HaveOccurred())
	Expect(ignoreCidr).NotTo(BeNil())
	ignoreClusterIP = ignoreCidr.ClusterIP
	ignoreNodeIP = ignoreCidr.NodeIP
	podCidr = ignoreCidr.PodCIDR

	// get egressgatewayconfigmap ipversion
	enableV4, enableV6, err = common.GetIPVersion(f)
	Expect(err).NotTo(HaveOccurred())
})
