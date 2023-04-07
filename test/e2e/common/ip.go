// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

// CheckEgressNodeIP check if egressnode.status vxlanIp is the same with egress.vxlan ip of the kindNode
func CheckEgressNodeIP(nodeName string, ip string, duration time.Duration) bool {
	command := fmt.Sprintf("ip a show %s | grep %s", EGRESS_VXLAN_INTERFACE_NAME, ip)
	if _, err := tools.ExecInKindNode(nodeName, command, duration); err != nil {
		return false
	}
	return true
}

// CheckEgressNodeMac check if egressnode.status vxlanMac is the same with egress.vxlan mac of the kindNode
func CheckEgressNodeMac(nodeName string, mac string, duration time.Duration) bool {
	command := fmt.Sprintf("ip l show %s | grep %s", EGRESS_VXLAN_INTERFACE_NAME, mac)
	if _, err := tools.ExecInKindNode(nodeName, command, duration); err != nil {
		return false
	}
	return true
}

// CheckEgressNodeInterface check if egressnode.status interfaceName is the same with the interfaceName( the default route been set on it ) of the kindNode
func CheckEgressNodeInterface(nodeName string, nic string, duration time.Duration) bool {
	command := fmt.Sprintf("ip r l default | grep %s", nic)
	if _, err := tools.ExecInKindNode(nodeName, command, duration); err != nil {
		return false
	}
	return true
}

// CheckNodeIP check if egressnode.status kindNodeIp is the same with the ip of the kindNode
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

// GetKindNodeDefaultInterface get the interfaceName( the default route been set on it ) of the kindNode
func GetKindNodeDefaultInterface(nodeName string, duration time.Duration) (string, error) {
	command := "ip r | grep default | awk '{print $5}'"
	b, err := tools.ExecInKindNode(nodeName, command, duration)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetKindNodeDefaultIPV4 get the ipv4 of the kindNode
func GetKindNodeDefaultIPV4(nodeName string, duration time.Duration) (string, error) {
	command := "ip -4 -br a show `ip r | grep default | awk '{print $5}'` | grep -E -o '[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+' | tr -d '\n' "
	b, err := tools.ExecInKindNode(nodeName, command, duration)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetKindNodeDefaultIPV6 get the ipv6 of the kindNode
func GetKindNodeDefaultIPV6(nodeName string, duration time.Duration) (string, error) {
	command := "ip -6 -br a show `ip r | grep default | awk '{print $5}'` | awk '{print $3}' | awk -F / '{print $1}' | tr -d '\n' "
	b, err := tools.ExecInKindNode(nodeName, command, duration)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetKindNodeDefaultIPV4Net get the ipv4( cidr notation ) of the kindNode
func GetKindNodeDefaultIPV4Net(nodeName string, duration time.Duration) (string, error) {
	command := "ip -4 -br a show `ip r | grep default | awk '{print $5}'` | awk '{print $3}' | tr -d '\n' "
	b, err := tools.ExecInKindNode(nodeName, command, duration)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetKindNodeDefaultIPV6Net get the ipv6( cidr notation ) of the kindNode
func GetKindNodeDefaultIPV6Net(nodeName string, duration time.Duration) (string, error) {
	command := "ip -6 -br a show `ip r | grep default | awk '{print $5}'` | awk '{print $3}' | tr -d '\n' "
	b, err := tools.ExecInKindNode(nodeName, command, duration)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetKindNodeIPV4ByGivenInterface get the ipv4 of the given interface
func GetKindNodeIPV4ByGivenInterface(nodeName, interfaceName string, duration time.Duration) (string, error) {
	command := fmt.Sprintf("ip -4 -br a show %s | grep -E -o '[0-9]+\\.[0-9]+\\.[0-9]+\\.[0-9]+' | tr -d '\n' ", interfaceName)
	b, err := tools.ExecInKindNode(nodeName, command, duration)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// GetKindNodeIPV6ByGivenInterface get the ipv6 of the given interface
func GetKindNodeIPV6ByGivenInterface(nodeName, interfaceName string, duration time.Duration) (string, error) {
	command := fmt.Sprintf("ip -6 -br a show %s | awk '{print $3}' | awk -F / '{print $1}' | tr -d '\n' ", interfaceName)
	b, err := tools.ExecInKindNode(nodeName, command, duration)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SetKindNodeInterface set the oldInterfaceName to newInterfaceName of the kindNode
func SetKindNodeInterface(nodeName, oldInterfaceName, newInterfaceName string, duration time.Duration) (string, error) {
	command := fmt.Sprintf("ip l set dev %s down && ip l set dev %s name %s && ip l set dev %s up", oldInterfaceName, oldInterfaceName, newInterfaceName, newInterfaceName)
	b, err := tools.ExecInKindNode(nodeName, command, duration)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SetKindNodeMacAddrByGivenInterface set the macAddr of the given interfaceName of the kindNode
func SetKindNodeMacAddrByGivenInterface(nodeName, interfaceName, macAddr string, duration time.Duration) (string, error) {
	command := fmt.Sprintf("ip link set dev %s addr %s", interfaceName, macAddr)
	b, err := tools.ExecInKindNode(nodeName, command, duration)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
