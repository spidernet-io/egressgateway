// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spidernet-io/e2eframework/framework"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

func GetEgressTunnel(f *framework.Framework, name string, egressTunnel *egressv1.EgressTunnel) error {
	key := client.ObjectKey{
		Name: name,
	}
	return f.GetResource(key, egressTunnel)
}

func ListEgressTunnels(f *framework.Framework, opt ...client.ListOption) (*egressv1.EgressTunnelList, error) {
	egressTunnelList := &egressv1.EgressTunnelList{}
	e := f.ListResource(egressTunnelList, opt...)
	if e != nil {
		return nil, e
	}
	return egressTunnelList, nil
}

// GetEgressTunnels return []string of the egressTunnels name
func GetEgressTunnels(f *framework.Framework, opt ...client.ListOption) (egressTunnels []string, e error) {
	egressTunnelList, e := ListEgressTunnels(f, opt...)
	if e != nil {
		return nil, e
	}
	for _, item := range egressTunnelList.Items {
		egressTunnels = append(egressTunnels, item.Name)
	}
	return
}

// CheckEgressTunnelStatus check the status of the egressTunnel cr, parameter 'nodes' is the cluster's nodes name
func CheckEgressTunnelStatus(f *framework.Framework, nodes []string, opt ...client.ListOption) {
	egressTunnels, e := GetEgressTunnels(f, opt...)
	Expect(e).NotTo(HaveOccurred())

	Expect(tools.IsSameSlice(egressTunnels, nodes)).To(BeTrue())

	// get IP version
	enableV4, enableV6, e := GetIPVersion(f)
	Expect(e).NotTo(HaveOccurred())

	for _, node := range nodes {
		egressTunnelObj := &egressv1.EgressTunnel{}
		e = GetEgressTunnel(f, node, egressTunnelObj)
		Expect(e).NotTo(HaveOccurred())
		GinkgoWriter.Printf("egressTunnelObj: %v\n", egressTunnelObj)

		// check egressTunnel status
		status := egressTunnelObj.Status
		// check phase
		Expect(status.Phase).To(Equal(egressv1.EgressTunnelReady))
		// check physicalInterface
		Expect(CheckEgressTunnelInterface(node, status.Tunnel.Parent.Name, time.Second*10)).To(BeTrue())
		// check mac
		Expect(CheckEgressTunnelMac(node, status.Tunnel.MAC, time.Second*10)).To(BeTrue())

		if enableV4 {
			// check vxlan ip
			Expect(CheckEgressTunnelIP(node, status.Tunnel.IPv4, time.Second*10)).To(BeTrue())
			// check node ip
			Expect(CheckNodeIP(node, status.Tunnel.Parent.Name, status.Tunnel.Parent.IPv4, time.Second*10)).To(BeTrue())
		}
		if enableV6 && !enableV4 {
			// check vxlan ip
			Expect(CheckEgressTunnelIP(node, status.Tunnel.IPv6, time.Second*10)).To(BeTrue())
			// check node ip
			Expect(CheckNodeIP(node, status.Tunnel.Parent.Name, status.Tunnel.Parent.IPv6, time.Second*10)).To(BeTrue())
		}
		if enableV6 && enableV4 {
			// check vxlan ip
			Expect(CheckEgressTunnelIP(node, status.Tunnel.IPv6, time.Second*10)).To(BeTrue())
		}
	}
}
