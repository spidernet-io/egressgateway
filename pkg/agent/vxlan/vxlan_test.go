// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package vxlan

import (
	"errors"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spidernet-io/egressgateway/pkg/ethtool"
	"github.com/stretchr/testify/assert"
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

func Test_WithCustomGetParent(t *testing.T) {

	getParent := func(version int) (*Parent, error) {
		return &Parent{}, nil
	}
	retFunc := WithCustomGetParent(getParent)

	de := New()
	retFunc(de)
}

func Test_New(t *testing.T) {
	opts := []func(*Device){
		func(d *Device) {
			t.Log("for test")
		},
	}
	New(opts...)
}

func Test_EnsureLink(t *testing.T) {
	cases := map[string]struct {
		prepare func() (name string, vni int, port int, mac net.HardwareAddr, mtu int,
			ipv4, ipv6 *net.IPNet,
			disableChecksumOffload bool)
		patchFunc       func(dev *Device) []gomonkey.Patches
		customGetParent func(getParent func(version int) (*Parent, error)) func(device *Device)
		expErr          bool
	}{
		"failed RangeSize": {
			prepare: mock_EnsureLink_params,
			// patchFunc: err_EnsureLink_getParent,
			customGetParent: err_EnsureLink_getParent,
			expErr:          true,
		},
		"ipv4 nil, ipv6 not nil ": {
			prepare: mock_EnsureLink_nil_ipv4,
			// patchFunc: err_EnsureLink_getParent,
			customGetParent: err_EnsureLink_getParent,
			expErr:          true,
		},
		"failed ensureLink": {
			prepare:         mock_EnsureLink_params,
			patchFunc:       err_EnsureLink_ensureLink,
			customGetParent: succ_EnsureLink_getParent,
			expErr:          true,
		},
		"failed ensureAddr v4": {
			prepare:         mock_EnsureLink_params,
			patchFunc:       err_EnsureLink_ensureAddr_v4,
			customGetParent: succ_EnsureLink_getParent,
			expErr:          true,
		},
		"failed ensureFilter": {
			prepare:         mock_EnsureLink_params,
			patchFunc:       err_EnsureLink_ensureFilter,
			customGetParent: succ_EnsureLink_getParent,
			expErr:          true,
		},
		"failed EthtoolTXOff": {
			prepare:         mock_EnsureLink_disableChecksumOffload_on,
			patchFunc:       err_EnsureLink_EthtoolTXOff,
			customGetParent: succ_EnsureLink_getParent,
			expErr:          true,
		},
		"failed LinkSetUp": {
			prepare:         mock_EnsureLink_disableChecksumOffload_on,
			patchFunc:       err_EnsureLink_LinkSetUp,
			customGetParent: succ_EnsureLink_getParent,
			expErr:          true,
		},
	}

	var dev *Device
	var err error

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.customGetParent != nil {
				dev = new(Device)
				tc.customGetParent(func(version int) (*Parent, error) {
					return nil, errors.New("some err")
				})(dev)
			} else {
				dev = New()
			}

			var patches = make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patchess := tc.patchFunc(dev)
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}

			name, vni, port, mac, mtu, ipv4, ipv6, disableChecksumOffload := tc.prepare()
			err = dev.EnsureLink(name, vni, port, mac, mtu, ipv4, ipv6, disableChecksumOffload)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_ensureLink(t *testing.T) {
	cases := map[string]struct {
		patchFunc func() []gomonkey.Patches
		expErr    bool
	}{
		"failed LinkByName": {
			patchFunc: err_ensureLink_LinkByName,
			expErr:    true,
		},
		"nil conflictAttr": {
			patchFunc: err_ensureLink_nil_conflictAttr,
		},
		"failed LinkDel": {
			patchFunc: err_ensureLink_LinkDel,
			expErr:    true,
		},
		"failed second LinkAdd": {
			patchFunc: err_ensureLink_second_LinkAdd,
			expErr:    true,
		},
		"failed first LinkAdd": {
			patchFunc: err_ensureLink_first_LinkAdd,
			expErr:    true,
		},
		"failed LinkByIndex": {
			patchFunc: err_ensureLink_LinkByIndex,
			expErr:    true,
		},
		"failed not Vxlan type": {
			patchFunc: err_ensureLink_not_Vxlan_type,
			expErr:    true,
		},
		"succeeded ensureLink": {
			patchFunc: succ_ensureLink,
		},
	}

	var dev *Device
	var vxlan *netlink.Vxlan
	var err error

	dev = new(Device)
	vxlan = &netlink.Vxlan{}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var patches = make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patchess := tc.patchFunc()
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}

			_, err = dev.ensureLink(vxlan)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_ensureFilter(t *testing.T) {
	dev := new(Device)
	patch := gomonkey.ApplyFuncReturn(writeProcSys, errors.New("some err"))
	defer patch.Reset()
	err := dev.ensureFilter(&net.IPNet{}, &net.IPNet{})
	assert.Error(t, err)
}

func Test_ListNeigh(t *testing.T) {
	cases := map[string]struct {
		patchFunc func(*Device) []gomonkey.Patches
		expErr    bool
	}{
		"device not ready": {
			patchFunc: err_ListNeigh_notReady,
		},
		"failed NeighList": {
			patchFunc: err_ListNeigh_NeighList,
			expErr:    true,
		},
		"succeeded ListNeigh": {
			patchFunc: succ_ListNeigh,
		},
	}

	var dev *Device
	var err error

	dev = new(Device)
	dev.link = &netlink.Vxlan{}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var patches = make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patchess := tc.patchFunc(dev)
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}

			_, err = dev.ListNeigh()
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_Add(t *testing.T) {
	cases := map[string]struct {
		setParams func(*Peer)
		patchFunc func(*Device) []gomonkey.Patches
		expErr    bool
	}{
		"device not ready": {
			patchFunc: err_Add_notReady,
		},
		"failed add v6": {
			setParams: mock_Add_params_v6(),
			patchFunc: err_Add_add,
			expErr:    true,
		},
		"failed add v4": {
			setParams: mock_Add_params_v4(),
			patchFunc: err_Add_add,
			expErr:    true,
		},
		"failed NeighSet": {
			patchFunc: err_Add_NeighSet,
			expErr:    true,
		},
		"succeeded Add": {
			patchFunc: succ_Add,
		},
	}

	var dev *Device
	var err error
	var peer Peer

	dev = new(Device)
	dev.link = &netlink.Vxlan{}
	peer = Peer{}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.setParams != nil {
				tc.setParams(&peer)
			}

			var patches = make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patchess := tc.patchFunc(dev)
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}

			err = dev.Add(peer)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_add(t *testing.T) {
	cases := map[string]struct {
		patchFunc func(*Device) []gomonkey.Patches
		expErr    bool
	}{
		"failed NeighSet": {
			patchFunc: err_add_NeighSet,
			expErr:    true,
		},
		"succeeded add": {
			patchFunc: succ_add,
		},
	}

	var dev *Device
	var err error
	var mac net.HardwareAddr
	var ip net.IP

	dev = new(Device)
	dev.link = &netlink.Vxlan{}
	mac, _ = net.ParseMAC("00:00:5e:00:53:01")
	ip = net.ParseIP("192.168.0.2")

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			var patches = make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patchess := tc.patchFunc(dev)
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}

			err = dev.add(mac, ip)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_Del(t *testing.T) {
	cases := map[string]struct {
		setParams func(*Peer)
		patchFunc func(*Device) []gomonkey.Patches
		expErr    bool
	}{
		"device not ready": {
			patchFunc: err_Del_notReady,
		},

		"failed NeightDel": {
			patchFunc: err_Del_NeighDel,
			expErr:    true,
		},
		"succeeded Del": {
			patchFunc: succ_Del,
		},
	}

	var dev *Device
	var err error
	var neigh netlink.Neigh

	dev = new(Device)
	dev.link = &netlink.Vxlan{}
	neigh = netlink.Neigh{}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var patches = make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patchess := tc.patchFunc(dev)
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}

			err = dev.Del(neigh)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_ensureAddr(t *testing.T) {
	cases := map[string]struct {
		setParams func() (ipn *net.IPNet, link netlink.Link, family int)
		patchFunc func() []gomonkey.Patches
		expErr    bool
	}{
		"nil ipn": {
			setParams: mock_ensureAddr_nil_ipn,
		},
		"failed AddrList": {
			setParams: mock_ensureAddr_params,
			patchFunc: err_ensureAddr_AddrList,
			expErr:    true,
		},
		"failed AddrDel": {
			setParams: mock_ensureAddr_params,
			patchFunc: err_ensureAddr_AddrDel,
			expErr:    true,
		},
		"failed AddrAdd": {
			setParams: mock_ensureAddr_params,
			patchFunc: err_ensureAddr_AddrAdd,
			expErr:    true,
		},
		"succeede ensureAddr": {
			setParams: mock_ensureAddr_params,
			patchFunc: succ_ensureAddr,
		},
	}

	var dev *Device
	var err error

	var ipn *net.IPNet
	var link netlink.Link
	var family int
	dev = new(Device)
	dev.link = &netlink.Vxlan{}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.setParams != nil {
				ipn, link, family = tc.setParams()
			} else {
				t.Fatal("need set params")
			}
			var patches = make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patchess := tc.patchFunc()
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}

			err = dev.ensureAddr(ipn, link, family)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_notReady(t *testing.T) {
	dev := new(Device)
	dev.link = nil
	ok := dev.notReady()
	assert.True(t, ok)
}

func Test_writeProcSys(t *testing.T) {
	cases := map[string]struct {
		patchFunc func() []gomonkey.Patches
		expErr    bool
	}{
		"failed OpenFile": {
			expErr: true,
		},

		"failed Write": {
			patchFunc: err_writeProcSys_Write,
			expErr:    true,
		},
		"failed short length": {
			patchFunc: err_writeProcSys_shortLen,
			expErr:    true,
		},

		"failed Close": {
			patchFunc: succ_writeProcSys,
		},
	}

	var err error
	var path, value string

	path = "foo/bar"
	value = "12345"

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var patches = make([]gomonkey.Patches, 0)
			if tc.patchFunc != nil {
				patchess := tc.patchFunc()
				patches = append(patches, patchess...)
				defer func() {
					for _, p := range patches {
						p.Reset()
					}
				}()
			}

			err = writeProcSys(path, value)
			if tc.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func mock_EnsureLink_params() (name string, vni int, port int, mac net.HardwareAddr, mtu int,
	ipv4, ipv6 *net.IPNet,
	disableChecksumOffload bool) {
	name = "vxlan0xx"
	vni = 100
	port = 4789
	mac, _ = net.ParseMAC("00:11:22:33:44:55")
	mtu = 1500
	ipv4 = &net.IPNet{
		IP:   net.IPv4(192, 168, 0, 1),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}
	ipv6 = &net.IPNet{
		IP:   net.ParseIP("2001:db8::1"),
		Mask: net.CIDRMask(64, 128),
	}
	disableChecksumOffload = false
	return
}

func mock_EnsureLink_nil_ipv4() (name string, vni int, port int, mac net.HardwareAddr, mtu int,
	ipv4, ipv6 *net.IPNet,
	disableChecksumOffload bool) {
	name = "vxlan0xx"
	vni = 100
	port = 4789
	mac, _ = net.ParseMAC("00:11:22:33:44:55")
	mtu = 1500
	ipv4 = nil
	ipv6 = &net.IPNet{
		IP:   net.ParseIP("2001:db8::1"),
		Mask: net.CIDRMask(64, 128),
	}
	disableChecksumOffload = false
	return
}

func mock_EnsureLink_disableChecksumOffload_on() (name string, vni int, port int, mac net.HardwareAddr, mtu int,
	ipv4, ipv6 *net.IPNet,
	disableChecksumOffload bool) {
	name = "vxlan0xx"
	vni = 100
	port = 4789
	mac, _ = net.ParseMAC("00:11:22:33:44:55")
	mtu = 1500
	ipv4 = &net.IPNet{
		IP:   net.IPv4(192, 168, 0, 1),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}
	ipv6 = &net.IPNet{
		IP:   net.ParseIP("2001:db8::1"),
		Mask: net.CIDRMask(64, 128),
	}
	disableChecksumOffload = true
	return
}

func err_EnsureLink_getParent(getParent func(version int) (*Parent, error)) func(device *Device) {
	return WithCustomGetParent(func(version int) (*Parent, error) {
		return nil, errors.New("some err")
	})
}

func succ_EnsureLink_getParent(getParent func(version int) (*Parent, error)) func(device *Device) {
	return WithCustomGetParent(func(version int) (*Parent, error) {
		return &Parent{}, nil
	})
}

func err_EnsureLink_ensureLink(dev *Device) []gomonkey.Patches {
	patch := gomonkey.NewPatches()
	patch.ApplyPrivateMethod(dev, "ensureLink", func(_ *Device) (*netlink.Vxlan, error) {
		return nil, errors.New("some err")
	})

	return []gomonkey.Patches{*patch}
}

func err_EnsureLink_ensureAddr_v4(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "ensureLink", func(_ *Device) (*netlink.Vxlan, error) {
		return &netlink.Vxlan{}, nil
	})
	patch2 := gomonkey.ApplyPrivateMethod(dev, "ensureAddr", func(_ *Device) error {
		return errors.New("some error")
	})

	return []gomonkey.Patches{*patch1, *patch2}
}

func err_EnsureLink_ensureFilter(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "ensureLink", func(_ *Device) (*netlink.Vxlan, error) {
		return &netlink.Vxlan{}, nil
	})
	patch2 := gomonkey.ApplyPrivateMethod(dev, "ensureAddr", func(_ *Device) error {
		return nil
	})
	patch3 := gomonkey.ApplyPrivateMethod(dev, "ensureFilter", func(_ *Device) error {
		return errors.New("some errr")
	})

	return []gomonkey.Patches{*patch1, *patch2, *patch3}
}

