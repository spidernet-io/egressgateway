// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	egressv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sync"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

var _ = Describe("EgressClusterInfo", func() {
	var patches []*gomonkey.Patches

	BeforeEach(func() {
		patches = make([]*gomonkey.Patches, 0)
		ctx, cancel = context.WithCancel(context.TODO())
		defer cancel()

		DeferCleanup(func() {
			for _, patch := range patches {
				if patch != nil {
					patch.Reset()
				}
			}
		})
	})

	Context("UT newEgressClusterInfoController", Label("newEgressClusterInfoController"), func() {

		It("nil log", func() {
			Expect(newEgressClusterInfoController(nil, nil, cfg)).To(HaveOccurred())
		})

		It("nil config", func() {
			Expect(newEgressClusterInfoController(nil, log, nil)).To(HaveOccurred())
		})

		It("failed to New egressClusterInfo controller", func() {
			patch := gomonkey.ApplyFuncReturn(controller.New, nil, ERR_FAILED_NEW_CONTROLLER)
			patches = append(patches, patch)
			Expect(newEgressClusterInfoController(mockManager, log, cfg)).To(MatchError(ERR_FAILED_NEW_CONTROLLER))
		})
	})

	Context("UT Reconcile", Label("eciReconciler"), func() {
		var apiServerPodLabel = map[string]string{"component": "kube-apiserver"}

		const (
			defaultEgressClusterInfoName = "default"
			calico                       = "calico"
			serviceClusterIpRange        = "service-cluster-ip-range"
			clusterIPRange               = "10.10.0.0/16,fddd:10::/16"
			nodeName                     = "test-node"
			podName                      = "test-pod"
			badNS                        = "badNS"
			nodeIPv4, nodeIPv6           = "10.10.0.2", "fddd:10::2"
		)

		var (
			r               *eciReconciler
			req             reconcile.Request
			namespace, name string
			eci             *egressv1beta1.EgressClusterInfo
			apiServerPod    *corev1.Pod
			node            *corev1.Node
		)

		BeforeEach(func() {
			eci = new(egressv1beta1.EgressClusterInfo)
			r = &eciReconciler{
				eci:               eci,
				client:            mockManager.GetClient(),
				log:               log,
				config:            cfg,
				doOnce:            sync.Once{},
				nodeIPv4Map:       make(map[string]string),
				nodeIPv6Map:       make(map[string]string),
				calicoV4IPPoolMap: make(map[string]string),
				calicoV6IPPoolMap: make(map[string]string),
			}

			req = reconcile.Request{
				NamespacedName: types.NamespacedName{},
			}

			// pod
			apiServerPod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   podName,
					Labels: apiServerPodLabel,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Command: []string{
								"--" + serviceClusterIpRange + "=" + clusterIPRange,
							},
						},
					},
				},
			}

			// node
			node = &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: nodeName,
				},
				Status: corev1.NodeStatus{
					Addresses: []corev1.NodeAddress{
						{
							Type:    corev1.NodeInternalIP,
							Address: nodeIPv4,
						},
						{
							Type:    corev1.NodeInternalIP,
							Address: nodeIPv6,
						},
					},
				},
			}

			err = r.client.Create(ctx, apiServerPod)
			Expect(err).NotTo(HaveOccurred())

			err = r.client.Delete(ctx, eci)
			Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			DeferCleanup(func() {
				// delete EgressClusterInfo
				err = r.client.Delete(ctx, eci)
				Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

				// delete api-server-pod
				err = r.client.Delete(ctx, apiServerPod)
				Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

				// delete node
				err = r.client.Delete(ctx, node)
				Expect(client.IgnoreNotFound(err)).NotTo(HaveOccurred())

			})
		})

		It("invalid request", func() {
			req.Namespace = badNS
			_, err = r.Reconcile(ctx, req)
			Expect(err).To(HaveOccurred())
		})

		It("reconcileNode delete node", func() {
			namespace, name = "Node/testNode", "testNode"
			req.Name, req.Namespace = name, namespace
			_, err = r.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
		})

		It("reconcileNode update node", func() {
			// create node
			err = r.client.Create(ctx, node)
			Expect(err).NotTo(HaveOccurred())

			namespace, name = "Node/", nodeName
			req.Name, req.Namespace = name, namespace
			_, err = r.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
