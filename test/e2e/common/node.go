// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"github.com/spidernet-io/e2eframework/framework"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetAllNodes(f *framework.Framework) (nodes []string, err error) {
	nodelist, err := f.GetNodeList()
	if err != nil {
		return nil, err
	}
	for _, node := range nodelist.Items {
		nodes = append(nodes, node.Name)
	}
	return nodes, nil
}

func GetNodesByMatchLabels(f *framework.Framework, matchLabels map[string]string) (nodes []string, err error) {
	nodeList := &corev1.NodeList{}
	err = f.ListResource(nodeList, client.MatchingLabels(matchLabels))
	if err != nil {
		return nil, err
	}
	for _, item := range nodeList.Items {
		nodes = append(nodes, item.Name)
	}
	return
}

func GetUnmatchedNodes(f *framework.Framework, matchedNodes []string) (nodes []string, err error) {
	nodes, err = GetAllNodes(f)
	if err != nil {
		return nil, err
	}
	nodes = tools.SubtractionSlice(nodes, matchedNodes)
	return nodes, nil
}

func LabelNodes(f *framework.Framework, nodes []string, labels map[string]string) error {
	for _, nodeName := range nodes {
		node, err := f.GetNode(nodeName)
		if err != nil {
			return err
		}
		for k, v := range labels {
			node.Labels[k] = v
		}
		node.SetLabels(node.Labels)
		err = f.UpdateResource(node)
		if err != nil {
			return err
		}
	}
	return nil
}

func UnLabelNodes(f *framework.Framework, nodes []string, labels map[string]string) error {
	for _, nodeName := range nodes {
		node, err := f.GetNode(nodeName)
		if err != nil {
			return err
		}
		nodeLabels := node.Labels
		for k := range labels {
			delete(nodeLabels, k)
		}
		node.SetLabels(nodeLabels)
		err = f.UpdateResource(node)
		if err != nil {
			return err
		}
	}
	return nil
}
