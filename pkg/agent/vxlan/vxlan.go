// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package vxlan

import (
	"fmt"
	"net"
	"reflect"
	"sync"
	"syscall"

	"github.com/vishvananda/netlink"

	"github.com/spidernet-io/egressgateway/pkg/ethtool"
)

// Device is vxlan device
type Device struct {
	name      string
	lock      sync.RWMutex
	link      *netlink.Vxlan
	getParent func(version int) (*Parent, error)
}

func New(options ...func(*Device)) *Device {
	d := &Device{
		name: "",
		link: nil,
		getParent: GetParentByDefaultRoute(NetLink{
			RouteListFiltered: netlink.RouteListFiltered,
			LinkByIndex:       netlink.LinkByIndex,
			AddrList:          netlink.AddrList,
			LinkByName:        netlink.LinkByName,
		}),
	}
	for _, o := range options {
		o(d)
	}
	return d
}

func WithCustomGetParent(getParent func(version int) (*Parent, error)) func(device *Device) {
	return func(d *Device) {
		d.getParent = getParent
	}
}

// EnsureLink ensure vxlan device
// name, vni, port, mac, mtu, ipv4, ipv6, disableChecksumOffload
func (dev *Device) EnsureLink(name string, vni int, port int, mac net.HardwareAddr, mtu int, ipv4 *net.IPNet, ipv6 *net.IPNet,
	disableChecksumOffload bool) error {

	v := 4
	if ipv4 == nil && ipv6 != nil {
		v = 6
	}

	parent, err := dev.getParent(v)
	if err != nil {
		return fmt.Errorf("failed to get parent: %v", err)
	}

	link := &netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:         name,
			HardwareAddr: mac,
		},
		VxlanId:      vni,
		VtepDevIndex: parent.Index,
		SrcAddr:      parent.IP,
		Port:         port,
		Learning:     false,
	}

	link, err = dev.ensureLink(link)
	if err != nil {
		return err
	}
	dev.setLink(link)

	err = dev.ensureAddr(ipv4, link, netlink.FAMILY_V4)
	if err != nil {
		return err
	}

	err = dev.ensureAddr(ipv6, link, netlink.FAMILY_V6)
	if err != nil {
		return err
	}

	if disableChecksumOffload {
		err = ethtool.EthtoolTXOff(name)
		if err != nil {
			return err
		}
	}

	if err := netlink.LinkSetUp(dev.link); err != nil {
		return fmt.Errorf("set interface to UP with error: %s, %v", dev.link.Attrs().Name, err)
	}

	return nil
}

func (dev *Device) ensureLink(vxlan *netlink.Vxlan) (*netlink.Vxlan, error) {
	err := netlink.LinkAdd(vxlan)
	if err == syscall.EEXIST {
		existing, err := netlink.LinkByName(vxlan.Name)
		if err != nil {
			return nil, err
		}

		conflictAttr := diffLink(vxlan, existing)
		if conflictAttr == nil {
			return existing.(*netlink.Vxlan), nil
		}

		if err = netlink.LinkDel(existing); err != nil {
			return nil, fmt.Errorf("delete vxlan with error: %v", err)
		}

		if err = netlink.LinkAdd(vxlan); err != nil {
			return nil, fmt.Errorf("create vxlan with error: %v", err)
		}
	} else if err != nil {
		return nil, err
	}

	index := vxlan.Index
	link, err := netlink.LinkByIndex(vxlan.Index)
	if err != nil {
		return nil, fmt.Errorf("can't locate created vxlan device with index %v", index)
	}

	var ok bool
	if vxlan, ok = link.(*netlink.Vxlan); !ok {
		return nil, fmt.Errorf("created vxlan device with index %v is not vxlan", index)
	}

	return vxlan, nil
}

type Peer struct {
	IPv4   *net.IP
	IPv6   *net.IP
	Parent net.IP
	MAC    net.HardwareAddr
	Mark   int
}

func (dev *Device) ListNeigh() ([]netlink.Neigh, error) {
	if dev.notReady() {
		return nil, nil
	}
	existingNeigh, err := netlink.NeighList(dev.link.Index, netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}
	return existingNeigh, nil
}

