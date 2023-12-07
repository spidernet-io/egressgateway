// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/spidernet-io/egressgateway/pkg/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
)

func CheckDaemonSetEgressIP(
	ctx context.Context, cli client.Client,
	cfg *Config, egressConfig config.FileConfig,
	ds *appsv1.DaemonSet, ipv4, ipv6 string, expectUsedEip bool) error {

	list := &corev1.PodList{}
	labels := &metav1.LabelSelector{MatchLabels: ds.Spec.Template.Labels}
	selector, err := metav1.LabelSelectorAsSelector(labels)
	if err != nil {
		return err
	}
	err = cli.List(ctx, list, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     ds.Namespace,
	})
	if err != nil {
		return err
	}

	for _, pod := range list.Items {
		// check v4
		if egressConfig.EnableIPv4 {
			err = CheckPodEgressIP(ctx, cfg, pod, ipv4, cfg.ServerAIPv4, expectUsedEip)
			if err != nil {
				return err
			}
		}

		// check v6
		if egressConfig.EnableIPv6 {
			err = CheckPodEgressIP(ctx, cfg, pod, ipv6, cfg.ServerAIPv6, expectUsedEip)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type CmdError struct {
	EgressIP      string
	ServerIP      string
	ExpectUsedEip bool
	Cmd           string
	Output        string
	CmdError      error
	NodeList      string
	PodList       string
	PolicyList    string
	GatewayList   string
	PolicyYAML    string
	GatewayYAML   string
}

func (c CmdError) Error() string {
	return fmt.Sprintf("%v\n %s", c.CmdError, c.Output)
}

func debugPodList(config *Config) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	args := fmt.Sprintf("kubectl --kubeconfig %s get pods -o wide -A", config.KubeConfigPath)
	cmd := exec.CommandContext(ctx, "sh", "-c", args)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("get pod list: %s\n", err.Error())
	}
	return string(raw)
}

func CheckPodEgressIP(ctx context.Context, cfg *Config, pod corev1.Pod, egressIP string, serverIP string, expectUsedEip bool) error {
	cmd := generateCmd(ctx, cfg, pod, egressIP, serverIP, expectUsedEip)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return CmdError{
			EgressIP:      egressIP,
			ServerIP:      serverIP,
			ExpectUsedEip: expectUsedEip,
			Cmd:           cmd.String(),
			Output:        string(raw),
			CmdError:      err,
			PodList:       debugPodList(cfg),
		}
	} else {
		fmt.Println(cmd)
	}
	return nil
}

func generateCmd(ctx context.Context, config *Config, pod corev1.Pod, eip, serverIP string, expectUsedEip bool) *exec.Cmd {
	curlServer := fmt.Sprintf("nettools-client -addr %s -protocol %s -tcpPort %v -udpPort %v -webPort %v -eip %s -batch true",
		serverIP, config.Mod, config.TcpPort, config.UdpPort, config.WebPort, eip)
	if !expectUsedEip {
		curlServer = curlServer + " -contain false"
	}
	args := fmt.Sprintf("kubectl --kubeconfig %s exec %s -n %s -- %s", config.KubeConfigPath, pod.Name, pod.Namespace, curlServer)
	return exec.CommandContext(ctx, "sh", "-c", args)
}

// CheckPodsEgressIP check pods egressIP my pod-egressPolicy-map
func CheckPodsEgressIP(ctx context.Context, cfg *Config, p2p map[*corev1.Pod]*egressv1.EgressPolicy, checkv4, checkv6 bool, expectUsedEip bool) error {
	for pod, egp := range p2p {
		if checkv4 {
			if len(egp.Status.Eip.Ipv4) == 0 {
				return fmt.Errorf("failed get eipV4")
			}
			return CheckPodEgressIP(ctx, cfg, *pod, egp.Status.Eip.Ipv4, cfg.ServerAIPv4, expectUsedEip)
		}
		if checkv6 {
			if len(egp.Status.Eip.Ipv6) == 0 {
				return fmt.Errorf("failed get eipV6")
			}
			return CheckPodEgressIP(ctx, cfg, *pod, egp.Status.Eip.Ipv6, cfg.ServerAIPv6, expectUsedEip)
		}
	}
	return nil
}

func CheckDeployEgressIP(
	ctx context.Context, cli client.Client,
	cfg *Config, egressConfig config.FileConfig,
	deploy *appsv1.Deployment, ipv4, ipv6 string, expectUsedEip bool) error {

	list := &corev1.PodList{}
	labels := &metav1.LabelSelector{MatchLabels: deploy.Spec.Template.Labels}
	selector, err := metav1.LabelSelectorAsSelector(labels)
	if err != nil {
		return err
	}
	err = cli.List(ctx, list, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     deploy.Namespace,
	})
	if err != nil {
		return err
	}

	for _, pod := range list.Items {
		// check v4
		if egressConfig.EnableIPv4 {
			err = CheckPodEgressIP(ctx, cfg, pod, ipv4, cfg.ServerAIPv4, expectUsedEip)
			if err != nil {
				return err
			}
		}

		// check v6
		if egressConfig.EnableIPv6 {
			err = CheckPodEgressIP(ctx, cfg, pod, ipv6, cfg.ServerAIPv6, expectUsedEip)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
