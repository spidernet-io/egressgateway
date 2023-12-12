// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"net"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	e2eerr "github.com/spidernet-io/egressgateway/test/e2e/err"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

func LabelNodes(ctx context.Context, cli client.Client, nodes []string, labels map[string]string) error {
	for _, nodeName := range nodes {
		node := &corev1.Node{}
		err := cli.Get(ctx, types.NamespacedName{Name: nodeName}, node)
		if err != nil {
			return err
		}
		for k, v := range labels {
			node.Labels[k] = v
		}
		err = cli.Update(ctx, node)
		if err != nil {
			return err
		}
	}
	return nil
}

func UnLabelNodes(ctx context.Context, cli client.Client, nodes []string, labels map[string]string) error {
	for _, nodeName := range nodes {
		node := &corev1.Node{}
		err := cli.Get(ctx, types.NamespacedName{Name: nodeName}, node)
		if err != nil {
			return err
		}
		l := node.Labels
		for k := range labels {
			delete(l, k)
		}
		node.Labels = l
		err = cli.Update(ctx, node)
		if err != nil {
			return err
		}
	}
	return nil
}

func PowerOffNodeUntilNotReady(ctx context.Context, cli client.Client, nodeName string, execTimeout, poweroffTimeout time.Duration) error {
	c := fmt.Sprintf("docker stop %s", nodeName)
	out, err := tools.ExecCommand(ctx, c, execTimeout)
	if err != nil {
		return fmt.Errorf("err: %v\nout: %v\n", err, string(out))
	}

	ctx, cancel := context.WithTimeout(ctx, poweroffTimeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return e2eerr.ErrTimeout
		default:
			node, err := GetNode(ctx, cli, nodeName)
			if err != nil {
				return err
			}
			down := CheckNodeStatus(node, false)
			if down {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func PowerOnNodeUntilReady(ctx context.Context, cli client.Client, nodeName string, execTimeout, poweronTimeout time.Duration) error {
	c := fmt.Sprintf("docker start %s", nodeName)
	out, err := tools.ExecCommand(ctx, c, execTimeout)
	if err != nil {
		return fmt.Errorf("err: %v\nout: %v\n", err, string(out))
	}

	ctx, cancel := context.WithTimeout(ctx, poweronTimeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return e2eerr.ErrTimeout
		default:
			node, err := GetNode(ctx, cli, nodeName)
			if err != nil {
				return err
			}
			up := CheckNodeStatus(node, true)
			if up {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func PowerOnNodesUntilClusterReady(ctx context.Context, cli client.Client, nodes []string, execTimeout, poweronTimeout time.Duration) error {
	for _, node := range nodes {
		err := PowerOnNodeUntilReady(ctx, cli, node, execTimeout, poweronTimeout)
		if err != nil {
			return err
		}
	}
	return WaitAllPodRunning(ctx, cli, poweronTimeout)
}

func GetNodeIP(node *corev1.Node) (string, string) {
	var ipv4, ipv6 string
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			if ip := net.ParseIP(address.Address); ip != nil {
				if ip.To4() != nil {
					ipv4 = address.Address
				} else if ip.To16() != nil {
					ipv6 = address.Address
				}
			}
		}
	}
	return ipv4, ipv6
}

func CheckNodeStatus(node *corev1.Node, expectReady bool) bool {

	unreachTaintTemp := &corev1.Taint{
		Key:    corev1.TaintNodeUnreachable,
		Effect: corev1.TaintEffectNoExecute,
	}
	notReadyTaintTemp := &corev1.Taint{
		Key:    corev1.TaintNodeNotReady,
		Effect: corev1.TaintEffectNoExecute,
	}
	for _, cond := range node.Status.Conditions {
		// check whether the ready host have taints
		if cond.Type == corev1.NodeReady {
			haveTaints := false
			taints := node.Spec.Taints
			for _, t := range taints {
				if t.MatchTaint(unreachTaintTemp) || t.MatchTaint(notReadyTaintTemp) {
					haveTaints = true
					break
				}
			}
			if expectReady {
				if (cond.Status == corev1.ConditionTrue) && !haveTaints {
					return true
				}
				return false
			}
			if cond.Status != corev1.ConditionTrue {
				return true
			}
			return false
		}
	}
	return false
}

func GetNode(ctx context.Context, cli client.Client, nodeName string) (*corev1.Node, error) {
	node := new(corev1.Node)
	err := cli.Get(ctx, types.NamespacedName{Name: nodeName}, node)
	if err != nil {
		return nil, err
	}
	return node, nil
}
