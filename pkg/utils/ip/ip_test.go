// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package ip_test

import (
	"net"
	"reflect"
	"testing"

	"github.com/spidernet-io/egressgateway/pkg/constant"
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
)

func TestIsIPv4(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
		wantErr  bool
	}{
		{"192.168.1.1", true, false},
		{"255.255.255.255", true, false},
		{"0.0.0.0", true, false},
		{"256.256.256.256", false, true},
		{"192.168.1", false, true},
		{"", false, true},
		{"abcd", false, true},
		{"1234:5678:9abc:def0:1234:5678:9abc:def0", false, false},
	}

	for _, tt := range tests {
		got, err := ip.IsIPv4(tt.ip)
		if got != tt.expected {
			t.Errorf("IsIPv4(%q) = %v; want %v", tt.ip, got, tt.expected)
		}

		if tt.wantErr && err == nil {
			t.Errorf("IsIPv4(%q) expected error, but got none", tt.ip)
		}

		if !tt.wantErr && err != nil {
			t.Errorf("IsIPv4(%q) unexpected error: %v", tt.ip, err)
		}
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
		wantErr  bool
	}{
		{"1234:5678:9abc:def0:1234:5678:9abc:def0", true, false},
		{"::1", true, false},
		{"fe80::1", true, false},
		{"192.168.1.1", false, false},
		{"255.255.255.255", false, false},
		{"0.0.0.0", false, false},
		{"::ffff:192.0.2.128", false, false},
		{"12345::", false, true},
		{"", false, true},
		{"abcd", false, true},
	}

	for _, tt := range tests {
		got, err := ip.IsIPv6(tt.ip)
		if got != tt.expected {
			t.Errorf("IsIPv6(%q) = %v; want %v", tt.ip, got, tt.expected)
		}

		if tt.wantErr {
			if err == nil {
				t.Errorf("IsIPv6(%q) expected error, but got none", tt.ip)
			} else if err.Error() != constant.InvalidIPFormat {
				t.Errorf("IsIPv6(%q) expected error message %q, but got %q", tt.ip, constant.InvalidIPFormat, err.Error())
			}
		} else {
			if err != nil {
				t.Errorf("IsIPv6(%q) unexpected error: %v", tt.ip, err)
			}
		}
	}
}

func TestCmp(t *testing.T) {
	ip1 := net.ParseIP("2001:db8::1")
	ip2 := net.ParseIP("2001:db8::2")
	ip3 := net.ParseIP("2001:db8::1")

	if ip.Cmp(ip1, ip2) != -1 {
		t.Errorf("Cmp(%v, %v) = %d; want -1", ip1, ip2, ip.Cmp(ip1, ip2))
	}

	if ip.Cmp(ip1, ip3) != 0 {
		t.Errorf("Cmp(%v, %v) = %d; want 0", ip1, ip3, ip.Cmp(ip1, ip3))
	}

	if ip.Cmp(ip2, ip1) != 1 {
		t.Errorf("Cmp(%v, %v) = %d; want 1", ip2, ip1, ip.Cmp(ip2, ip1))
	}
}

