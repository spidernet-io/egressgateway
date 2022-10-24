// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0
package example_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	e2e "github.com/spidernet-io/e2eframework/framework"
	// "k8s.io/apimachinery/pkg/runtime"
)

func TestAssignIP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "example Suite")
}

var frame *e2e.Framework

var _ = BeforeSuite(func() {
	defer GinkgoRecover()
	var e error
	frame, e = e2e.NewFramework(GinkgoT(), nil)
	Expect(e).NotTo(HaveOccurred())

})
