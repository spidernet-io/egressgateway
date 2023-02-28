// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package vxlan

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

type NetLink struct {
	RouteListFiltered func(family int, filter *netlink.Route, filterMask uint64) ([]netlink.Route, error)
	LinkByIndex       func(index int) (netlink.Link, error)
	AddrList          func(link netlink.Link, family int) ([]netlink.Addr, error)
	LinkByName        func(name string) (netlink.Link, error)
}

// Parent defines the parent interface information
type Parent struct {
	Name  string
	IP    net.IP
	Index int
}

// GetParentByDefaultRoute get vxlan parent interface by default route
func GetParentByDefaultRoute(cli NetLink) func(version int) (*Parent, error) {
	return func(version int) (*Parent, error) {
		routeFilter := &netlink.Route{Table: 254}
		filterMask := netlink.RT_FILTER_TABLE
		family := netlink.FAMILY_V4
		if version == 6 {
			family = netlink.FAMILY_V6
		}

		routes, err := cli.RouteListFiltered(family, routeFilter, filterMask)
		if err != nil {
			return nil, fmt.Errorf("failed to list routes: %v", err)
		}

		index := -1
		for _, route := range routes {
			if route.Family == family {
				index = route.LinkIndex
				break
			}
		}
		if index == -1 {
			return nil, fmt.Errorf("not found default route link: family IPv%v", version)
		}

		link, err := cli.LinkByIndex(index)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent link by index: %v, %v", index, err)
		}
		addrs, err := cli.AddrList(link, family)
		if err != nil {
			return nil, fmt.Errorf("failed to list parent link addrs: %v", err)
		}
		for _, addr := range addrs {
			if !addr.IP.IsGlobalUnicast() {
				continue
			}
			return &Parent{Name: link.Attrs().Name, IP: addr.IP, Index: link.Attrs().Index}, nil
		}
		return nil, fmt.Errorf("failed to find parent interface")
	}
}

func GetParentByName(cli NetLink, name string) func(version int) (*Parent, error) {
	return func(version int) (*Parent, error) {
		link, err := cli.LinkByName(name)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent link by name: %v, %v", name, err)
		}
		family := netlink.FAMILY_V4
		if version == 6 {
			family = netlink.FAMILY_V6
		}
		addrs, err := cli.AddrList(link, family)
		if err != nil {
			return nil, fmt.Errorf("failed to list parent link addrs: %v", err)
		}
		for _, addr := range addrs {
			if !addr.IP.IsGlobalUnicast() {
				continue
			}
			return &Parent{Name: link.Attrs().Name, IP: addr.IP, Index: link.Attrs().Index}, nil
		}
		return nil, fmt.Errorf("failed to find parent interface")
	}
}
