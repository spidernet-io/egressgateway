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

func GetEgressNode(f *framework.Framework, name string, egressNode *egressv1.EgressTunnel) error {
	key := client.ObjectKey{
		Name: name,
	}
	return f.GetResource(key, egressNode)
}

func ListEgressNodes(f *framework.Framework, opt ...client.ListOption) (*egressv1.EgressTunnelList, error) {
	egressNodeList := &egressv1.EgressTunnelList{}
	e := f.ListResource(egressNodeList, opt...)
	if e != nil {
		return nil, e
	}
	return egressNodeList, nil
}

// GetEgressNodes return []string of the egressNodes name
func GetEgressNodes(f *framework.Framework, opt ...client.ListOption) (egressNodes []string, e error) {
	egressNodeList, e := ListEgressNodes(f, opt...)
	if e != nil {
		return nil, e
	}
	for _, item := range egressNodeList.Items {
		egressNodes = append(egressNodes, item.Name)
	}
	return
}

// CheckEgressNodeStatus check the status of the egressNode cr, parameter 'nodes' is the cluster's nodes name
func CheckEgressNodeStatus(f *framework.Framework, nodes []string, opt ...client.ListOption) {
	egressNodes, e := GetEgressNodes(f, opt...)
	Expect(e).NotTo(HaveOccurred())

	Expect(tools.IsSameSlice(egressNodes, nodes)).To(BeTrue())

	// get IP version
	enableV4, enableV6, e := GetIPVersion(f)
	Expect(e).NotTo(HaveOccurred())

	for _, node := range nodes {
		egressNodeObj := &egressv1.EgressTunnel{}
		e = GetEgressNode(f, node, egressNodeObj)
		Expect(e).NotTo(HaveOccurred())
		GinkgoWriter.Printf("egressNodeObj: %v\n", egressNodeObj)

		// check egressNode status
		status := egressNodeObj.Status
		// check phase
		Expect(status.Phase).To(Equal(egressv1.EgressNodeReady))
		// check physicalInterface
		Expect(CheckEgressNodeInterface(node, status.Tunnel.Parent.Name, time.Second*10)).To(BeTrue())
		// check mac
		Expect(CheckEgressNodeMac(node, status.Tunnel.MAC, time.Second*10)).To(BeTrue())

		if enableV4 {
			// check vxlan ip
			Expect(CheckEgressNodeIP(node, status.Tunnel.IPv4, time.Second*10)).To(BeTrue())
			// check node ip
			Expect(CheckNodeIP(node, status.Tunnel.Parent.Name, status.Tunnel.Parent.IPv4, time.Second*10)).To(BeTrue())
		}
		if enableV6 && !enableV4 {
			// check vxlan ip
			Expect(CheckEgressNodeIP(node, status.Tunnel.IPv6, time.Second*10)).To(BeTrue())
			// check node ip
			Expect(CheckNodeIP(node, status.Tunnel.Parent.Name, status.Tunnel.Parent.IPv6, time.Second*10)).To(BeTrue())
		}
		if enableV6 && enableV4 {
			// check vxlan ip
			Expect(CheckEgressNodeIP(node, status.Tunnel.IPv6, time.Second*10)).To(BeTrue())
		}
	}
}
