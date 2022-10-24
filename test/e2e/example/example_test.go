// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0
package example_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("example ", Label("example"), func() {

	It("example", Label("example-1"), func() {
		GinkgoWriter.Printf("cluster information: %+v \n", frame.Info)

		d, e := frame.GetDeployment("test", "default")
		Expect(e).NotTo(HaveOccurred(), "failed to get deployment ")
		GinkgoWriter.Printf("deployment: %+v \n", d)
	})
})
