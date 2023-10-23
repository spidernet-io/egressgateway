// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/e2eframework/framework"
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
			return fmt.Errorf("power off node timeout")
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
			return fmt.Errorf("power on node timeout")
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
