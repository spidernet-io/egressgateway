// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"net"
	"sort"
)

func GetIPV4V6(ips []string) (IPV4s, IPV6s []string, err error) {
	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			err = fmt.Errorf("invalid ip format")
			return nil, nil, err
		}
		if isIPv4, _ := IsIPv4(ip); isIPv4 {
			IPV4s = append(IPV4s, ip)
		}
		if isIPv6, _ := IsIPv6(ip); isIPv6 {
			IPV6s = append(IPV6s, ip)
		}
	}
	return
}

func IsSameIPs(a, b []string) (bool, error) {
	if len(a) != len(b) {
		return false, nil
	}
	sortedA, err := SortIPs(a)
	if err != nil {
		return false, err
	}
	sortedB, err := SortIPs(b)
	if err != nil {
		return false, err
	}

	for i := range sortedA {
		if sortedA[i] != sortedB[i] {
			return false, nil
		}
	}
	return true, nil
}

func IsIPv4(ip string) (bool, error) {
	netIP := net.ParseIP(ip)
	if netIP == nil {
		err := fmt.Errorf("invalid ip format")
		return false, err
	}

	if netIP.To4() == nil {
		return false, nil
	}
	return true, nil
}

func IsIPv6(ip string) (bool, error) {
	netIP := net.ParseIP(ip)
	if netIP == nil {
		err := fmt.Errorf("invalid ip format")
		return false, err
	}

	if netIP.To4() == nil {
		return true, nil
	}
	return false, nil
}

func SortIPs(ips []string) ([]string, error) {
	sortedIPs := make([]string, len(ips))
	for _, ip := range ips {
		netIP := net.ParseIP(ip)
		if netIP == nil {
			err := fmt.Errorf("invalid ip format")
			return nil, err
		}
		sortedIPs = append(sortedIPs, netIP.String())
	}
	sort.Strings(sortedIPs)
	return sortedIPs, nil
}

func IsIPv4Cidr(cidr string) (bool, error) {
	netIP, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	if netIP.To4() != nil {
		return true, nil
	}
	return false, nil
}

func IsIPv6Cidr(cidr string) (bool, error) {
	netIP, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	if netIP.To4() == nil {
		return true, nil
	}
	return false, nil
}
