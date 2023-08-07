// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package vxlan

import (
	"net"
	"testing"

	"github.com/vishvananda/netlink"
)

type LinkCase struct {
	l1          netlink.Link
	l2          netlink.Link
	expConflict bool
}

func TestDiffLink(t *testing.T) {
	cases := map[string]LinkCase{
		"case1": {
			l1: &netlink.Vxlan{
				VxlanId:      100,
				VtepDevIndex: 1,
			},
			l2: &netlink.Vxlan{
				VxlanId:      100,
				VtepDevIndex: 1,
			},
			expConflict: false,
		},
		"case2 type": {
			l1:          &netlink.Vxlan{},
			l2:          &netlink.Bond{},
			expConflict: true,
		},
		"case3 id": {
			l1:          &netlink.Vxlan{VxlanId: 100},
			l2:          &netlink.Vxlan{VxlanId: 101},
			expConflict: true,
		},
		"case4 vtep dev index": {
			l1: &netlink.Vxlan{
				VxlanId:      100,
				VtepDevIndex: 1,
			},
			l2: &netlink.Vxlan{
				VxlanId:      100,
				VtepDevIndex: 2,
			},
			expConflict: true,
		},
		"case5 addr": {
			l1: &netlink.Vxlan{
				VxlanId:      100,
				VtepDevIndex: 1,
				SrcAddr:      net.ParseIP("10.6.0.1"),
			},
			l2: &netlink.Vxlan{
				VxlanId:      100,
				VtepDevIndex: 1,
				SrcAddr:      net.ParseIP("10.6.0.2"),
			},
			expConflict: true,
		},
		"case6 group": {
			l1:          &netlink.Vxlan{Group: net.ParseIP("10.6.0.1")},
			l2:          &netlink.Vxlan{Group: net.ParseIP("10.6.0.2")},
			expConflict: true,
		},
		"case7 l2miss": {
			l1:          &netlink.Vxlan{L2miss: true},
			l2:          &netlink.Vxlan{L2miss: false},
			expConflict: true,
		},
		"case8 port": {
			l1:          &netlink.Vxlan{Port: 1234},
			l2:          &netlink.Vxlan{Port: 1235},
			expConflict: true,
		},
	}

	for name, linkCase := range cases {
		t.Run(name, func(t *testing.T) {
			conflict := diffLink(linkCase.l1, linkCase.l2)
			if (conflict != nil) != linkCase.expConflict {
				t.Fatal("not equal link")
			}
		})
	}
}

func TestVxlan(t *testing.T) {
	device := New()
	mac, err := net.ParseMAC("66:bf:c7:47:5c:14")
	if err != nil {
		t.Fatal(err)
	}

	ipv6, ipv6Net, err := net.ParseCIDR("fd01::1/120")
	if err != nil {
		t.Fatal(err)
	}

	ipv4Net := &net.IPNet{
		IP:   []byte{10, 6, 1, 21},
		Mask: []byte{255, 255, 255, 0},
	}

	ipv6Net = &net.IPNet{
		IP:   ipv6,
		Mask: ipv6Net.Mask,
	}

	err = device.EnsureLink("egress",
		101, 3456, mac, 0,
		ipv4Net, nil, true)
	if err != nil {
		t.Fatal(err)
	}
}
