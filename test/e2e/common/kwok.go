// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GenerateKwokNodeYaml(n int) *corev1.Node {
	node := &corev1.Node{
		TypeMeta: v1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Node",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: "kwok-node-" + strconv.Itoa(n),
			Annotations: map[string]string{
				"node.alpha.kubernetes.io/ttl": "0",
				"kwok.x-k8s.io/node":           "fake",
			},
			Labels: map[string]string{
				"beta.kubernetes.io/arch":       "amd64",
				"beta.kubernetes.io/os":         "linux",
				"kubernetes.io/arch":            "amd64",
				"kubernetes.io/hostname":        "kwok-node-" + strconv.Itoa(n),
				"kubernetes.io/os":              "linux",
				"kubernetes.io/role":            "agent",
				"node-role.kubernetes.io/agent": "",
				"type":                          "kwok",
			},
		},
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{
				// KwokNodeTaint,
			},
		},
		Status: corev1.NodeStatus{
			Phase: "Running",
		},
	}
	for k, v := range KwokNodeLabel {
		node.Labels[k] = v
	}
	return node
}

func CreateKwokNodes(ctx context.Context, cli client.Client, n int) error {
	for i := 0; i < n; i++ {
		err := cli.Create(ctx, GenerateKwokNodeYaml(i))
		if err != nil {
			return err
		}
	}
	return nil
}

func GetKwokNodes(ctx context.Context, cli client.Client) (*corev1.NodeList, error) {
	nodeList := new(corev1.NodeList)
	err := cli.List(ctx, nodeList, client.MatchingLabels(KwokNodeLabel))
	if err != nil {
		return nil, err
	}
	return nodeList, nil
}

func DeleteKwokNodes(ctx context.Context, cli client.Client, nodes *corev1.NodeList) error {
	for _, node := range nodes.Items {
		err := DeleteObj(ctx, cli, &node)
		if err != nil {
			return err
		}
	}
	return nil
}
