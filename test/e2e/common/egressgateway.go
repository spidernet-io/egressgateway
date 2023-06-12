// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/spidernet-io/e2eframework/framework"
	"github.com/spidernet-io/egressgateway/test/e2e/err"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
			// gateway
			GinkgoWriter.Printf("gateway: %v\n", gateway)
			if e != nil {
				return nil, e
			}
			// expectNodes
			GinkgoWriter.Printf("expectNodes: %v\n", expectNodes)
			for _, node := range gateway.Status.NodeList {
				GinkgoWriter.Printf("node: %v\n", node.Name)
			}
			if len(gateway.Status.NodeList) == len(expectNodes) {
				nodes = []string{}
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
