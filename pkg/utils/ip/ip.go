// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package ip

import (
	"bytes"
	"fmt"
	"math/big"
	"net"
	"sort"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/spidernet-io/egressgateway/pkg/constant"
)

// ============== IP ==============

// Cmp compares two IP addresses, returns according to the following rules:
// ip1 < ip2: -1
// ip1 = ip2: 0
// ip1 > ip2: 1
func Cmp(ip1, ip2 net.IP) int {
	int1 := ipToInt(ip1)
	int2 := ipToInt(ip2)
	return int1.Cmp(int2)
}

// ipToInt converts net.IP to big.Int.
func ipToInt(ip net.IP) *big.Int {
	if v := ip.To4(); v != nil {
		return big.NewInt(0).SetBytes(v)
	}
	return big.NewInt(0).SetBytes(ip.To16())
}

// intToIP converts big.Int to net.IP.
func intToIP(i *big.Int) net.IP {
	return net.IP(i.Bytes()).To16()
}

// NextIP returns the next IP address.
func NextIP(ip net.IP) net.IP {
	i := ipToInt(ip)
	return intToIP(i.Add(i, big.NewInt(1)))
}

// IsIPVersion reports whether version is a valid IP version (4 or 6).
func IsIPVersion(version constant.IPVersion) error {
	if version != constant.IPv4 && version != constant.IPv6 {
		return fmt.Errorf("%w '%d'", ErrInvalidIPVersion, version)
	}

	return nil
}