func (dev *Device) Add(peer Peer) error {
	if dev.notReady() {
		return nil
	}
	if peer.IPv6 != nil {
		err := dev.add(peer.MAC, *peer.IPv6)
		if err != nil {
			return err
		}
	}
	if peer.IPv4 != nil {
		err := dev.add(peer.MAC, *peer.IPv4)
		if err != nil {
			return err
		}
	}
	// fdb
	err := netlink.NeighSet(&netlink.Neigh{
		LinkIndex:    dev.link.Index,
		State:        netlink.NUD_PERMANENT,
		Family:       syscall.AF_BRIDGE,
		Flags:        netlink.NTF_SELF,
		IP:           peer.Parent,
		HardwareAddr: peer.MAC,
	})
	if err != nil {
		return err
	}
	return nil
}

func (dev *Device) add(mac net.HardwareAddr, ip net.IP) error {
	// arp
	err := netlink.NeighSet(&netlink.Neigh{
		LinkIndex:    dev.link.Index,
		State:        netlink.NUD_PERMANENT,
		Type:         syscall.RTN_UNICAST,
		IP:           ip,
		HardwareAddr: mac,
	})
	if err != nil {
		return err
	}
	return nil
}

func (dev *Device) Del(neigh netlink.Neigh) error {
	if dev.notReady() {
		return nil
	}

	// fdb
	n := netlink.Neigh{
		LinkIndex:    neigh.LinkIndex,
		State:        netlink.NUD_PERMANENT,
		Family:       syscall.AF_BRIDGE,
		Flags:        netlink.NTF_SELF,
		IP:           neigh.IP,
		HardwareAddr: neigh.HardwareAddr,
	}
	err1 := netlink.NeighDel(&n)

	// arp
	err2 := netlink.NeighDel(&neigh)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("delete neigh, err1=%v err2=%v", err1, err2)
	}
	return nil
}

type conflictAttr struct {
	name string
	got  interface{}
	exp  interface{}
}

func diffLink(l1, l2 netlink.Link) *conflictAttr {
	if l1.Type() != l2.Type() {
		return &conflictAttr{name: "link type", got: l1.Type(), exp: l2.Type()}
	}

	v1 := l1.(*netlink.Vxlan)
	v2 := l2.(*netlink.Vxlan)

	if v1.VxlanId != v2.VxlanId {
		return &conflictAttr{name: "vni", got: v1.VxlanId, exp: v2.VxlanId}
	}

	if v1.VtepDevIndex > 0 && v2.VtepDevIndex > 0 && v1.VtepDevIndex != v2.VtepDevIndex {
		return &conflictAttr{name: "parent interface", got: v1.VtepDevIndex, exp: v2.VtepDevIndex}
	}

	if len(v1.SrcAddr) > 0 && len(v2.SrcAddr) > 0 && !v1.SrcAddr.Equal(v2.SrcAddr) {
		return &conflictAttr{name: "src addr", got: v1.SrcAddr.String(), exp: v2.SrcAddr.String()}
	}

	if len(v1.Group) > 0 && len(v2.Group) > 0 && !v1.Group.Equal(v2.Group) {
		return &conflictAttr{name: "group address", got: v1.Group.String(), exp: v2.Group.String()}
	}

	if v1.L2miss != v2.L2miss {
		return &conflictAttr{name: "l2miss", got: v1.L2miss, exp: v2.L2miss}
	}

	if v1.Port > 0 && v2.Port > 0 && v1.Port != v2.Port {
		return &conflictAttr{name: "port", got: v1.Port, exp: v2.Port}
	}
	return nil
}

func (dev *Device) ensureAddr(ipn *net.IPNet, link netlink.Link, family int) error {
	if ipn == nil {
		return nil
	}

	addr := netlink.Addr{IPNet: ipn}
	gotAddrs, err := netlink.AddrList(link, family)
	if err != nil {
		return err
	}

	needAdd := true
	for _, item := range gotAddrs {
		if !reflect.DeepEqual(item.IPNet, addr.IPNet) {
			if err := netlink.AddrDel(link, &item); err != nil {
				return fmt.Errorf("del addr with error: %s, %v", item, err)
			}
			continue
		}
		needAdd = false
	}

	if needAdd {
		if err := netlink.AddrAdd(link, &addr); err != nil {
			return fmt.Errorf("add addr with error: %v", err)
		}
	}
	return nil
}

func (dev *Device) notReady() bool {
	dev.lock.RLock()
	defer dev.lock.RUnlock()
	return dev.link == nil
}

func (dev *Device) setLink(link *netlink.Vxlan) {
	dev.lock.Lock()
	defer dev.lock.Unlock()
	dev.link = link
}
