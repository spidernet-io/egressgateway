// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package constant

var WebhookMsgClusterDefaultGateway = "A cluster can only have one default gateway"

var (
	KubeControllerManagerLabel = map[string]string{"component": "kube-controller-manager"}
	KubeApiServerLabel         = map[string]string{"component": "kube-apiserver"}
	KubeEtcdLabel              = map[string]string{"component": "etcd"}
	KubeSchedulerLabel         = map[string]string{"component": "kube-scheduler"}
	KubeProxyLabel             = map[string]string{"k8s-app": "kube-proxy"}
	CalicoNodeLabel            = map[string]string{"k8s-app": "calico-node"}
	CalicoControllerLabel      = map[string]string{"k8s-app": "calico-kube-controllers"}
)
