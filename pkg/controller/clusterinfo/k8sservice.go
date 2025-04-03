// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package clusterinfo

import (
	"context"
	"fmt"

	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
)

// kubeControllerManagerPodLabelList Store the labels for the Kubernetes Controller Manager.
// We use these labels to retrieve the Kubernetes Controller Manager Pod arguments in order
// to obtain the cluster CIDR. Since different clusters have different Kubernetes Controller
// Manager labels, the content of the labels also varies.
var kubeControllerManagerPodLabelList = []map[string]string{
	{
		"component": "kube-controller-manager",
	},
	{
		"k8s-app": "kube-controller-manager",
	},
}

func GetClusterCIDR(ctx context.Context, cli client.Client) (ipv4, ipv6 []string, err error) {
	for _, item := range kubeControllerManagerPodLabelList {
		pods, err := listPodByLabel(ctx, cli, item)
		if err != nil {
			return nil, nil, err
		}
		if len(pods) < 1 {
			continue
		}
		return parseCIDRFromControllerManager(&pods[0], "--service-cluster-ip-range=")
	}
	return nil, nil, nil
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
	return pods, nil
}

func parseCIDRFromControllerManager(pod *corev1.Pod, param string) (ipv4, ipv6 []string, err error) {
	containers := pod.Spec.Containers
	if len(containers) == 0 {
		return nil, nil, fmt.Errorf("failed to found containers")
	}
	container := containers[0]
	cmdAndArgs := append(container.Command, container.Args...)
	ipRange := ""
	for _, c := range cmdAndArgs {
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