func err_EnsureLink_EthtoolTXOff(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "ensureLink", func(_ *Device) (*netlink.Vxlan, error) {
		return &netlink.Vxlan{}, nil
	})
	patch2 := gomonkey.ApplyPrivateMethod(dev, "ensureAddr", func(_ *Device) error {
		return nil
	})
	patch3 := gomonkey.ApplyPrivateMethod(dev, "ensureFilter", func(_ *Device) error {
		return nil
	})
	patch4 := gomonkey.ApplyFuncReturn(ethtool.EthtoolTXOff, errors.New("some err"))

	return []gomonkey.Patches{*patch1, *patch2, *patch3, *patch4}
}

func err_EnsureLink_LinkSetUp(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "ensureLink", func(_ *Device) (*netlink.Vxlan, error) {
		return &netlink.Vxlan{}, nil
	})
	patch2 := gomonkey.ApplyPrivateMethod(dev, "ensureAddr", func(_ *Device) error {
		return nil
	})
	patch3 := gomonkey.ApplyPrivateMethod(dev, "ensureFilter", func(_ *Device) error {
		return nil
	})
	patch4 := gomonkey.ApplyFuncReturn(ethtool.EthtoolTXOff, nil)

	patch5 := gomonkey.ApplyFuncReturn(netlink.LinkSetUp, errors.New("some err"))

	return []gomonkey.Patches{*patch1, *patch2, *patch3, *patch4, *patch5}
}