func TestIsIPv6IPRange(t *testing.T) {
	tests := []struct {
		name string
		args string
		want bool
	}{
		{
			name: "empty",
			args: "",
			want: false,
		},
		{
			name: "ipv4",
			args: "10.6.1.21-10.6.1.22",
			want: false,
		},
		{
			name: "ipv6",
			args: "fd00::01-fd00::06",
			want: true,
		},
		{
			name: "ipv6",
			args: "fd00::01-fd00::02-fd00::03",
			want: false,
		},
		{
			name: "ipv6",
			args: "fd00::01-fd::00",
			want: false,
		},
		{
			name: "ipv6",
			args: "fd00::02-fd00::01",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ip.IsIPv6IPRange(tt.args); got != tt.want {
				t.Errorf("IsIPv6IPRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsIPv4IPRange(t *testing.T) {
	tests := []struct {
		name string
		args string
		want bool
	}{
		{
			name: "empty",
			args: "",
			want: false,
		},
		{
			name: "ipv4",
			args: "10.6.1.21-10.6.1.22",
			want: true,
		},
		{
			name: "ipv6",
			args: "fd00::01-fd00::06",
			want: false,
		},
		{
			name: "ipv6",
			args: "10.6.1.22-10.6.1.21",
			want: false,
		},
		{
			name: "ipv6",
			args: "10.6.1.21-10.6.1.22-10.6.1.23",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ip.IsIPv4IPRange(tt.args); got != tt.want {
				t.Errorf("IsIPv4IPRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsIPv4Cidr(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    bool
		wantErr bool
	}{
		{
			name:    "valid IPv4 CIDR",
			args:    "192.168.0.0/16",
			want:    true,
			wantErr: false,
		},
		{
			name:    "invalid IPv4 CIDR",
			args:    "192.168.0.0/33",
			want:    false,
			wantErr: true,
		},
		{
			name:    "IPv6 CIDR",
			args:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334/64",
			want:    false,
			wantErr: false,
		},
		{
			name:    "invalid input format",
			args:    "92.168.0.0/16/24",
			want:    false,
			wantErr: true,
		},
		{
			name:    "invalid IP address",
			args:    "256.256.256.256/16",
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ip.IsIPv4Cidr(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsIPv4Cidr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsIPv4Cidr() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsIPv6Cidr(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    bool
		wantErr bool
	}{
		{
			name:    "valid IPv6 CIDR",
			args:    "2001:0db8:84a3:0000:0000:8a2e:0370:7234/64",
			want:    true,
			wantErr: false,
		},
		{
			name:    "invalid IPv6 CIDR",
			args:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334/129",
			want:    false,
			wantErr: true,
		},
		{
			name:    "not cidr",
			args:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			want:    false,
			wantErr: true,
		},
		{
			name:    "IPv4 cidr",
			args:    "192.168.0.0/16",
			want:    false,
			wantErr: false,
		},
		{
			name:    "invalid CIDR",
			args:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334/abc",
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ip.IsIPv6Cidr(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsIPv6Cidr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsIPv6Cidr() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSameIPs(t *testing.T) {
	type args struct {
		a []string
		b []string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				a: nil,
				b: nil,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "different lengths",
			args: args{
				a: []string{"192.0.2.1", "192.0.2.2"},
				b: []string{"192.0.2.1"},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "same IP",
			args: args{
				a: []string{"192.0.2.1", "192.0.2.2", "192.0.2.3"},
				b: []string{"192.0.2.1", "192.0.2.2", "192.0.2.3"},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "same lists, different order",
			args: args{
				a: []string{"192.0.2.1", "192.0.2.2", "192.0.2.3"},
				b: []string{"192.0.2.2", "192.0.2.1", "192.0.2.3"},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ip.IsSameIPs(tt.args.a, tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsSameIPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsSameIPs() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsIPRange(t *testing.T) {
	type args struct {
		version constant.IPVersion
		ipRange string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ipv4",
			args: args{
				version: constant.IPv4,
				ipRange: "172.18.40.0-172.18.40.10",
			},
			wantErr: false,
		},
		{
			name: "ipv4",
			args: args{
				version: constant.IPv4,
				ipRange: "192.168.0.1-192.168.0.10",
			},
			wantErr: false,
		},
		{
			name: "invalid ipv4",
			args: args{
				version: constant.IPv4,
				ipRange: "172.18.40.0 - 172.18.40.10",
			},
			wantErr: true,
		},
		{
			name: "invalid ipv4",
			args: args{
				version: constant.IPv4,
				ipRange: "172.18.40.1-2001:db8:a0b:12f0::1",
			},
			wantErr: true,
		},
		{
			name: "invalid version",
			args: args{
				version: 0,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ip.IsIPRange(tt.args.version, tt.args.ipRange); (err != nil) != tt.wantErr {
				t.Errorf("IsIPRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMergeIPRanges(t *testing.T) {
	type args struct {
		version  constant.IPVersion
		ipRanges []string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "invalid version",
			args: args{
				version:  0,
				ipRanges: nil,
			},
			wantErr: true,
		},
		{
			name: "",
			args: args{
				version: constant.IPv4,
				ipRanges: []string{
					"172.18.40.1-172.18.40.3",
					"172.18.40.2-172.18.40.5",
				},
			},
			want: []string{
				"172.18.40.1-172.18.40.5",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ip.MergeIPRanges(tt.args.version, tt.args.ipRanges)
			if (err != nil) != tt.wantErr {
				t.Errorf("MergeIPRanges() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeIPRanges() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetIPV4V6(t *testing.T) {
	type args struct {
		ips []string
	}
	tests := []struct {
		name      string
		args      args
		wantIPV4s []string
		wantIPV6s []string
		wantErr   bool
	}{
		{
			name: "empty",
			args: args{
				ips: nil,
			},
			wantIPV4s: nil,
			wantIPV6s: nil,
			wantErr:   false,
		},
		{
			name: "empty",
			args: args{
				ips: []string{""},
			},
			wantIPV4s: nil,
			wantIPV6s: nil,
			wantErr:   true,
		},
		{
			name: "ipv6-empty",
			args: args{
				ips: []string{
					"10.6.1.21",
				},
			},
			wantIPV4s: []string{"10.6.1.21"},
			wantIPV6s: nil,
			wantErr:   false,
		},
		{
			name: "ipv4-empty",
			args: args{
				ips: []string{
					"fd00::1",
				},
			},
			wantIPV4s: nil,
			wantIPV6s: []string{"fd00::1"},
			wantErr:   false,
		},
		{
			name: "ipv4-ipv6",
			args: args{
				ips: []string{
					"10.6.1.21",
					"fd00::1",
					"10.6.1.22",
					"fd00::2",
				},
			},
			wantIPV4s: []string{"10.6.1.21", "10.6.1.22"},
			wantIPV6s: []string{"fd00::1", "fd00::2"},
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIPV4s, gotIPV6s, err := ip.GetIPV4V6(tt.args.ips)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIPV4V6() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotIPV4s, tt.wantIPV4s) {
				t.Errorf("GetIPV4V6() gotIPV4s = %v, want %v", gotIPV4s, tt.wantIPV4s)
			}
			if !reflect.DeepEqual(gotIPV6s, tt.wantIPV6s) {
				t.Errorf("GetIPV4V6() gotIPV6s = %v, want %v", gotIPV6s, tt.wantIPV6s)
			}
		})
	}
}

func TestIsIPRangeOverlap(t *testing.T) {
	type args struct {
		version  constant.IPVersion
		ipRange1 string
		ipRange2 string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "invalid version",
			args: args{
				version: 0,
			},
			wantErr: true,
		},
		{
			name: "valid ipv4",
			args: args{
				version:  constant.IPv4,
				ipRange1: "10.6.1.1-10.6.1.10",
				ipRange2: "10.6.1.11-10.6.1.20",
			},
			wantErr: false,
		},
		{
			name: "invalid range",
			args: args{
				version:  constant.IPv4,
				ipRange1: "10.6.1.1-10.6.1.11",
				ipRange2: "10.6.1.10-10.6.1.20",
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ip.IsIPRangeOverlap(tt.args.version, tt.args.ipRange1, tt.args.ipRange2)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsIPRangeOverlap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsIPRangeOverlap() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsSameIPCidrs(t *testing.T) {
	type args struct {
		a []string
		b []string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				a: []string{},
				b: []string{},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "not-cidr",
			args: args{
				a: []string{"10.6.1.21"},
				b: []string{"10.6.1.21"},
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "cidr-32",
			args: args{
				a: []string{"10.6.1.21/32"},
				b: []string{"10.6.1.21/32"},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cidr-ipv6",
			args: args{
				a: []string{"fd00::1/120"},
				b: []string{"fd00::2/120"},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "cidr",
			args: args{
				a: []string{"fd00::1/120", "10.6.1.21/28"},
				b: []string{"10.6.1.21/28", "fd00::2/120"},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ip.IsSameIPCidrs(tt.args.a, tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsSameIPCidrs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsSameIPCidrs() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsIPIncludedRange(t *testing.T) {
	type args struct {
		version constant.IPVersion
		ip      string
		ipRange []string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "invalid-version",
			args: args{
				version: 0,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "ipv4",
			args: args{
				version: constant.IPv4,
				ip:      "10.6.1.21",
				ipRange: []string{
					"10.6.1.21-10.6.1.22",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "ipv6",
			args: args{
				version: constant.IPv6,
				ip:      "fd00::1",
				ipRange: []string{
					"fd00::1-fd00::10",
				},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ip.IsIPIncludedRange(tt.args.version, tt.args.ip, tt.args.ipRange)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsIPIncludedRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsIPIncludedRange() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetIPV4V6Cidr(t *testing.T) {
	type args struct {
		ipCidrs []string
	}
	tests := []struct {
		name          string
		args          args
		wantIPV4Cidrs []string
		wantIPV6Cidrs []string
		wantErr       bool
	}{
		{
			name: "ipv4",
			args: args{
				ipCidrs: []string{"10.6.1.0/24", "fd00::/120"},
			},
			wantIPV4Cidrs: []string{"10.6.1.0/24"},
			wantIPV6Cidrs: []string{"fd00::/120"},
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIPV4Cidrs, gotIPV6Cidrs, err := ip.GetIPV4V6Cidr(tt.args.ipCidrs)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIPV4V6Cidr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotIPV4Cidrs, tt.wantIPV4Cidrs) {
				t.Errorf("GetIPV4V6Cidr() gotIPV4Cidrs = %v, want %v", gotIPV4Cidrs, tt.wantIPV4Cidrs)
			}
			if !reflect.DeepEqual(gotIPV6Cidrs, tt.wantIPV6Cidrs) {
				t.Errorf("GetIPV4V6Cidr() gotIPV6Cidrs = %v, want %v", gotIPV6Cidrs, tt.wantIPV6Cidrs)
			}
		})
	}
}

func TestCheckIPIncluded(t *testing.T) {
	type args struct {
		ip  string
		ips []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		wantOk  bool
	}{
		{
			name: "invalid ip",
			args: args{
				ip:  "bad-ip",
				ips: []string{"10.6.1.0/24", "10.6.1.1-10.6.1.10", "10.6.1.0"},
			},
			wantOk:  false,
			wantErr: true,
		},
		{
			name: "ipv4 not in range",
			args: args{
				ip:  "10.7.1.0",
				ips: []string{"10.6.1.0/24", "10.6.1.1-10.6.1.10", "10.6.1.0"},
			},
			wantOk:  false,
			wantErr: false,
		},
		{
			name: "ipv6 not in range",
			args: args{
				ip:  "fddd:10:7::1",
				ips: []string{"10.6.1.0/24", "10.6.1.1-10.6.1.10", "10.6.1.0"},
			},
			wantOk:  false,
			wantErr: false,
		},
		{
			name: "ipv4 in single ip",
			args: args{
				ip:  "10.6.1.0",
				ips: []string{"10.6.1.0"},
			},
			wantOk:  true,
			wantErr: false,
		},
		{
			name: "ipv4 in cidr",
			args: args{
				ip:  "10.6.1.0",
				ips: []string{"10.6.1.0/24"},
			},
			wantOk:  true,
			wantErr: false,
		},
		{
			name: "ipv4 in ip range",
			args: args{
				ip:  "10.6.1.9",
				ips: []string{"10.6.1.1-10.6.1.10"},
			},
			wantOk:  true,
			wantErr: false,
		},
		{
			name: "ipv6 in single ip",
			args: args{
				ip:  "fddd:10::0",
				ips: []string{"fddd:10::0"},
			},
			wantOk:  true,
			wantErr: false,
		},
		{
			name: "ipv6 in cidr",
			args: args{
				ip:  "fddd:10::0",
				ips: []string{"fdde:10::1-fdde:10::a", "fddd:10::/120"},
			},
			wantOk:  true,
			wantErr: false,
		},
		{
			name: "ipv6 in ipRange",
			args: args{
				ip:  "fdde:10::2",
				ips: []string{"fddd:10::/120", "fdde:10::1-fdde:10::a"},
			},
			wantOk:  true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := ip.CheckIPIncluded(tt.args.ip, tt.args.ips)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckIPIncluded() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if ok != tt.wantOk {
				t.Errorf("CheckIPIncluded() isIncluded = %v, wantOk %v", ok, tt.wantOk)
				return
			}
		})
	}
}

func TestCidrToIPs(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		wantIPs     []string
		expectError bool
	}{
		{
			name:        "Valid CIDR /32",
			cidr:        "192.168.1.1/32",
			wantIPs:     []string{"192.168.1.1"},
			expectError: false,
		},
		{
			name:        "Valid CIDR /30",
			cidr:        "192.168.1.4/30",
			wantIPs:     []string{"192.168.1.5", "192.168.1.6"},
			expectError: false,
		},
		{
			name:        "Invalid CIDR",
			cidr:        "192.168.1.1/33",
			wantIPs:     nil,
			expectError: true,
		},
		{
			name:        "Valid IPv6 CIDR /128",
			cidr:        "2001:db8::1/128",
			wantIPs:     []string{"2001:db8::1"},
			expectError: false,
		},
		{
			name: "Valid small IPv6 CIDR /124",
			cidr: "2001:db8::/124",
			wantIPs: []string{
				"2001:db8::", "2001:db8::1", "2001:db8::2", "2001:db8::3",
				"2001:db8::4", "2001:db8::5", "2001:db8::6", "2001:db8::7",
				"2001:db8::8", "2001:db8::9", "2001:db8::a", "2001:db8::b",
				"2001:db8::c", "2001:db8::d", "2001:db8::e", "2001:db8::f",
			},
			expectError: false,
		},
		{
			name:        "Invalid IPv6 CIDR",
			cidr:        "2001:db8::1/129",
			wantIPs:     nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIPs, err := ip.CidrToIPs(tt.cidr)
			if (err != nil) != tt.expectError {
				t.Errorf("CidrToIPs() error = %v, expectError %v", err, tt.expectError)
				return
			}
			var gotIPsStr []string
			for _, item := range gotIPs {
				gotIPsStr = append(gotIPsStr, item.String())
			}
			if !reflect.DeepEqual(gotIPsStr, tt.wantIPs) {
				t.Errorf("CidrToIPs() = %v, want %v", gotIPsStr, tt.wantIPs)
			}
		})
	}
}

func TestCidrsToIPs(t *testing.T) {
	tests := []struct {
		name    string
		cidrs   []string
		want    []net.IP
		wantErr bool
	}{
		{
			name:    "single CIDR",
			cidrs:   []string{"192.168.1.0/30"},
			want:    []net.IP{net.ParseIP("192.168.1.1"), net.ParseIP("192.168.1.2")},
			wantErr: false,
		},
		{
			name:    "multiple CIDRs",
			cidrs:   []string{"192.168.1.0/30", "192.168.1.4/31"},
			want:    []net.IP{net.ParseIP("192.168.1.1"), net.ParseIP("192.168.1.2"), net.ParseIP("192.168.1.4"), net.ParseIP("192.168.1.5")},
			wantErr: false,
		},
		{
			name:    "invalid CIDR",
			cidrs:   []string{"invalid"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty slice",
			cidrs:   []string{},
			want:    []net.IP{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIPs, err := ip.CidrsToIPs(tt.cidrs)
			if (err != nil) != tt.wantErr {
				t.Errorf("CidrsToIPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Compare the sorted slices.
			if !ipsEqual(gotIPs, tt.want) {
				t.Errorf("CidrsToIPs() = %v, want %v", gotIPs, tt.want)
			}
		})
	}
}

func TestConvertCidrOrIPRangeToIPs(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		version constant.IPVersion
		want    []net.IP
		wantErr bool
	}{
		//{
		//	name:    "single IP",
		//	in:      []string{"172.30.0.2"},
		//	version: constant.IPv4, // Assuming constant.IPv4 is a valid constant for IPv4
		//	want:    []net.IP{net.ParseIP("172.30.0.2")},
		//	wantErr: false,
		//},
		{
			name:    "IP range",
			in:      []string{"172.30.0.3-172.30.0.5"},
			version: constant.IPv4,
			want: []net.IP{
				net.ParseIP("172.30.0.3"),
				net.ParseIP("172.30.0.4"),
				net.ParseIP("172.30.0.5"),
			},
			wantErr: false,
		},
		{
			name:    "IP CIDR",
			in:      []string{"172.30.1.0/30"},
			version: constant.IPv4,
			want: []net.IP{
				net.ParseIP("172.30.1.1"),
				net.ParseIP("172.30.1.2"),
			},
			wantErr: false,
		},
		{
			name:    "invalid input",
			in:      []string{"this-is-not-an-ip"},
			version: constant.IPv4,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ip.ConvertCidrOrIPrangeToIPs(tt.in, tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertCidrOrIPrangeToIPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !ipsEqual(got, tt.want) {
				t.Errorf("ConvertCidrOrIPrangeToIPs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ipsEqual(got, want []net.IP) bool {
	if len(got) != len(want) {
		return false
	}

	gotMap := make(map[string]struct{}, len(got))

	// Convert the 'got' slice into a map for quick lookup
	for _, v := range got {
		gotMap[v.String()] = struct{}{}
	}

	// Check if all IPs in 'want' are in 'got'
	for _, v := range want {
		if _, found := gotMap[v.String()]; !found {
			return false
		}
		delete(gotMap, v.String()) // Ensure duplicates in 'want' are also in 'got'
	}

	// If the map is empty, all IPs were matched
	return len(gotMap) == 0
}
