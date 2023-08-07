// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"regexp"
	"strings"

	. "github.com/onsi/gomega"
	"github.com/spidernet-io/e2eframework/framework"
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
)

func GetClusterIpCidr(f *framework.Framework) (ipv4s, ipv6s []string) {
	configMap, err := f.GetConfigmap(kubeadmConfig, kubeSystem)
	Expect(err).NotTo(HaveOccurred())
	Expect(configMap).NotTo(BeNil())
	v, ok := configMap.Data[clusterConfiguration]
	Expect(ok).To(BeTrue())
	Expect(v).NotTo(BeEmpty())

	reg := regexp.MustCompile(serviceSubnet + `: (.*)`)
	svcSubnetKV := reg.FindStringSubmatch(v)
	Expect(svcSubnetKV).NotTo(BeNil())
	Expect(len(svcSubnetKV)).To(Equal(2))
	subnets := strings.Split(svcSubnetKV[1], ",")

	ipv4s, ipv6s, err = ip.GetIPV4V6Cidr(subnets)
	Expect(err).NotTo(HaveOccurred())
	return ipv4s, ipv6s
}
