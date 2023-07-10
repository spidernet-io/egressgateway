// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/e2eframework/framework"

	"github.com/spidernet-io/egressgateway/pkg/utils"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
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

func GetAllNodesIP(f *framework.Framework) (nodesIPv4, nodesIPv6 []string) {
	nodeList, err := f.GetNodeList()
	Expect(err).NotTo(HaveOccurred())
	nodesIPv4, nodesIPv6 = make([]string, 0), make([]string, 0)
	nodeIPv4Map, nodeIPv6Map := make(map[string]struct{}), make(map[string]struct{})
	for _, node := range nodeList.Items {
		ipv4, ipv6 := utils.GetNodeIP(&node)
		if len(ipv4) != 0 {
			nodeIPv4Map[ipv4] = struct{}{}
		}
		if len(ipv6) != 0 {
			nodeIPv6Map[ipv6] = struct{}{}
		}
	}
	for ip := range nodeIPv4Map {
		nodesIPv4 = append(nodesIPv4, ip)
	}
	for ip := range nodeIPv6Map {
		nodesIPv6 = append(nodesIPv6, ip)
	}
	return
}

func PowerOffNodeUntilNotReady(f *framework.Framework, nodeName string, timeout time.Duration) error {
	c := fmt.Sprintf("docker stop %s", nodeName)
	out, err := tools.ExecCommand(c, timeout)
	GinkgoWriter.Printf("out: %s\n", out)
	Expect(err).NotTo(HaveOccurred())

	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ERR_TIMEOUT
		default:
			node, err := f.GetNode(nodeName)
			Expect(err).NotTo(HaveOccurred())
			down := f.CheckNodeStatus(node, false)
			if down {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func PowerOnNodeUntilReady(f *framework.Framework, nodeName string, timeout time.Duration) error {
	c := fmt.Sprintf("docker start %s", nodeName)
	_, err := tools.ExecCommand(c, timeout)
	Expect(err).NotTo(HaveOccurred())

	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ERR_TIMEOUT
		default:
			node, err := f.GetNode(nodeName)
			Expect(err).NotTo(HaveOccurred())
			up := f.CheckNodeStatus(node, true)
			if up {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func PowerOnNodesUntilClusterReady(f *framework.Framework, nodes []string, timeout time.Duration) error {
	for _, node := range nodes {
		err := PowerOnNodeUntilReady(f, node, timeout)
		if err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	return f.WaitAllPodUntilRunning(ctx)
}
