// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"time"

	. "github.com/onsi/gomega"
	"github.com/spidernet-io/e2eframework/framework"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetEgressNode(f *framework.Framework, name string, egressNode *egressv1.EgressNode) error {
	key := client.ObjectKey{
		Name: name,
	}
	return f.GetResource(key, egressNode)
}

func ListEgressNodes(f *framework.Framework, opt ...client.ListOption) (*egressv1.EgressNodeList, error) {
	egressNodeList := &egressv1.EgressNodeList{}
	e := f.ListResource(egressNodeList, opt...)
	if e != nil {
		return nil, e
	}
	return egressNodeList, nil
}

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

func CheckEgressNodeStatus(f *framework.Framework, nodes []string, opt ...client.ListOption) {
	egressNodes, e := GetEgressNodes(f, opt...)
	Expect(e).NotTo(HaveOccurred())

	Expect(tools.IsSameSlice(egressNodes, nodes)).To(BeTrue())

	// get IP version
	enableV4, enableV6, e := GetIPVersion(f)
	Expect(e).NotTo(HaveOccurred())

	for _, node := range nodes {
		egressNodeObj := &egressv1.EgressNode{}
		e = GetEgressNode(f, node, egressNodeObj)
		Expect(e).NotTo(HaveOccurred())

		// check egressNode status
		status := egressNodeObj.Status
		// check phase
		Expect(status.Phase).To(Equal(egressv1.EgressNodeSucceeded))
		// check physicalInterface
		Expect(CheckEgressNodeInterface(node, status.PhysicalInterface, time.Second*10)).To(BeTrue())
		// check mac
		Expect(CheckEgressNodeMac(node, status.TunnelMac, time.Second*10)).To(BeTrue())

		if enableV4 {
			// check vxlan ip
			Expect(CheckEgressNodeIP(node, status.VxlanIPv4, time.Second*10)).To(BeTrue())
			// check node ip
			Expect(CheckEgressNodeIP(node, status.PhysicalInterfaceIPv4, time.Second*10)).To(BeTrue())
		}
		if enableV6 {
			// check vxlan ip
			Expect(CheckEgressNodeIP(node, status.VxlanIPv6, time.Second*10)).To(BeTrue())
			// check node ip
			Expect(CheckEgressNodeIP(node, status.PhysicalInterfaceIPv6, time.Second*10)).To(BeTrue())
		}
	}
}