func err_ensureLink_LinkByName() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkAdd, syscall.EEXIST)
	patch2 := gomonkey.ApplyFuncReturn(netlink.LinkByName, nil, errors.New("some err"))
	return []gomonkey.Patches{*patch1, *patch2}
}

func err_ensureLink_nil_conflictAttr() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkAdd, syscall.EEXIST)
	patch2 := gomonkey.ApplyFuncReturn(netlink.LinkByName, &netlink.Vxlan{}, nil)
	return []gomonkey.Patches{*patch1, *patch2}
}

func err_ensureLink_LinkDel() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkAdd, syscall.EEXIST)
	patch2 := gomonkey.ApplyFuncReturn(netlink.LinkByName, &netlink.Vxlan{VxlanId: 100}, nil)
	patch3 := gomonkey.ApplyFuncReturn(netlink.LinkDel, errors.New("some err"))
	return []gomonkey.Patches{*patch1, *patch2, *patch3}
}

func err_ensureLink_second_LinkAdd() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncSeq(netlink.LinkAdd, []gomonkey.OutputCell{
		{Values: gomonkey.Params{syscall.EEXIST}, Times: 1},
		{Values: gomonkey.Params{errors.New("some err")}, Times: 1},
	})
	patch2 := gomonkey.ApplyFuncReturn(netlink.LinkByName, &netlink.Vxlan{VxlanId: 100}, nil)
	patch3 := gomonkey.ApplyFuncReturn(netlink.LinkDel, nil)
	return []gomonkey.Patches{*patch1, *patch2, *patch3}
}

