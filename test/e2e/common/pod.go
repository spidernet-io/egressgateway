// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-faker/faker/v4"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreatePod(ctx context.Context, cli client.Client, image string) (*corev1.Pod, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	var terminationGracePeriodSeconds int64 = 0

	name := faker.Word() + "-" + strings.ToLower(faker.FirstName()) + "-" + strings.ToLower(faker.FirstName())
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
