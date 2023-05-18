// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package constant

type IPVersion = int64

const (
	IPv4 IPVersion = 4
	IPv6 IPVersion = 6
)

const (
	InvalidIPVersion = IPVersion(976)
	InvalidCIDR      = "invalid CIDR"
	InvalidIP        = "invalid IP"
	InvalidIPRange   = "invalid IP range"
	InvalidDst       = "invalid routing destination"
	InvalidGateway   = "invalid routing gateway"
	InvalidIPFormat  = "invalid ip format"
)

var InvalidIPRanges = []string{InvalidIPRange}
