// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	"sigs.k8s.io/controller-runtime/pkg/client"

	e2eerr "github.com/spidernet-io/egressgateway/test/e2e/err"
	"github.com/spidernet-io/egressgateway/test/e2e/tools"
)

func LabelNodes(ctx context.Context, cli client.Client, nodes []string, labels map[string]string) error {
	for _, nodeName := range nodes {
		err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			node := &corev1.Node{}
			err := cli.Get(ctx, types.NamespacedName{Name: nodeName}, node)
			if err != nil {
				return err
			}
			if node.Labels == nil {
				node.Labels = make(map[string]string)
			}
			for k, v := range labels {
				node.Labels[k] = v
			}
			return cli.Update(ctx, node)
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func UnLabelNodes(ctx context.Context, cli client.Client, nodes []string, labels map[string]string) error {
	for _, nodeName := range nodes {
		err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			node := &corev1.Node{}
			err := cli.Get(ctx, types.NamespacedName{Name: nodeName}, node)
			if err != nil {
				return err
			}
			for k := range labels {
				delete(node.Labels, k)
			}
			return cli.Update(ctx, node)
		})
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
		return fmt.Errorf("err: %v\nout: %v", err, string(out))
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
		return fmt.Errorf("docker start error: %v\nout: %v", err, string(out))
	}

	ctx, cancel := context.WithTimeout(ctx, poweronTimeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return e2eerr.ErrWaitNodeOnTimeout
		default:
			node, err := GetNode(ctx, cli, nodeName)
			if err != nil {
				return fmt.Errorf("failed to get node %s: %v", nodeName, err)
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
			return fmt.Errorf("failed to power on node %s: %v", node, err)
		}
	}
	return WaitAllPodRunning(ctx, cli, poweronTimeout)
}

// CollectClusterDebugInfo collects diagnostic information about the cluster state,
// including node status, pod status, disk usage, and describes for NotReady nodes
// and non-Running pods. Returns a single string to avoid truncation when printed.
func CollectClusterDebugInfo(ctx context.Context, cli client.Client) string {
	var sb strings.Builder
	timeout := time.Minute

	sb.WriteString("\n========== kubectl get node -o wide ==========\n")
	if out, err := tools.ExecCommand(ctx, "kubectl get node -o wide", timeout); err != nil {
		sb.WriteString(fmt.Sprintf("error: %v\n", err))
	} else {
		sb.Write(out)
	}

	sb.WriteString("\n========== kubectl get pod -A -o wide ==========\n")
	if out, err := tools.ExecCommand(ctx, "kubectl get pod -A -o wide", timeout); err != nil {
		sb.WriteString(fmt.Sprintf("error: %v\n", err))
	} else {
		sb.Write(out)
	}

	sb.WriteString("\n========== df -h ==========\n")
	if out, err := tools.ExecCommand(ctx, "df -h", timeout); err != nil {
		sb.WriteString(fmt.Sprintf("error: %v\n", err))
	} else {
		sb.Write(out)
	}

	// describe NotReady nodes
	nodeList := &corev1.NodeList{}
	if err := cli.List(ctx, nodeList); err == nil {
		for _, node := range nodeList.Items {
			if !CheckNodeStatus(&node, true) {
				sb.WriteString(fmt.Sprintf("\n========== kubectl describe node %s ==========\n", node.Name))
				if out, err := tools.ExecCommand(ctx, fmt.Sprintf("kubectl describe node %s", node.Name), timeout); err != nil {
					sb.WriteString(fmt.Sprintf("error: %v\n", err))
				} else {
					sb.Write(out)
				}
			}
		}
	}

	// describe non-Running pods
	podList := &corev1.PodList{}
	if err := cli.List(ctx, podList); err == nil {
		calicoNodeHosts := make(map[string]string)
		for _, pod := range podList.Items {
			if pod.Namespace == "calico-system" && strings.HasPrefix(pod.Name, "calico-node-") && pod.Spec.NodeName != "" && !isPodRunningOrCompleted(&pod) && pod.DeletionTimestamp == nil {
				calicoNodeHosts[pod.Spec.NodeName] = pod.Name
			}
			if !isPodRunningOrCompleted(&pod) && pod.DeletionTimestamp == nil {
				sb.WriteString(fmt.Sprintf("\n========== kubectl describe pod %s -n %s ==========\n", pod.Name, pod.Namespace))
				if out, err := tools.ExecCommand(ctx, fmt.Sprintf("kubectl describe pod %s -n %s", pod.Name, pod.Namespace), timeout); err != nil {
					sb.WriteString(fmt.Sprintf("error: %v\n", err))
				} else {
					sb.Write(out)
				}
				appendNonRunningContainerLogs(ctx, &sb, &pod, timeout)
			}
		}

		for nodeName, podName := range calicoNodeHosts {
			sb.WriteString(fmt.Sprintf("\n========== calico-node readiness %s on %s ==========\n", podName, nodeName))
			cmd := fmt.Sprintf("docker exec %s curl -s -w '\\n%%{http_code}\\n' http://127.0.0.1:9099/readiness", nodeName)
			if out, err := tools.ExecCommand(ctx, cmd, timeout); err != nil {
				sb.WriteString(fmt.Sprintf("error: %v\n", err))
			} else {
				sb.Write(out)
			}
		}
	}

	return sb.String()
}

func appendNonRunningContainerLogs(ctx context.Context, sb *strings.Builder, pod *corev1.Pod, timeout time.Duration) {
	appendContainerLogs := func(containerType string, status corev1.ContainerStatus) {
		if status.State.Running != nil {
			return
		}

		sb.WriteString(fmt.Sprintf("\n========== kubectl logs pod/%s -n %s -c %s (%s) ==========\n", pod.Name, pod.Namespace, status.Name, containerType))
		cmd := fmt.Sprintf("kubectl logs %s -n %s -c %s --tail=200", pod.Name, pod.Namespace, status.Name)
		if out, err := tools.ExecCommand(ctx, cmd, timeout); err != nil {
			sb.WriteString(fmt.Sprintf("error: %v\n", err))
		} else {
			sb.Write(out)
		}

		if status.RestartCount > 0 {
			sb.WriteString(fmt.Sprintf("\n========== kubectl logs pod/%s -n %s -c %s --previous (%s) ==========\n", pod.Name, pod.Namespace, status.Name, containerType))
			previousCmd := fmt.Sprintf("kubectl logs %s -n %s -c %s --previous --tail=200", pod.Name, pod.Namespace, status.Name)
			if out, err := tools.ExecCommand(ctx, previousCmd, timeout); err != nil {
				sb.WriteString(fmt.Sprintf("error: %v\n", err))
			} else {
				sb.Write(out)
			}
		}
	}

	for _, status := range pod.Status.InitContainerStatuses {
		appendContainerLogs("init", status)
	}

	for _, status := range pod.Status.ContainerStatuses {
		appendContainerLogs("container", status)
	}
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
