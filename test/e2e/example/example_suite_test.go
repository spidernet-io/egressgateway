// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package example_test

import (
	egressgatewayv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/e2eframework/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAssignIP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "example Suite")
}

var (
	frame *framework.Framework
	err   error
	c     client.WithWatch
)

var _ = BeforeSuite(func() {
	GinkgoRecover()

	frame, err = framework.NewFramework(GinkgoT(), []func(scheme *runtime.Scheme) error{egressgatewayv1.AddToScheme})
	Expect(err).NotTo(HaveOccurred(), "failed to NewFramework, details: %w", err)
	c = frame.KClient
})
