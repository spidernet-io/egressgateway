// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreatePod(ctx context.Context, cli client.Client, image string) (*corev1.Pod, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	var terminationGracePeriodSeconds int64 = 0

	name := "pod-" + uuid.NewString()
	label := map[string]string{"app": name}
	res := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels:    label,
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
			Containers: []corev1.Container{
				{
					Name:            name,
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/bin/sh", "-c", "sleep infinity"},
				},
			},
		}}

	err := cli.Create(ctx, res)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			_ = DeleteObj(context.Background(), cli, res)
			return nil, fmt.Errorf("create Pod time out")
		default:
			err := cli.Get(ctx, types.NamespacedName{Namespace: res.Namespace, Name: res.Name}, res)
			if err != nil {
				return nil, err
			}

			if res.Status.Phase == corev1.PodRunning {
				return res, nil
			}

			time.Sleep(time.Second / 2)
		}
	}
}

func CreatePodCustom(ctx context.Context, cli client.Client, name, image string, setUp func(pod *corev1.Pod)) (*corev1.Pod, error) {
	var terminationGracePeriodSeconds int64 = 0

	res := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
			Containers: []corev1.Container{
				{
					Name:            name,
					Image:           image,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command:         []string{"/bin/sh", "-c", "sleep infinity"},
				},
			},
		}}

	setUp(res)

	err := cli.Create(ctx, res)
	if err != nil {
		return nil, fmt.Errorf("error:\n%w\npod yaml:\n%s\n", err, GetObjYAML(res))
	}
	return res, nil
}

// CreatePods create pods by gaven number "n"
func CreatePods(ctx context.Context, cli client.Client, img string, n int) []*corev1.Pod {
	var res []*corev1.Pod
	for i := 0; i < n; {
		pod, err := CreatePod(ctx, cli, img)
		if err != nil {
			continue
		}
		res = append(res, pod)
		i++
	}
	return res
}

func WaitPodRunning(ctx context.Context, cli client.Client, pod *corev1.Pod, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var e error

	for {
		select {
		case <-ctx.Done():
			if e != nil {
				return fmt.Errorf("timeout to wait the pod running, error: %v", e)
			}
			return fmt.Errorf("timeout to wait the pod running")

		default:
			err := cli.Get(ctx, types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}, pod)
			if err != nil {
				e = err
				time.Sleep(time.Second)
				continue
			}
			if pod.Status.Phase == corev1.PodRunning {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}