func err_ensureLink_first_LinkAdd() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkAdd, errors.New("some err"))
	return []gomonkey.Patches{*patch1}
}

func err_ensureLink_LinkByIndex() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkAdd, nil)
	patch2 := gomonkey.ApplyFuncReturn(netlink.LinkByIndex, nil, errors.New("some err"))
	return []gomonkey.Patches{*patch1, *patch2}
}

func err_ensureLink_not_Vxlan_type() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkAdd, nil)
	patch2 := gomonkey.ApplyFuncReturn(netlink.LinkByIndex, &netlink.Vlan{}, nil)
	return []gomonkey.Patches{*patch1, *patch2}
}

func succ_ensureLink() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.LinkAdd, nil)
	patch2 := gomonkey.ApplyFuncReturn(netlink.LinkByIndex, &netlink.Vxlan{}, nil)
	return []gomonkey.Patches{*patch1, *patch2}
}

func err_ListNeigh_notReady(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "notReady", func(_ *Device) bool {
		return true
	})
	return []gomonkey.Patches{*patch1}
}

func err_ListNeigh_NeighList(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "notReady", func(_ *Device) bool {
		return false
	})
	patch2 := gomonkey.ApplyFuncReturn(netlink.NeighList, nil, errors.New("some err"))
	return []gomonkey.Patches{*patch1, *patch2}
}

