// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"net"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
)

func GenIPPools(ctx context.Context, cli client.Client, enableIPv4, enableIPv6 bool, num int64, increase uint8) (egressv1.Ippools, error) {
	res := egressv1.Ippools{}
	if num < 1 {
		return res, fmt.Errorf("the parameter `num` cannot be less then 1")
	}
	num = num - 1
	ipv4, ipv6, err := getRandomEgressIPByNode(ctx, cli, increase)
	if err != nil {
		return res, err
	}
	if len(ipv4) != 0 && enableIPv4 {
		if num == 0 {
			res.IPv4 = []string{ipv4}
		} else {
			end, err := AddIP(ipv4, num)
			if err != nil {
				return res, err
			}
			res.IPv4 = append(res.IPv4, ipv4+"-"+end)
		}
	}
	if len(ipv6) != 0 && enableIPv6 {
		if num == 0 {
			res.IPv6 = []string{ipv6}
		} else {
			end, err := AddIP(ipv6, num)
			if err != nil {
				return res, err
			}
			res.IPv6 = append(res.IPv6, ipv6+"-"+end)
		}
	}
	return res, nil
}

// getRandomEgressIPByNode returns a random IPv4 and IPv6 egress IP address
// based on the node's IP addresses.
func getRandomEgressIPByNode(ctx context.Context, cli client.Client, increase uint8) (ipv4, ipv6 string, err error) {
	// Get all IPv4 and IPv6 addresses assigned to the node
	ipv4List, ipv6List, err := GetAllNodesIPNew(ctx, cli)
	if err != nil {
		return "", "", err
	}

	// If IPv4 addresses exist, generate a random one
	if len(ipv4List) != 0 {
		nodeIP := ipv4List[0]
		ipObj := net.ParseIP(nodeIP)
		ipObj[14] += increase
		ipv4 = ipObj.String()
	}

	// If IPv6 addresses exist, generate a random one
	if len(ipv6List) != 0 {
		nodeIp := ipv6List[0]
		ipObj := net.ParseIP(nodeIp)
		ipObj[14] += increase
		ipv6 = ipObj.String()
	}

	return
}

func GetAllNodesIPNew(ctx context.Context, cli client.Client) (ipv4List, ipv6List []string, err error) {
	nodes := &corev1.NodeList{}
	err = cli.List(ctx, nodes)
	if err != nil {
		return nil, nil, err
	}

	for _, node := range nodes.Items {
		for _, address := range node.Status.Addresses {
			switch address.Type {
			case corev1.NodeInternalIP:
				if ip := net.ParseIP(address.Address); ip != nil {
					if ip.To4() != nil {
						ipv4List = append(ipv4List, address.Address)
					} else if ip.To16() != nil {
						ipv6List = append(ipv6List, address.Address)
					}
				}
			}
		}
	}
	return
}
