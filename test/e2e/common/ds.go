// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateDaemonSet(ctx context.Context, cli client.Client, name string, image string) (*appsv1.DaemonSet, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	var terminationGracePeriodSeconds int64 = 0

	label := map[string]string{"app": name}
	res := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
			Labels:    label,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: label},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: label},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers: []corev1.Container{{Name: name,
						Image:           image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         []string{"/bin/sh", "-c", "sleep infinity"},
					}},
				},
			},
		},
	}

	err := cli.Create(ctx, res)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			_ = DeleteObj(context.Background(), cli, res)
			return nil, fmt.Errorf("create DaemonSet time out")
		default:
			err := cli.Get(ctx, types.NamespacedName{Namespace: res.Namespace, Name: res.Name}, res)
			if err != nil {
				return nil, err
			}

			a := res.Status.CurrentNumberScheduled
			b := res.Status.DesiredNumberScheduled
			c := res.Status.NumberAvailable

			if a == b && b == c && a > 0 {
				return res, nil
			}

			time.Sleep(time.Second / 2)
		}
	}
}

func GetDaemonSetPodIPs(ctx context.Context, cli client.Client, ds *appsv1.DaemonSet) (ipv4List, ipv6List []string, err error) {
	podList := new(corev1.PodList)
	listOps := client.MatchingLabels(ds.Spec.Selector.MatchLabels)
	err = cli.List(ctx, podList, listOps)
	if err != nil {
		return nil, nil, err
	}
	ipv4List, ipv6List = GetPodListIPs(podList)
	return ipv4List, ipv6List, nil
}