func succ_ListNeigh(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "notReady", func(_ *Device) bool {
		return false
	})
	patch2 := gomonkey.ApplyFuncReturn(netlink.NeighList, nil, nil)
	return []gomonkey.Patches{*patch1, *patch2}
}

func err_Add_notReady(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "notReady", func(_ *Device) bool {
		return true
	})
	return []gomonkey.Patches{*patch1}
}

func err_Add_add(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "notReady", func(_ *Device) bool {
		return false
	})
	patch2 := gomonkey.ApplyPrivateMethod(dev, "add", func(_ *Device) error {
		return errors.New("some err")
	})
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_Add_params_v4() func(*Peer) {
	return func(peer *Peer) {
		ipv4 := net.ParseIP("192.168.0.2")
		peer.IPv4 = &ipv4
	}
}

func mock_Add_params_v6() func(*Peer) {
	return func(peer *Peer) {
		ipv6 := net.ParseIP("fddd:12::12")
		peer.IPv6 = &ipv6
	}
}

func err_Add_NeighSet(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "notReady", func(_ *Device) bool {
		return false
	})
	patch2 := gomonkey.ApplyPrivateMethod(dev, "add", func(_ *Device) error {
		return nil
	})
	patch3 := gomonkey.ApplyFuncReturn(netlink.NeighSet, errors.New("some err"))

	return []gomonkey.Patches{*patch1, *patch2, *patch3}
}

