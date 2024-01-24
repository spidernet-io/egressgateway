// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package vxlan

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
)

type TestCase struct {
	NetLink   NetLink
	Version   int
	expErr    bool
	expParent *Parent
}

func TestGetParentByDefaultRoute(t *testing.T) {
	cases := map[string]TestCase{
		"caseIPv4":           caseIPv4(),
		"caseIPv6":           caseIPv6(),
		"caseListRouteErr":   caseListRouteErr(),
		"caseListRouteEmpty": caseListRouteEmpty(),
	}
	for name, item := range cases {
		t.Run(name, func(t *testing.T) {
			getParent := GetParentByDefaultRoute(item.NetLink)
			parent, err := getParent(item.Version)
			if err != nil {
				if item.expErr {
					return
				}
				assert.NoError(t, err)
			}
			assert.Equal(t, item.expParent, parent)
		})
	}
}

func caseIPv4() TestCase {
	ip := net.ParseIP("10.6.0.1")
	multicastIP := net.ParseIP("224.0.0.0")
	return TestCase{
		NetLink: NetLink{
			RouteListFiltered: func(family int, filter *netlink.Route, filterMask uint64) ([]netlink.Route, error) {
				return []netlink.Route{{
					LinkIndex: 10,
					Dst:       nil,
					Family:    netlink.FAMILY_V4,
				}}, nil
			},
			LinkByIndex: func(index int) (netlink.Link, error) {
				return &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{
						Index: 10,
						Name:  "ens160",
					},
				}, nil
			},
			AddrList: func(link netlink.Link, family int) ([]netlink.Addr, error) {
				return []netlink.Addr{
					{IPNet: &net.IPNet{
						IP: multicastIP,
					}},
					{
						IPNet: &net.IPNet{
							IP: ip,
						},
					},
				}, nil
			},
		},
		Version: 4,
		expErr:  false,
		expParent: &Parent{
			Name:  "ens160",
			IP:    ip,
			Index: 10,
		},
	}
}

func caseIPv6() TestCase {
	ip := net.ParseIP("fd00::21")
	return TestCase{
		NetLink: NetLink{
			RouteListFiltered: func(family int, filter *netlink.Route, filterMask uint64) ([]netlink.Route, error) {
				return []netlink.Route{{LinkIndex: 10, Dst: nil, Family: netlink.FAMILY_V6}}, nil
			},
			LinkByIndex: func(index int) (netlink.Link, error) {
				return &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{Index: 10, Name: "ens160"},
				}, nil
			},
			AddrList: func(link netlink.Link, family int) ([]netlink.Addr, error) {
				return []netlink.Addr{
					{IPNet: &net.IPNet{IP: ip}},
				}, nil
			},
		},
		Version:   6,
		expErr:    false,
		expParent: &Parent{Name: "ens160", IP: ip, Index: 10},
	}
}

func caseListRouteErr() TestCase {
	return TestCase{
		NetLink: NetLink{
			RouteListFiltered: func(family int, filter *netlink.Route, filterMask uint64) ([]netlink.Route, error) {
				return nil, errors.New("some error")
			},
		},
		Version:   4,
		expErr:    true,
		expParent: &Parent{},
	}
}

func caseListRouteEmpty() TestCase {
	return TestCase{
		NetLink: NetLink{
			RouteListFiltered: func(family int, filter *netlink.Route, filterMask uint64) ([]netlink.Route, error) {
				return nil, nil
			},
		},
		Version:   4,
		expErr:    true,
		expParent: &Parent{},
	}
}

func TestGetParentByName(t *testing.T) {
	cases := map[string]TestCase{
		"case2": case2(),
	}
	for name, item := range cases {
		t.Run(name, func(t *testing.T) {
			getParent := GetParentByName(item.NetLink, "ens160")

			parent, err := getParent(item.Version)
			if !item.expErr {
				assert.NoError(t, err)
			}

			assert.Equal(t, item.expParent, parent)
		})
	}
}

func case2() TestCase {
	ip := net.ParseIP("10.6.0.1")
	return TestCase{
		NetLink: NetLink{
			AddrList: func(link netlink.Link, family int) ([]netlink.Addr, error) {
				return []netlink.Addr{
					{IPNet: &net.IPNet{IP: ip}},
				}, nil
			},
			LinkByName: func(name string) (netlink.Link, error) {
				return &netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{Index: 10, Name: "ens160"},
				}, nil
			},
		},
		Version: 4,
		expErr:  false,
		expParent: &Parent{
			Name:  "ens160",
			IP:    ip,
			Index: 10,
		},
	}
}

func Test_GetParentByDefaultRoute(t *testing.T) {
	mockLink := NetLink{
		RouteListFiltered: func(family int, filter *netlink.Route, filterMask uint64) ([]netlink.Route, error) {
			return nil, errors.New("failed to get routes")
		},
		LinkByIndex: func(index int) (netlink.Link, error) {
			return nil, errors.New("failed to get link by index")
		},
		AddrList: func(link netlink.Link, family int) ([]netlink.Addr, error) {
			return nil, errors.New("failed to list link addrs")
		},
	}

	// error for getting route list
	_, err := GetParentByDefaultRoute(mockLink)(4)
	assert.Error(t, err)

	// error for linking
	mockLink.RouteListFiltered = func(family int, filter *netlink.Route, filterMask uint64) ([]netlink.Route, error) {
		routes := []netlink.Route{
			{
				Family:    netlink.FAMILY_V4,
				LinkIndex: 1,
			},
		}
		return routes, nil
	}
	_, err = GetParentByDefaultRoute(mockLink)(4)
	assert.Error(t, err)

	// error for addrList
	mockLink.LinkByIndex = func(index int) (netlink.Link, error) {
		link := &netlink.Dummy{}
		return link, nil
	}

	_, err = GetParentByDefaultRoute(mockLink)(4)
	assert.Error(t, err)

	// error to find parent interface
	mockLink.AddrList = func(link netlink.Link, family int) ([]netlink.Addr, error) {
		addrs := []netlink.Addr{
			{
				IPNet: &net.IPNet{
					IP:   net.ParseIP("192.168.0"),
					Mask: net.CIDRMask(24, 32),
				},
			},
		}
		return addrs, nil
	}

	_, err = GetParentByDefaultRoute(mockLink)(4)
	assert.Error(t, err)
}

func Test_GetParentByName(t *testing.T) {
	mockLink := NetLink{
		LinkByName: func(name string) (netlink.Link, error) {
			return nil, errors.New("failed to get link by name")
		},
		AddrList: func(link netlink.Link, family int) ([]netlink.Addr, error) {
			return nil, errors.New("failed to list link addrs")
		},
	}

	// error to LinkByName
	_, err := GetParentByName(mockLink, "eth0")(4)
	assert.Error(t, err)

	// error to AddrList
	mockLink.LinkByName = func(name string) (netlink.Link, error) {
		link := &netlink.Dummy{}
		return link, nil
	}

	_, err = GetParentByName(mockLink, "eth0")(4)
	assert.Error(t, err)

	// error to get parent interface
	mockLink.AddrList = func(link netlink.Link, family int) ([]netlink.Addr, error) {
		addrs := []netlink.Addr{
			{
				IPNet: &net.IPNet{
					IP:   net.ParseIP("192.168.0"),
					Mask: net.CIDRMask(24, 32),
				},
			},
		}
		return addrs, nil
	}

	_, err = GetParentByName(mockLink, "eth0")(4)
	assert.Error(t, err)

	_, err = GetParentByName(mockLink, "eth0")(6)
	assert.Error(t, err)
}
