// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/e2eframework/framework"
)

func GenerateDeployYaml(deployName, nodeName string, replicas int32, label map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: NSDefault,
			Name:      deployName,
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
					NodeName: nodeName,
					Containers: []corev1.Container{
						{
							Name:            deployName,
							Image:           Env[IMAGE],
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c", "sleep infinity"},
						},
					},
				},
			},
		},
	}
}

func GenerateDSYaml(dsName string, label map[string]string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: NSDefault,
			Name:      dsName,
			Labels:    label,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: label,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: label,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            dsName,
							Image:           Env[IMAGE],
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         []string{"/bin/sh", "-c", "sleep infinity"},
						},
					},
				},
			},
		},
	}
}

func CreateDSUntilReady(f *framework.Framework, dsYaml *appsv1.DaemonSet, timeout time.Duration, opts ...client.CreateOption) (*corev1.PodList, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	return f.CreateDaemonsetUntilReady(ctx, dsYaml, opts...)
}

func DeleteDeployIfExists(f *framework.Framework, deployName, namespace string, duration time.Duration, opts ...client.DeleteOption) error {
	if len(deployName) == 0 || len(namespace) == 0 {
		return INVALID_INPUT
	}
	_, err := f.GetDeployment(deployName, namespace)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return f.DeleteDeploymentUntilFinish(deployName, namespace, duration, opts...)
}

func DeleteDSIfExists(f *framework.Framework, dsName, namespace string, timeout time.Duration, opts ...client.DeleteOption) error {
	if len(dsName) == 0 || len(namespace) == 0 {
		return INVALID_INPUT
	}
	ds, err := f.GetDaemonSet(dsName, namespace)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	err = f.DeleteDaemonSet(dsName, namespace, opts...)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	return f.WaitPodListDeleted(namespace, ds.Spec.Selector.MatchLabels, ctx)
}

func GetDSRunningPodList(f *framework.Framework, ds *appsv1.DaemonSet) (*corev1.PodList, error) {
	list, err := f.GetDaemonSetPodList(ds)
	if err != nil {
		return nil, err
	}
	runList := new(corev1.PodList)
	for _, item := range list.Items {
		if item.Status.Phase == corev1.PodRunning {
			runList.Items = append(runList.Items, item)
		}
	}
	return runList, nil
}

func GetPodListByNodeAndLabel(f *framework.Framework, nodeName string, label map[string]string) (*corev1.PodList, error) {
	podList := new(corev1.PodList)
	ops := []client.ListOption{
		client.MatchingFields(map[string]string{"spec.nodeName": nodeName}),
		client.MatchingLabels(label),
	}
	err := f.ListResource(podList, ops...)
	if err != nil {
		return nil, err
	}
	return podList, nil
}

func ListNodesPod(f *framework.Framework, label map[string]string, nodes []string) (*corev1.PodList, error) {
	podList := new(corev1.PodList)
	for _, node := range nodes {
		list, err := GetPodListByNodeAndLabel(f, node, label)
		if err != nil {
			return nil, err
		}
		podList.Items = append(podList.Items, list.Items...)
	}
	return podList, nil
}

func WaitAllPodsRunning(f *framework.Framework, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	return f.WaitAllPodUntilRunning(ctx)
}
