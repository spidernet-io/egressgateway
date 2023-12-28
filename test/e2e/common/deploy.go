// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	e2eerr "github.com/spidernet-io/egressgateway/test/e2e/err"
)

func CreateDeploy(ctx context.Context, cli client.Client, name string, image string, repolicas int, timeout time.Duration) (*appsv1.Deployment, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var terminationGracePeriodSeconds int64 = 0

	label := map[string]string{"app": name}
	res := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
			Labels:    label,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](int32(repolicas)),
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

func WaitDeployDeleted(ctx context.Context, cli client.Client, deploy *appsv1.Deployment, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := DeleteObj(ctx, cli, deploy)
	if err != nil {
		if apiserrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	dp := new(appsv1.Deployment)

	for {
		select {
		case <-ctx.Done():
			return e2eerr.ErrTimeout
		default:
			err = cli.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, dp)
			if !apiserrors.IsNotFound(err) {
				time.Sleep(time.Second / 2)
				continue
			}
			pl, err := GetNodesPodList(ctx, cli, deploy.Spec.Template.Labels, []string{})
			if err != nil || len(pl.Items) != 0 {
				time.Sleep(time.Second / 2)
				continue
			}
			return nil
		}
	}
}
