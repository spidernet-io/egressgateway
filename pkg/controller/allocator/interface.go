// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package allocator

import "net"

// Interface manages the allocation of IP addresses out of a range.
type Interface interface {
	Allocate(net.IP) error
	AllocateNext() (net.IP, error)
	Release(net.IP) error
	ForEach(func(net.IP))
	CIDR() net.IPNet
	Has(ip net.IP) bool
}
