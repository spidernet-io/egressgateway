// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/e2eframework/framework"
	egressv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/err"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

func GenerateEgressGatewayYaml(name string, ipPools egressv1beta1.Ippools, nodeSelector egressv1beta1.NodeSelector) *egressv1beta1.EgressGateway {
	return &egressv1beta1.EgressGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: egressv1beta1.EgressGatewaySpec{
			Ippools:      ipPools,
			NodeSelector: nodeSelector,
		},
	}
}

func GetEgressGateway(f *framework.Framework, name string, gateway *egressv1beta1.EgressGateway) error {
	key := client.ObjectKey{
		Name: name,
	}
	return f.GetResource(key, gateway)
}

func CreateEgressGateway(f *framework.Framework, gateway *egressv1beta1.EgressGateway, opts ...client.CreateOption) error {
	return f.CreateResource(gateway, opts...)
}

func DeleteEgressGateway(f *framework.Framework, gateway *egressv1beta1.EgressGateway, opts ...client.DeleteOption) error {
	return f.DeleteResource(gateway, opts...)
}

// DeleteEgressGatewayIfExists delete egressGateway if its exists
func DeleteEgressGatewayIfExists(f *framework.Framework, name string, duration time.Duration) error {
	gateway := new(egressv1beta1.EgressGateway)
	e := GetEgressGateway(f, name, gateway)
	if e == nil {
		return DeleteEgressGatewayUntilFinish(f, gateway, duration)
	}
	if errors.IsNotFound(e) {
		return nil
	}
	return e
}

func DeleteEgressGatewayUntilFinish(f *framework.Framework, gateway *egressv1beta1.EgressGateway, duration time.Duration, opts ...client.DeleteOption) error {
	e := DeleteEgressGateway(f, gateway, opts...)
	if errors.IsNotFound(e) {
		return nil
	}
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
			if errors.IsNotFound(e) {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func WaitEgressGatewayUpdatedStatus(f *framework.Framework, name string, expectNodes []string, duration time.Duration) (gateway *egressv1beta1.EgressGateway, e error) {
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	gateway = new(egressv1beta1.EgressGateway)
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

// GenerateSingleEgressGatewayIPPools generate single-format `EgressGatewaySpec.Ipools` by kind node ip, for e2e test
func GenerateSingleEgressGatewayIPPools(f *framework.Framework) egressv1beta1.Ippools {
	ipv4s, ipv6s := make([]string, 0), make([]string, 0)
	ipv4, ipv6 := generateIPbyKindNode(f)
	if len(ipv4) != 0 {
		ipv4s = append(ipv4s, ipv4)
	}
	if len(ipv6) != 0 {
		ipv6s = append(ipv6s, ipv6)
	}
	return egressv1beta1.Ippools{IPv4: ipv4s, IPv6: ipv6s}
}

// GenerateRangeEgressGatewayIPPools generate range-format `EgressGatewaySpec.Ipools` by kind node ip, for e2e test
func GenerateRangeEgressGatewayIPPools(f *framework.Framework, x int64) egressv1beta1.Ippools {
	ipv4s, ipv6s := make([]string, 0), make([]string, 0)
	ipv4, ipv6 := generateIPbyKindNode(f)
	if len(ipv4) != 0 {
		end, e := AddIP(ipv4, x)
		Expect(e).NotTo(HaveOccurred())
		ipv4s = append(ipv4s, fmt.Sprintf("%s-%s", ipv4, end))
	}
	if len(ipv6) != 0 {
		end, e := AddIP(ipv6, x)
		Expect(e).NotTo(HaveOccurred())
		ipv6s = append(ipv6s, fmt.Sprintf("%s-%s", ipv6, end))
	}
	return egressv1beta1.Ippools{IPv4: ipv4s, IPv6: ipv6s}
}

// generateIPbyKindNode generate ip by kind node ip, for e2e test
func generateIPbyKindNode(f *framework.Framework) (ipv4, ipv6 string) {
	nodeIpv4s, nodeIpv6s := GetAllNodesIP(f)
	if len(nodeIpv4s) != 0 {
		nodeIp := nodeIpv4s[0]
		ip := strings.Split(nodeIp, ".")
		ip[2] = "1"
		ipv4 = strings.Join(ip, ".")
	}
	if len(nodeIpv6s) != 0 {
		nodeIp := nodeIpv6s[0]
		ip := strings.Split(nodeIp, ":")
		ip[4] = "a:"
		ipv6 = strings.Join(ip, ":")
	}
	return
}
