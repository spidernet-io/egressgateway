// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"

	"math/big"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/spidernet-io/egressgateway/pkg/constant"
	iputil "github.com/spidernet-io/egressgateway/pkg/utils/ip"
	e "github.com/spidernet-io/egressgateway/test/e2e/err"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

func CheckEgressNodeIP(nodeName string, ip string, duration time.Duration) bool {
	command := fmt.Sprintf("ip a show %s | grep %s", EGRESS_VXLAN_INTERFACE_NAME, ip)
	if _, err := tools.ExecInKindNode(nodeName, command, duration); err != nil {
		return false
	}
	return true
}

func CheckEgressNodeMac(nodeName string, mac string, duration time.Duration) bool {
	command := fmt.Sprintf("ip l show %s | grep %s", EGRESS_VXLAN_INTERFACE_NAME, mac)
	if _, err := tools.ExecInKindNode(nodeName, command, duration); err != nil {
		return false
	}
	return true
}

func CheckEgressNodeInterface(nodeName string, nic string, duration time.Duration) bool {
	command := fmt.Sprintf("ip r l default | grep %s", nic)
	if _, err := tools.ExecInKindNode(nodeName, command, duration); err != nil {
		return false
	}
	return true
}

func CheckNodeIP(nodeName string, nic, ip string, duration time.Duration) bool {
	command := fmt.Sprintf("ip a show %s | grep %s", nic, ip)
	if _, err := tools.ExecInKindNode(nodeName, command, duration); err != nil {
		return false
	}
	return true
}

func GetHostIPV4(duration time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	a := "ip -4 -br a show `ip r | grep default | awk '{print $5}'` | grep -E -o '[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+' | tr -d '\n' "
	return exec.CommandContext(ctx, "sh", "-c", a).Output()
}

func GetHostIPV6(duration time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	a := "ip -6 -br a show `ip r | grep default | awk '{print $5}'` | awk '{print $3}' | awk -F / '{print $1}' | tr -d '\n' "
	return exec.CommandContext(ctx, "sh", "-c", a).Output()
}

func GetHostIPV4Net(duration time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	a := "ip -4 -br a show `ip r | grep default | awk '{print $5}'` | awk '{print $3}' | tr -d '\n' "
	return exec.CommandContext(ctx, "sh", "-c", a).Output()
}

func GetHostIPV6Net(duration time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	a := "ip -6 -br a show `ip r | grep default | awk '{print $5}'` | awk '{print $3}' | tr -d '\n' "
	return exec.CommandContext(ctx, "sh", "-c", a).Output()
}

func RandomIPPoolV4Cidr(prefix string) string {
	n1 := tools.GenerateRandomNumber(255)
	n2 := tools.GenerateRandomNumber(255)
	return fmt.Sprintf("10.%s.%s.0/%s", n1, n2, prefix)
}

func RandomIPPoolV6Cidr(prefix string) string {
	n1 := tools.GenerateString(4, true)
	n2 := tools.GenerateString(4, true)
	return fmt.Sprintf("fddd:%s:%s::0/%s", n1, n2, prefix)
}

func RandomIPV4() string {
	n1 := tools.GenerateRandomNumber(200)
	n2 := tools.GenerateRandomNumber(255)
	n3 := tools.GenerateRandomNumber(255)
	return fmt.Sprintf("10.%s.%s.%s", n1, n2, n3)
}

func RandomIPV6() string {
	n1 := tools.GenerateString(4, true)
	n2 := tools.GenerateString(4, true)
	return fmt.Sprintf("fddd:%s::%s", n1, n2)
}

func RandomIPPoolV4Range(start, end string) string {
	n1 := tools.GenerateRandomNumber(255)
	n2 := tools.GenerateRandomNumber(255)
	return fmt.Sprintf("10.%s.%s.%s-10.%s.%s.%s", n1, n2, start, n1, n2, end)
}

func RandomIPPoolV6Range(start, end string) string {
	n1 := tools.GenerateString(4, true)
	n2 := tools.GenerateString(4, true)
	return fmt.Sprintf("fddd:%s:%s::%s-fddd:%s:%s::%s", n1, n2, start, n1, n2, end)
}

// CheckIPinCidr check if "cidr" contains "ip"
func CheckIPinCidr(ip, cidr string) (bool, error) {
	IPip := net.ParseIP(ip)
	if IPip == nil {
		return false, e.IPVERSION_ERR
	}
	IPip2, IPnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	if IPip2.String() == strings.Split(IPnet.String(), "/")[0] {
		return true, nil
	}
	return IPnet.Contains(IPip), nil
}

// CheckIPIncluded check if "ips" contains "ip", the "ips" can contains 3 formats: single IP, IP range, IP cidr, like ["172.30.0.2", "172.30.0.3-172.30.0-5", 172.30.1.0/24]
func CheckIPIncluded(version constant.IPVersion, ip string, ips []string) (bool, error) {
	for _, item := range ips {
		_, _, err := net.ParseCIDR(item)
		if err != nil {
			// item is not cidr
			include, err := iputil.IsIPIncludedRange(version, ip, []string{item})
			if err != nil {
				return false, err
			}
			if include {
				return true, nil
			}
		}
		// item is cidr
		include, err := CheckIPinCidr(ip, item)
		if err != nil {
			return false, err
		}
		if include {
			return true, nil
		}
	}
	return false, nil
}

// CheckIPSlice check if slice is valid ip format, it supports 3 formats: single IP, IP range, IP cidr, like ["172.30.0.2", "172.30.0.3-172.30.0-5", 172.30.1.0/24]
func CheckIPSlice(ipSlice []string) error {
	for _, s := range ipSlice {
		i := 0
		_, _, err := net.ParseCIDR(s)
		if err != nil {
			if iputil.IsIPv4IPRange(s) {
				i++
			}
			if iputil.IsIPv6IPRange(s) {
				i++
			}
			if i == 0 {
				return ERR_IP_FORMAT
			}
		}
	}
	return nil
}

// IpToInt converts net.IP to big.Int.
func IpToInt(ip net.IP) *big.Int {
	if v := ip.To4(); v != nil {
		return big.NewInt(0).SetBytes(v)
	}
	return big.NewInt(0).SetBytes(ip.To16())
}

// IntToIP converts big.Int to net.IP.
func IntToIP(i *big.Int) net.IP {
	return net.IP(i.Bytes()).To16()
}

// AddIP returns the xth IP starting from "ip"
func AddIP(ip string, x int64) (string, error) {
	netIp := net.ParseIP(ip)
	if netIp == nil {
		return "", ERR_IP_FORMAT
	}
	intIp := IpToInt(netIp)
	intIp.Add(intIp, big.NewInt(x))
	return IntToIP(intIp).String(), nil
}
