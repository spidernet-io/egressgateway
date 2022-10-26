// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package types

type ConfigmapConfig struct {
	EnableIPv4      bool `yaml:"enableIPv4"`
	EnableIPv6      bool `yaml:"enableIPv6"`
	StartRouteTable int  `yaml:"startRouteTable"`
	// ["auto", "legacy", "nft"]
	IptablesMode string `yaml:"iptablesMode"`
	// ["iptables", "ebpf"]
	DatapathMode     string `yaml:"datapathMode"`
	TunnelIpv4Subnet string `yaml:"tunnelIpv4Subnet"`
	TunnelIpv6Subnet string `yaml:"tunnelIpv6Subnet"`
	TunnelInterface  string `yaml:"tunnelInterface"`
	ForwardMethod    string `yaml:"forwardMethod"`

	VxlanID      int `yaml:"VxlanID"`
	VxlanUdpPort int `yaml:"vxlanUdpPort"`
}
