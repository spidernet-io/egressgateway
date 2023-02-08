// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package vxlan

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

type Parent struct {
	Name  string
	IP    net.IP
	Index int
}

var defaultInterfacesToExclude = []string{
	"^docker.*", "^cbr.*", "^dummy.*",
	"^virbr.*", "^lxcbr.*", "^veth.*", "^lo",
	"^cali.*", "^tunl.*", "^flannel.*", "^kube-ipvs.*", "^cni.*",
	"^vxlan.calico.*", "^vxlan-v6.calico.*", "vxlan",
}

func GetParent(version int) (*Parent, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var excludeRegexp *regexp.Regexp
	if len(defaultInterfacesToExclude) > 0 {
		expr := "(" + strings.Join(defaultInterfacesToExclude, ")|(") + ")"
		excludeRegexp, err = regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
	}

	filtered := make([]net.Interface, 0)
	for _, iface := range ifaces {
		exclude := excludeRegexp.MatchString(iface.Name)
		if !exclude {
			filtered = append(filtered, iface)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("interface filtered list is empty")
	}

	var tmpIP net.IP
	var name string
	var index int

	for _, iface := range filtered {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		name = iface.Name
		index = iface.Index
		for _, addr := range addrs {
			str := addr.String()
			ip, _, err := net.ParseCIDR(str)
			if err != nil {
				return nil, err
			}

			if !ip.IsGlobalUnicast() {
				continue
			}

			if version == 4 {
				if ip.To4() != nil {
					tmpIP = ip
					break
				}
			}
			if version == 6 {
				if ip.To16() != nil {
					tmpIP = ip
					break
				}
			}
		}
	}

	res := &Parent{
		Name:  name,
		IP:    tmpIP,
		Index: index,
	}
	return res, nil
}
