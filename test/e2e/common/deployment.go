// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spidernet-io/e2eframework/framework"
)

func GenerateDeployYaml(DeployName, NodeName, serverIP string, replicas int32, label map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: POD_NAMESPACE,
			Name:      DeployName,
			Labels:    label,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: label,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: label,
				},
				Spec: corev1.PodSpec{
					NodeName: NodeName,
					Containers: []corev1.Container{
						{
							Name:            DeployName,
							Image:           Env[IMAGE],
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{"bin/sh", "-c",
								fmt.Sprintf("sleep 10s && nettools-client -addr %s -protocol %s -tcpPort %s -udpPort %s -webPort %s", serverIP, Env[MOD], Env[TCP_PORT], Env[UDP_PORT], Env[WEB_PORT])},
						},
					},
				},
			},
		},
	}
}

func CreateClientPod(f *framework.Framework, DeployName, nodeName, serverIP string, replicas int32, label map[string]string, duration time.Duration) (deployment *appsv1.Deployment) {
	yaml := GenerateDeployYaml(DeployName, nodeName, serverIP, replicas, label)
	deployment, err := f.CreateDeploymentUntilReady(yaml, duration)
	Expect(err).NotTo(HaveOccurred())
	GinkgoWriter.Printf("deployment: %v\n", deployment)
	return
}

func CreateClientPodAndCheck(f *framework.Framework, eIP, deployName, nodeName, serverIP string, replicas int32, PodLabel map[string]string, expect bool, testTime, duration time.Duration) {
	GinkgoWriter.Printf("eIP: %v\n", eIP)
	deployment := CreateClientPod(f, deployName, nodeName, serverIP, replicas, PodLabel, duration)
	CheckEIP(f, deployment, eIP, expect, testTime, duration)
}

func CheckEIP(f *framework.Framework, deploy *appsv1.Deployment, eIP string, expect bool, testTime, duration time.Duration) {
	podList, err := f.GetDeploymentPodList(deploy)
	Expect(err).NotTo(HaveOccurred())
	Expect(podList).NotTo(BeNil())
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	time.Sleep(testTime)
	for _, pod := range podList.Items {
		// webSocket
		c := fmt.Sprintf("logs %s", pod.Name)
		b, err := f.ExecKubectl(c, ctx)
		Expect(err).NotTo(HaveOccurred())
		out := string(b)
		GinkgoWriter.Printf("out: %s\n", out)

		Expect(strings.Contains(out, UDP_CONNECTED)).To(BeTrue())
		Expect(strings.Contains(out, TCP_CONNECTED)).To(BeTrue())
		Expect(strings.Contains(out, WEB_CONNECTED)).To(BeTrue())

		if expect {
			Expect(strings.Contains(out, WEBSOCKET)).To(BeTrue())
			Expect(strings.Contains(out, UDP)).To(BeTrue())
			Expect(strings.Contains(out, TCP)).To(BeTrue())
			line := strings.Split(out, "\n")
			for _, l := range line {
				if strings.HasSuffix(l, WEBSOCKET) || strings.HasSuffix(l, UDP) || strings.HasSuffix(l, TCP) {
					GinkgoWriter.Println(l)
					Expect(strings.Contains(l, eIP)).To(BeTrue())
				}
			}
		} else {
			Expect(strings.Contains(out, eIP)).To(BeFalse())
		}
	}
}

func DeleteDeployIfExists(f *framework.Framework, deployName, nameSpace string, duration time.Duration, opts ...client.DeleteOption) error {
	if len(deployName) == 0 || len(nameSpace) == 0 {
		return INVALID_INPUT
	}
	deploy, err := f.GetDeployment(deployName, nameSpace)
	if err == nil && deploy != nil {
		return f.DeleteDeploymentUntilFinish(deployName, nameSpace, duration, opts...)
	}
	return nil
}
