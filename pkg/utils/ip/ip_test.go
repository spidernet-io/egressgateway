// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package ip_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
	"net"
	"testing"
)

var _ = Describe("Ip", func() {
	Describe("UT GetIPV4V6", Label("GetIPV4V6"), func() {
		ipv4 := "10.10.0.1"
		ipv6 := "fddd:10::1"
		invalidIPv4 := "10.10.1"

		ipv4s := []string{ipv4}
		ipv6s := []string{ipv6}
		ips := []string{ipv4, ipv6}

		invalidIPs := []string{invalidIPv4, ipv6}

		It("UT GetIPV4V6, expect success", func() {
			v4, v6, err := ip.GetIPV4V6(ips)
			Expect(err).NotTo(HaveOccurred())
			Expect(v4).To(Equal(ipv4s))
			Expect(v6).To(Equal(ipv6s))
		})

		It("UT GetIPV4V6, invalid ip format", func() {
			v4, v6, err := ip.GetIPV4V6(invalidIPs)
			Expect(err).To(HaveOccurred())
			Expect(v4).To(BeNil())
			Expect(v6).To(BeNil())
		})
	})

})

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
			wantErr: true,
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
			wantErr: true,
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
