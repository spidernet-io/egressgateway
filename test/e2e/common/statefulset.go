// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"time"

	"github.com/spidernet-io/egressgateway/pkg/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	e2eerr "github.com/spidernet-io/egressgateway/test/e2e/err"
)

func CreateStatefulSet(ctx context.Context, cli client.Client, name string, image string, replicas int, timeout time.Duration) (*appsv1.StatefulSet, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var terminationGracePeriodSeconds int64 = 0

	label := map[string]string{"app": name}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
			Labels:    label,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  label,
		},
	}
	if err := cli.Create(ctx, svc); err != nil {
		return nil, err
	}

	res := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
			Labels:    label,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            ptr.To(int32(replicas)),
			ServiceName:         svc.Name,
			Selector:            &metav1.LabelSelector{MatchLabels: label},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: label},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers: []corev1.Container{{
						Name:            name,
						Image:           image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         []string{"/bin/sh", "-c", "sleep infinity"},
					}},
				},
			},
		},
	}

	if err := cli.Create(ctx, res); err != nil {
		_ = DeleteObj(context.Background(), cli, svc)
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			_ = DeleteObj(context.Background(), cli, res)
			_ = DeleteObj(context.Background(), cli, svc)
			return nil, e2eerr.ErrTimeout
		default:
			err := cli.Get(ctx, types.NamespacedName{Namespace: res.Namespace, Name: res.Name}, res)
			if err != nil {
				return nil, err
			}

			a := *res.Spec.Replicas
			b := res.Status.ReadyReplicas

			if a == b && b > 0 {
				return res, nil
			}

			time.Sleep(time.Second / 2)
		}
	}
}

func CheckStatefulSetEgressIP(
	ctx context.Context, cli client.Client,
	cfg *Config, egressConfig config.FileConfig,
	sts *appsv1.StatefulSet, ipv4, ipv6 string, expectUsedEip bool) error {

	list := &corev1.PodList{}
	labels := &metav1.LabelSelector{MatchLabels: sts.Spec.Template.Labels}
	selector, err := metav1.LabelSelectorAsSelector(labels)
	if err != nil {
		return err
	}
	err = cli.List(ctx, list, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     sts.Namespace,
	})
	if err != nil {
		return err
	}

	for _, pod := range list.Items {
		if egressConfig.EnableIPv4 {
			err = CheckPodEgressIP(ctx, cfg, pod, ipv4, cfg.ServerAIPv4, expectUsedEip)
			if err != nil {
				return err
			}
		}

		if egressConfig.EnableIPv6 {
			err = CheckPodEgressIP(ctx, cfg, pod, ipv6, cfg.ServerAIPv6, expectUsedEip)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
