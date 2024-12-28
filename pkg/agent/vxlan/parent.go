// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package vxlan

import (
	"context"
	"fmt"
	"net"
	"os"

	corev1 "k8s.io/api/core/v1"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/vishvananda/netlink"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
			ones, bits := addr.Mask.Size()
			if ones == 32 && bits == 32 || ones == 128 && bits == 128 {
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

// GetCustomParentName get parent interface name from configured TunnelDetectCustomInterface
// If no node matches from TunnelDetectCustomInterface, return default name from configured TunnelDetectMethod with interface=...
func GetCustomParentName(cl client.Client, defaultName string, override []config.TunnelDetectCustomInterface) (string, error) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return "", fmt.Errorf("NODE_NAME environment variable is not set")
	}
	// Retrieve the current node's labels
	node := &corev1.Node{}
	if err := cl.Get(context.Background(), client.ObjectKey{Name: nodeName}, node); err != nil {
		return "", fmt.Errorf("can not get current node: %s", err)
	}
	// ...
	for _, ovrd := range override {
		if matchesNodeSelector(node, ovrd.NodeSelector) {
			return ovrd.InterfaceName, nil
		}
	}
	return defaultName, nil
}

// matchesNodeSelector checks if a node's labels match the given nodeSelector
func matchesNodeSelector(node *corev1.Node, nodeSelector map[string]string) bool {
	for key, value := range nodeSelector {
		if node.Labels[key] != value {
			return false
		}
	}
	return true
}
