// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spidernet-io/egressgateway/pkg/utils"
)

var _ = Describe("Ip", func() {
	Describe("UT GetIPV4V6",Label("GetIPV4V6"), func() {
		ipv4:="10.10.0.1"
		ipv6:="fddd:10::1"
		invalidIPv4:="10.10.1"

		ipv4s:=[]string{ipv4}
		ipv6s:=[]string{ipv6}
		ips:=[]string{ipv4,ipv6}

		invalidIPs:=[]string{invalidIPv4,ipv6}

		It("UT GetIPV4V6, expect success", func() {
			v4, v6, err := utils.GetIPV4V6(ips)
			Expect(err).NotTo(HaveOccurred())
			Expect(v4).To(Equal(ipv4s))
			Expect(v6).To(Equal(ipv6s))
		})

		It("UT GetIPV4V6, invalid ip format", func() {
			v4, v6, err := utils.GetIPV4V6(invalidIPs)
			Expect(err).To(HaveOccurred())
			Expect(v4).To(BeNil())
			Expect(v6).To(BeNil())
		})
	})

})
