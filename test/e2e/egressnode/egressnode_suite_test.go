// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressnode_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/e2eframework/framework"
	egressgatewayv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestEgressnode(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Egressnode Suite")
}

var (
	f   *framework.Framework
	err error
	c   client.WithWatch
)

var _ = BeforeSuite(func() {
	GinkgoRecover()

	f, err = framework.NewFramework(GinkgoT(), []func(scheme *runtime.Scheme) error{egressgatewayv1.AddToScheme})
	Expect(err).NotTo(HaveOccurred(), "failed to NewFramework, details: %w", err)
	c = f.KClient
})
