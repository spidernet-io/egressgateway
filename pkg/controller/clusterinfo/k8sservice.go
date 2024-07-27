// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package clusterinfo

import (
	"context"
	"fmt"
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

var kubeControllerManagerPodLabel = map[string]string{"component": "kube-controller-manager"}

func GetClusterCIDR(ctx context.Context, cli client.Client) (ipv4, ipv6 []string, err error) {
	pods, err := listPodByLabel(ctx, cli, kubeControllerManagerPodLabel)
	if err != nil {
		return nil, nil, err
	}
	return parseCIDRFromControllerManager(&pods[0], "--service-cluster-ip-range=")
}

func listPodByLabel(ctx context.Context, cli client.Client,
	label map[string]string) ([]corev1.Pod, error) {
	podList := new(corev1.PodList)
	opts := client.MatchingLabels(label)
	err := cli.List(ctx, podList, opts)
	if err != nil {
		return nil, err
	}
	pods := podList.Items
	if len(pods) == 0 {
		return nil, fmt.Errorf("failed to get pod")
	}
	return pods, nil
}

func parseCIDRFromControllerManager(pod *corev1.Pod, param string) (ipv4, ipv6 []string, err error) {
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
	ipRanges := strings.Split(ipRange, ",")
	if len(ipRanges) == 1 {
		if isV4, _ := ip.IsIPv4Cidr(ipRanges[0]); isV4 {
			ipv4 = ipRanges
			ipv6 = []string{}
		}
		if isV6, _ := ip.IsIPv6Cidr(ipRanges[0]); isV6 {
			ipv6 = ipRanges
			ipv4 = []string{}
		}
	}
	if len(ipRanges) == 2 {
		ipv4, ipv6 = ipRanges[:1], ipRanges[1:]
	}
	return
}
