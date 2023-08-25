// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
)

// ParseCidrFromControllerManager get cidr value from kube controller manager
func ParseCidrFromControllerManager(pod *corev1.Pod, param string) (ipv4Range, ipv6Range []string, err error) {
	containers := pod.Spec.Containers
	if len(containers) == 0 {
		return nil, nil, fmt.Errorf("failed to found containers")
	}
	commands := containers[0].Command
	ipRange := ""
	for _, c := range commands {
		if strings.Contains(c, param) {
			ipRange = strings.Split(c, "=")[1]
			break
		}
	}
	if len(ipRange) == 0 {
		return nil, nil, fmt.Errorf("failed to found %s", param)
	}
	// get cidr
	ipRanges := strings.Split(ipRange, ",")
	if len(ipRanges) == 1 {
		if isV4, _ := ip.IsIPv4Cidr(ipRanges[0]); isV4 {
			ipv4Range = ipRanges
			ipv6Range = []string{}
		}
		if isV6, _ := ip.IsIPv6Cidr(ipRanges[0]); isV6 {
			ipv6Range = ipRanges
			ipv4Range = []string{}

		}
	}
	if len(ipRanges) == 2 {
		ipv4Range, ipv6Range = ipRanges[:1], ipRanges[1:]
	}
	return
}

// GetPodByLabel get pod by label
func GetPodByLabel(c client.Client, label map[string]string) (*corev1.Pod, error) {
	podList := corev1.PodList{}
	opts := client.MatchingLabels(label)
	err := c.List(context.Background(), &podList, opts)
	if err != nil {
		return nil, err
	}
	pods := podList.Items
	if len(pods) == 0 {
		return nil, fmt.Errorf("failed to get pod")
	}
	return &pods[0], nil
}

// GetClusterCidr get k8s default podCidr
func GetClusterCidr(c client.Client) (ipv4Range, ipv6Range []string, err error) {
	pod, err := GetPodByLabel(c, kubeControllerManagerPodLabel)
	if err != nil {
		return nil, nil, err
	}
	return ParseCidrFromControllerManager(pod, clusterCidr)
}