func succ_Add(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "notReady", func(_ *Device) bool {
		return false
	})
	patch2 := gomonkey.ApplyPrivateMethod(dev, "add", func(_ *Device) error {
		return nil
	})
	patch3 := gomonkey.ApplyFuncReturn(netlink.NeighSet, nil)

	return []gomonkey.Patches{*patch1, *patch2, *patch3}
}

func err_add_NeighSet(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.NeighSet, errors.New("some err"))
	return []gomonkey.Patches{*patch1}
}

func succ_add(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.NeighSet, nil)
	return []gomonkey.Patches{*patch1}
}

func err_Del_notReady(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "notReady", func(_ *Device) bool {
		return true
	})
	return []gomonkey.Patches{*patch1}
}

func err_Del_NeighDel(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "notReady", func(_ *Device) bool {
		return false
	})
	patch2 := gomonkey.ApplyFuncReturn(netlink.NeighDel, errors.New("some err"))

	return []gomonkey.Patches{*patch1, *patch2}
}

func succ_Del(dev *Device) []gomonkey.Patches {
	patch1 := gomonkey.ApplyPrivateMethod(dev, "notReady", func(_ *Device) bool {
		return false
	})
	patch2 := gomonkey.ApplyFuncReturn(netlink.NeighDel, nil)

	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_ensureAddr_params() (ipn *net.IPNet, link netlink.Link, family int) {
	return &net.IPNet{IP: net.ParseIP("192.168.0.1")}, &netlink.Dummy{}, 4
}

func mock_ensureAddr_nil_ipn() (ipn *net.IPNet, link netlink.Link, family int) {
	return nil, &netlink.Dummy{}, 4
}

func err_ensureAddr_AddrList() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.AddrList, nil, errors.New("some err"))

	return []gomonkey.Patches{*patch1}
}

func err_ensureAddr_AddrDel() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.AddrList, []netlink.Addr{
		{IPNet: &net.IPNet{IP: net.ParseIP("192.168.0.2")}},
		{IPNet: &net.IPNet{IP: net.ParseIP("192.168.0.3")}},
	}, nil)
	patch2 := gomonkey.ApplyFuncReturn(netlink.AddrDel, errors.New("some err"))

	return []gomonkey.Patches{*patch1, *patch2}
}

func err_ensureAddr_AddrAdd() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.AddrList, []netlink.Addr{
		{IPNet: &net.IPNet{IP: net.ParseIP("192.168.0.2")}},
		{IPNet: &net.IPNet{IP: net.ParseIP("192.168.0.3")}},
	}, nil)
	patch2 := gomonkey.ApplyFuncReturn(netlink.AddrDel, nil)
	patch3 := gomonkey.ApplyFuncReturn(netlink.AddrAdd, errors.New("some err"))

	return []gomonkey.Patches{*patch1, *patch2, *patch3}
}

func succ_ensureAddr() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(netlink.AddrList, []netlink.Addr{
		{IPNet: &net.IPNet{IP: net.ParseIP("192.168.0.2")}},
		{IPNet: &net.IPNet{IP: net.ParseIP("192.168.0.3")}},
	}, nil)
	patch2 := gomonkey.ApplyFuncReturn(netlink.AddrDel, nil)
	patch3 := gomonkey.ApplyFuncReturn(netlink.AddrAdd, nil)

	return []gomonkey.Patches{*patch1, *patch2, *patch3}
}

func err_writeProcSys_Write() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(os.OpenFile, &os.File{}, nil)
	patch2 := gomonkey.ApplyMethodReturn(&os.File{}, "Write", 0, errors.New("some err"))
	return []gomonkey.Patches{*patch1, *patch2}
}

func err_writeProcSys_shortLen() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(os.OpenFile, &os.File{}, nil)
	patch2 := gomonkey.ApplyMethodReturn(&os.File{}, "Write", 0, nil)
	return []gomonkey.Patches{*patch1, *patch2}
}

func succ_writeProcSys() []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(os.OpenFile, &os.File{}, nil)
	patch2 := gomonkey.ApplyMethodReturn(&os.File{}, "Write", 10, nil)
	return []gomonkey.Patches{*patch1, *patch2}
}