func GetIPV4V6(ips []string) (IPV4s, IPV6s []string, err error) {
	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			return nil, nil, ErrInvalidIP
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

func GetIPV4V6Cidr(ipCidrs []string) (IPV4Cidrs, IPV6Cidrs []string, err error) {
	for _, ipCidr := range ipCidrs {
		if strings.Contains(ipCidr, ".") {
			isIPv4Cidr, err := IsIPv4Cidr(ipCidr)
			if err != nil {
				return nil, nil, err
			}
			if isIPv4Cidr {
				IPV4Cidrs = append(IPV4Cidrs, ipCidr)
				continue
			}
		} else {
			isIPv6Cidr, err := IsIPv6Cidr(ipCidr)
			if err != nil {
				return nil, nil, err
			}
			if isIPv6Cidr {
				IPV6Cidrs = append(IPV6Cidrs, ipCidr)
			}
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
	if len(sortedA) != len(sortedB) {
		return false, nil
	}

	for i := range sortedA {
		if sortedA[i] != sortedB[i] {
			return false, nil
		}
	}
	return true, nil
}

func IsSameIPCidrs(a, b []string) (bool, error) {
	if len(a) != len(b) {
		return false, nil
	}
	sortedA, err := SortIPCidrs(a)
	if err != nil {
		return false, err
	}
	sortedB, err := SortIPCidrs(b)
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
		err := fmt.Errorf(constant.InvalidIPFormat)
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
		err := fmt.Errorf(constant.InvalidIPFormat)
		return false, err
	}

	if netIP.To4() == nil {
		return true, nil
	}
	return false, nil
}

func SortIPs(ips []string) ([]string, error) {
	sortedIPs := make([]string, 0)
	for _, ip := range ips {
		netIP := net.ParseIP(ip)
		if netIP == nil {
			err := fmt.Errorf(constant.InvalidIPFormat)
			return nil, err
		}
		sortedIPs = append(sortedIPs, netIP.String())
	}
	sort.Strings(sortedIPs)
	return sortedIPs, nil
}

func SortIPCidrs(ips []string) ([]string, error) {
	ipcidrs := make([]string, 0)
	for _, ip := range ips {
		_, netIP, err := net.ParseCIDR(ip)
		if err != nil {
			return nil, err
		}
		ipcidrs = append(ipcidrs, netIP.String())
	}
	sort.Strings(ipcidrs)
	return ipcidrs, nil
}

func IsIPv4Cidr(cidr string) (bool, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	verified := ipnet.IP.To4() != nil && ipnet.Mask != nil && len(ipnet.Mask) == net.IPv4len
	if !verified {
		return false, nil
	}
	return true, nil
}

func IsIPv6Cidr(cidr string) (bool, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	verified := ipnet.IP.To16() != nil && ipnet.Mask != nil && len(ipnet.Mask) == net.IPv6len
	if !verified {
		return false, nil
	}
	return true, nil
}

// IPsDiffSet calculates the difference set of two IP address slices.
// For example, the difference set between [172.18.40.1 172.18.40.2] and
// [172.18.40.2 172.18.40.3] is [172.18.40.1].
//
// If sorted is true, the result set of IP addresses will be sorted.
func IPsDiffSet(ips1, ips2 []net.IP, sorted bool) []net.IP {
	var ips []net.IP
	marks := make(map[string]bool)
	for _, ip := range ips1 {
		if ip != nil {
			marks[ip.String()] = true
		}
	}

	for _, ip := range ips2 {
		if ip != nil {
			delete(marks, ip.String())
		}
	}

	for k := range marks {
		ips = append(ips, net.ParseIP(k))
	}

	if sorted {
		sort.Slice(ips, func(i, j int) bool {
			return bytes.Compare(ips[i].To16(), ips[j].To16()) < 0
		})
	}

	return ips
}

// ============ IPRange ============

// MergeIPRanges merges dispersed IP ranges.
// For example, transport [172.18.40.1-172.18.40.3, 172.18.40.2-172.18.40.5]
// to [172.18.40.1-172.18.40.5]. The overlapping part of two IP ranges will
// be ignored.
func MergeIPRanges(version constant.IPVersion, ipRanges []string) ([]string, error) {
	ips, err := ParseIPRanges(version, ipRanges)
	if err != nil {
		return nil, err
	}

	return ConvertIPsToIPRanges(version, ips)
}

// ConvertIPsToIPRanges converts the IP address slices of the specified
// IP version into a group of distinct, sorted and merged IP ranges.
func ConvertIPsToIPRanges(version constant.IPVersion, ips []net.IP) ([]string, error) {
	if err := IsIPVersion(version); err != nil {
		return nil, err
	}

	set := map[string]struct{}{}
	for _, ip := range ips {
		if (version == constant.IPv4 && ip.To4() == nil) ||
			(version == constant.IPv6 && ip.To4() != nil) {
			return nil, fmt.Errorf("%wv%d IP '%s'", ErrInvalidIP, version, ip.String())
		}
		set[ip.String()] = struct{}{}
	}

	ips = ips[0:0]
	for v := range set {
		ips = append(ips, net.ParseIP(v))
	}

	sort.Slice(ips, func(i, j int) bool {
		return bytes.Compare(ips[i].To16(), ips[j].To16()) < 0
	})

	var ipRanges []string
	var start, end int
	for {
		if start == len(ips) {
			break
		}

		if end+1 < len(ips) && ips[end+1].Equal(NextIP(ips[end])) {
			end++
			continue
		}

		if start == end {
			ipRanges = append(ipRanges, ips[start].String())
		} else {
			ipRanges = append(ipRanges, fmt.Sprintf("%s-%s", ips[start], ips[end]))
		}

		start = end + 1
		end = start
	}

	return ipRanges, nil
}

// IsIPIncludedRange determines whether an IP address is included in the destination range
func IsIPIncludedRange(version constant.IPVersion, ip string, ipRange []string) (bool, error) {
	ips, err := MergeIPRanges(version, ipRange)
	if err != nil {
		return false, err
	}

	var result bool

	for _, item := range ips {
		result, err := IsIPRangeOverlap(version, ip, item)
		if err != nil {
			return false, err
		}

		if result {
			return true, nil
		}

	}

	return result, nil
}

// IsIPRangeOverlap reports whether the IP address slices of specific IP
// version parsed from two IP ranges overlap.
func IsIPRangeOverlap(version constant.IPVersion, ipRange1, ipRange2 string) (bool, error) {
	if err := IsIPVersion(version); err != nil {
		return false, err
	}
	if err := IsIPRange(version, ipRange1); err != nil {
		return false, err
	}
	if err := IsIPRange(version, ipRange2); err != nil {
		return false, err
	}

	// Ignore the error returned here. The format of the IP range has been
	// verified in IsIPRange above.
	ips1, _ := ParseIPRange(version, ipRange1)
	ips2, _ := ParseIPRange(version, ipRange2)
	if len(ips1) > len(IPsDiffSet(ips1, ips2, false)) {
		return true, nil
	}

	return false, nil
}

// ParseIPRange parses IP range as an IP address slices of the specified
// IP version.
func ParseIPRange(version constant.IPVersion, ipRange string) ([]net.IP, error) {
	if err := IsIPRange(version, ipRange); err != nil {
		return nil, err
	}

	arr := strings.Split(ipRange, "-")
	n := len(arr)
	var ips []net.IP
	if n == 1 {
		ips = append(ips, net.ParseIP(arr[0]))
	}

	if n == 2 {
		cur := net.ParseIP(arr[0])
		end := net.ParseIP(arr[1])
		for Cmp(cur, end) <= 0 {
			ips = append(ips, cur)
			cur = NextIP(cur)
		}
	}

	return ips, nil
}

// ParseIPRanges parses IP ranges as a IP address slices of the specified
// IP version.
func ParseIPRanges(version constant.IPVersion, ipRanges []string) ([]net.IP, error) {
	var sum []net.IP
	for _, r := range ipRanges {
		ips, err := ParseIPRange(version, r)
		if err != nil {
			return nil, err
		}
		sum = append(sum, ips...)
	}

	return sum, nil
}

// IsIPRange reports whether ipRange string is a valid IP range. An IP
// range can be a single IP address in the style of '172.18.40.0', or
// an address range in the form of '172.18.40.0-172.18.40.10'.
// The following formats are invalid:
// "172.18.40.0 - 172.18.40.10": there can be no space between two IP
// addresses.
// "172.18.40.1-2001:db8:a0b:12f0::1": invalid combination of IPv4 and
// IPv6.
// "172.18.40.10-172.18.40.1": the IP range must be ordered.
func IsIPRange(version constant.IPVersion, ipRange string) error {
	if err := IsIPVersion(version); err != nil {
		return err
	}

	if (version == constant.IPv4 && !IsIPv4IPRange(ipRange)) ||
		(version == constant.IPv6 && !IsIPv6IPRange(ipRange)) {
		return fmt.Errorf("%w in IPv%d '%s'", ErrInvalidIPRangeFormat, version, ipRange)
	}

	return nil
}

// IsIPv4IPRange reports whether ipRange string is a valid IPv4 range.
// See IsIPRange for more description of IP range.
func IsIPv4IPRange(ipRange string) bool {
	ips := strings.Split(ipRange, "-")
	n := len(ips)
	if n > 2 {
		return false
	}

	if n == 1 {
		return govalidator.IsIPv4(ips[0])
	}

	if n == 2 {
		if !govalidator.IsIPv4(ips[0]) || !govalidator.IsIPv4(ips[1]) {
			return false
		}
		if Cmp(net.ParseIP(ips[0]), net.ParseIP(ips[1])) == 1 {
			return false
		}
	}

	return true
}

// IsIPv6IPRange reports whether ipRange string is a valid IPv6 range.
// See IsIPRange for more description of IP range.
func IsIPv6IPRange(ipRange string) bool {
	ips := strings.Split(ipRange, "-")
	n := len(ips)
	if n > 2 {
		return false
	}

	if n == 1 {
		return govalidator.IsIPv6(ips[0])
	}

	if n == 2 {
		if !govalidator.IsIPv6(ips[0]) || !govalidator.IsIPv6(ips[1]) {
			return false
		}
		if Cmp(net.ParseIP(ips[0]), net.ParseIP(ips[1])) == 1 {
			return false
		}
	}

	return true
}
