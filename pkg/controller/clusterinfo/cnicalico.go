// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package clusterinfo

import (
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"
)

func getCalicoIPPoolList(pool *calicov1.IPPool) (ipv4, ipv6 []string) {
	if pool == nil {
		return
	}
	if isV4, err := ip.IsIPv4Cidr(pool.Spec.CIDR); err == nil && isV4 {
		ipv4 = append(ipv4, pool.Spec.CIDR)
	}
	if isV6, err := ip.IsIPv6Cidr(pool.Spec.CIDR); err == nil && isV6 {
		ipv6 = append(ipv6, pool.Spec.CIDR)
	}
	return
}
