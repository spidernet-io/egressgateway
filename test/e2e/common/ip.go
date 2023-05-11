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

func RandomIPPoolV4Cidr() string {
	n1 := tools.GenerateRandomNumber(200)
	n2 := tools.GenerateRandomNumber(255)
	n3 := tools.GenerateRandomNumber(255)
	return fmt.Sprintf("%s.%s.%s.0/24", n1, n2, n3)
}

func RandomIPPoolV6Cidr() string {
	n1 := tools.GenerateString(4, true)
	n2 := tools.GenerateString(4, true)
	return fmt.Sprintf("%s:%s::0/112", n1, n2)
}
