// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	"github.com/spidernet-io/e2eframework/framework"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/test/e2e/err"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GenerateEgressGatewayYaml(name string, matchLabels map[string]string) *egressv1.EgressGateway {
	return &egressv1.EgressGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: egressv1.EgressGatewaySpec{
			NodeSelector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
	}
}

func GetEgressGateway(f *framework.Framework, name string, gateway *egressv1.EgressGateway) error {
	key := client.ObjectKey{
		Name: name,
	}
	return f.GetResource(key, gateway)
}

func CreateEgressGateway(f *framework.Framework, gateway *egressv1.EgressGateway, opts ...client.CreateOption) error {
	return f.CreateResource(gateway, opts...)
}

func DeleteEgressGateway(f *framework.Framework, gateway *egressv1.EgressGateway, opts ...client.DeleteOption) error {
	return f.DeleteResource(gateway, opts...)
}

func EditEgressGatewayMatchLabels(f *framework.Framework, gateway *egressv1.EgressGateway, labels map[string]string, opts ...client.UpdateOption) error {
	gateway.Spec.NodeSelector.MatchLabels = labels
	return f.UpdateResource(gateway, opts...)
}

// DeleteEgressGatewayIfExists delete egressgateway if its exists
func DeleteEgressGatewayIfExists(f *framework.Framework, name string, duration time.Duration) error {
	gateway := new(egressv1.EgressGateway)
	e := GetEgressGateway(f, name, gateway)
	if e == nil {
		return DeleteEgressGatewayUntilFinish(f, gateway, duration)
	}
	return nil
}

func DeleteEgressGatewayUntilFinish(f *framework.Framework, gateway *egressv1.EgressGateway, duration time.Duration, opts ...client.DeleteOption) error {
	e := DeleteEgressGateway(f, gateway, opts...)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return err.TIME_OUT
		default:
			e = GetEgressGateway(f, gateway.Name, gateway)
			if e != nil {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func WaitEgressGatewayUpdatedStatus(f *framework.Framework, name string, expectNodes []string, duration time.Duration) (gateway *egressv1.EgressGateway, e error) {
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	gateway = new(egressv1.EgressGateway)
	var nodes []string
	for {
		select {
		case <-ctx.Done():
			return nil, err.TIME_OUT
		default:
			e = GetEgressGateway(f, name, gateway)
			if e != nil {
				return nil, e
			}
			if len(gateway.Status.NodeList) == len(expectNodes) {
				for _, node := range gateway.Status.NodeList {
					nodes = append(nodes, node.Name)
				}
				if tools.IsSameSlice(nodes, expectNodes) {
					return gateway, nil
				}
			}
			time.Sleep(time.Second)
		}
	}
}

func WaitEgressGatewayUpdatedMatchLabels(f *framework.Framework, name string, labels map[string]string, duration time.Duration) (gateway *egressv1.EgressGateway, e error) {
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	gateway = new(egressv1.EgressGateway)
	for {
		select {
		case <-ctx.Done():
			return nil, err.TIME_OUT
		default:
			e = GetEgressGateway(f, name, gateway)
			if e != nil {
				return nil, e
			}
			l := gateway.Spec.NodeSelector.MatchLabels
			bingo := true
			for k, v := range l {
				if labels[k] != v {
					bingo = false
					time.Sleep(time.Second)
					break
				}
			}
			if bingo {
				return gateway, nil
			}
		}
	}
}

func CheckEgressGatewayNodeList(f *framework.Framework, gateway *egressv1.EgressGateway, matchedNodes []string) {
	// if have no matched nodes, then egressgateway status.nodelist is empty
	if len(matchedNodes) == 0 {
		Expect(gateway.Status.NodeList).To(BeEmpty())
	} else {
		nodesStatus := map[string]egressv1.SelectedEgressNode{}
		for _, node := range gateway.Status.NodeList {
			nodesStatus[node.Name] = node
		}

		// check if egressgateway status.nodelist is equal with real matched ready nodes
		Expect(len(gateway.Status.NodeList)).To(Equal(len(matchedNodes)))

		var gatewayNodes []string
		for _, node := range gateway.Status.NodeList {
			gatewayNodes = append(gatewayNodes, node.Name)
		}
		Expect(tools.IsSameSlice(gatewayNodes, matchedNodes)).To(BeTrue())

		// check filed "Ready" "Active"
		for _, nodeName := range matchedNodes {
			node, e := f.GetNode(nodeName)
			Expect(e).NotTo(HaveOccurred())
			if f.CheckNodeStatus(node, true) {
				Expect(nodesStatus[nodeName].Ready).To(BeTrue())
			} else {
				Expect(nodesStatus[nodeName].Ready).To(BeFalse())
			}
		}
	}
}

func GetEgressGatewayIPsV4(f *framework.Framework, name string) ([]string, error) {
	gateway := &egressv1.EgressGateway{}
	e := GetEgressGateway(f, name, gateway)
	if e != nil {
		return nil, e
	}
	var eIPs []string
	for _, node := range gateway.Status.NodeList {
		if node.Active {
			n, e := f.GetNode(node.Name)
			if e != nil {
				return nil, e
			}
			eIPs = append(eIPs, n.Status.Addresses[0].Address)
		}
	}
	return eIPs, nil
}

func GetEgressGatewayIPsV6(f *framework.Framework, name string) ([]string, error) {
	gateway := &egressv1.EgressGateway{}
	e := GetEgressGateway(f, name, gateway)
	if e != nil {
		return nil, e
	}
	var eIPs []string
	for _, node := range gateway.Status.NodeList {
		if node.Active {
			n, e := f.GetNode(node.Name)
			if e != nil {
				return nil, e
			}
			eIPs = append(eIPs, n.Status.Addresses[1].Address)
		}
	}
	return eIPs, nil
}
