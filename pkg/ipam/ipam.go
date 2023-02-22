// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package ipam

import (
	"fmt"
	"net"

	"github.com/cilium/ipam/service/ipallocator"
	"github.com/spidernet-io/egressgateway/pkg/lock"
	"go.uber.org/zap"
)

type TunnleIpam struct {
	mutex         lock.RWMutex
	ipv4cidr      *net.IPNet
	ipv6cidr      *net.IPNet
	ipmap         map[string]string
	ipmapIPv6     map[string]string
	allocator     *ipallocator.Range
	allocatorIPv6 *ipallocator.Range
	log           *zap.Logger
	EnableIPv4    bool
	EnableIPv6    bool
}

func (t *TunnleIpam) Init(ipv4cidr, ipv6cidr string, log *zap.Logger) error {
	t.log = log
	if t.EnableIPv4 {
		_, cidr, err := net.ParseCIDR(ipv4cidr)
		if err != nil {
			t.log.Sugar().Errorf("CIDR(%v) format is incorrect", ipv4cidr)
			return fmt.Errorf("CIDR(%v) format is incorrect", ipv4cidr)
		}

		cidrRange, err := ipallocator.NewCIDRRange(cidr)
		if err != nil {
			t.log.Sugar().Errorf("Failed to initialize the tunnel IPv4 CIDR(%v)", err)
			return fmt.Errorf("Failed to initialize the tunnel IPv4 CIDR(%v)", err)
		}

		t.ipv4cidr = cidr
		t.allocator = cidrRange
		t.ipmap = make(map[string]string)
	}

	if t.EnableIPv6 {
		_, cidrIPV6, err := net.ParseCIDR(ipv6cidr)
		if err != nil {
			t.log.Sugar().Errorf("CIDR(%v) format is incorrect", ipv6cidr)
			return fmt.Errorf("CIDR(%v) format is incorrect", ipv6cidr)
		}

		cidrRangeIPV6, err := ipallocator.NewCIDRRange(cidrIPV6)
		if err != nil {
			t.log.Sugar().Errorf("Failed to initialize the tunnel IPv6 CIDR(%v)", err)
			return fmt.Errorf("Failed to initialize the tunnel IPv6 CIDR(%v)", err)
		}

		t.ipv6cidr = cidrIPV6
		t.allocatorIPv6 = cidrRangeIPV6
		t.ipmapIPv6 = make(map[string]string)
	}

	return nil
}

func (t *TunnleIpam) GetNode(ip string) string {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	for k, v := range t.ipmap {
		if v == ip {
			return k
		}
	}

	for k, v := range t.ipmapIPv6 {
		if v == ip {
			return k
		}
	}

	return ""
}

func (t *TunnleIpam) SetNodeIP(ip, node string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.ipmap[node] != "" {
		if t.ipmap[node] != ip {
			if err := t.allocator.Release(net.ParseIP(t.ipmap[node])); err != nil {
				t.log.Sugar().Errorf("The IP(%v) fails to be released when the node(%v) is bound to IP(%v) again", t.ipmap[node], node, ip)
				return err
			}
		}
	}

	err := t.allocator.Allocate(net.ParseIP(ip))
	if err != nil && err != ipallocator.ErrAllocated {
		return err
	}

	t.ipmap[node] = ip
	t.log.Sugar().Infof("The node(%v) is bound to IP(%v) successfully", node, ip)
	return nil
}

func (t *TunnleIpam) SetNodeIPv6(ip, node string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.ipmapIPv6[node] != "" {
		if t.ipmapIPv6[node] != ip {
			if err := t.allocatorIPv6.Release(net.ParseIP(t.ipmapIPv6[node])); err != nil {
				t.log.Sugar().Errorf("The IP(%v) fails to be released when the node(%v) is bound to IP(%v) again", t.ipmapIPv6[node], node, ip)
				return err
			}
		}
	}

	err := t.allocatorIPv6.Allocate(net.ParseIP(ip))
	if err != nil && err != ipallocator.ErrAllocated {
		return err
	}

	t.ipmapIPv6[node] = ip
	t.log.Sugar().Infof("The node(%v) is bound to IP(%v) successfully", node, ip)
	return nil
}

func (t *TunnleIpam) CheckUse(ip, node string) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	for i := 0; i < len(ip); i++ {
		switch ip[i] {
		case '.':
			return t.ipmap[node] == ip
		case ':':
			return t.ipmapIPv6[node] == ip
		}
	}
	return false
}

func (t *TunnleIpam) CheckIsOK(ip string) bool {
	for i := 0; i < len(ip); i++ {
		switch ip[i] {
		case '.':
			return t.ipv4cidr.Contains(net.ParseIP(ip))
		case ':':
			return t.ipv6cidr.Contains(net.ParseIP(ip))
		}
	}
	return false
}

func (t *TunnleIpam) Acquire(node string) (net.IP, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	ip, err := t.allocator.AllocateNext()
	if err != nil {
		return nil, err
	}

	t.ipmap[node] = ip.String()
	t.log.Sugar().Infof("The node(%v) is bound to IP(%v) successfully", node, ip.String())
	return ip, nil
}

func (t *TunnleIpam) AcquireIPv6(node string) (net.IP, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	ip, err := t.allocatorIPv6.AllocateNext()
	if err != nil {
		return nil, err
	}

	t.ipmapIPv6[node] = ip.String()
	t.log.Sugar().Infof("The node(%v) is bound to IP(%v) successfully", node, ip.String())
	return ip, nil
}

func (t *TunnleIpam) ReleaseByIP(ip string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	for i := 0; i < len(ip); i++ {
		switch ip[i] {
		case '.':
			releaseIP := net.ParseIP(ip)
			if err := t.allocator.Release(releaseIP); err != nil {
				t.log.Sugar().Errorf("IP(%v) release failure; err(%v)", ip, err)
				return err
			}

			for k, v := range t.ipmap {
				if v == ip {
					t.log.Sugar().Infof("IP release succeeded; IP=%v, Node=%v", v, k)
					delete(t.ipmap, k)
				}
			}

			return nil
		case ':':
			releaseIP := net.ParseIP(ip)
			if err := t.allocatorIPv6.Release(releaseIP); err != nil {
				t.log.Sugar().Errorf("IP(%v) release failure; err(%v)", ip, err)
				return err
			}

			for k, v := range t.ipmapIPv6 {
				if v == ip {
					t.log.Sugar().Infof("IP release succeeded; IP=%v, Node=%v", v, k)
					delete(t.ipmapIPv6, k)
				}
			}

			return nil
		}
	}

	return fmt.Errorf("IP(%v) release failure", ip)
}

func (t *TunnleIpam) ReleaseByNode(node string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	releaseIP := net.ParseIP(t.ipmap[node])
	if err := t.allocator.Release(releaseIP); err != nil {
		t.log.Sugar().Errorf("release failure; IP=%v, Node=%v; err(%v)", releaseIP, node, err)
		return err
	}

	t.log.Sugar().Infof("IP release succeeded; IP=%v, Node=%v", releaseIP, node)
	delete(t.ipmap, node)
	return nil
}

func (t *TunnleIpam) ReleaseIPv6ByNode(node string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	releaseIP := net.ParseIP(t.ipmapIPv6[node])
	if err := t.allocatorIPv6.Release(releaseIP); err != nil {
		t.log.Sugar().Errorf("release failure; IP=%v, Node=%v; err(%v)", releaseIP, node, err)
		return err
	}

	t.log.Sugar().Infof("IP release succeeded; IP=%v, Node=%v", releaseIP, node)
	delete(t.ipmapIPv6, node)
	return nil
}
