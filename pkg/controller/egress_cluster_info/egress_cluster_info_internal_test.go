// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressclusterinfo

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"

	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"

	egressv1beta1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

var _ = Describe("EgressClusterInfo", Serial, Label("EgressClusterInfo UT"), func() {
	var ctx context.Context

	var (
		// err     error
		builder *fake.ClientBuilder
		r       *eciReconciler
		egci    *egressv1beta1.EgressClusterInfo
		objs    []client.Object
		// cli     client.WithWatch
		controllerManagerPod, controllerManagerPodV4, controllerManagerPodV6, controllerManagerPodNoCommand, controllerManagerPodNoContainer *corev1.Pod
		calicoIPPoolV4, calicoIPPoolV6                                                                                                       *calicov1.IPPool
		testNode                                                                                                                             *corev1.Node
	)

	BeforeEach(func() {
		ctx = context.TODO()

		// builder
		builder = fake.NewClientBuilder()
		builder.WithScheme(schema.GetScheme())

		// objs
		objs = []client.Object{}

		// eciReconciler
		r = &eciReconciler{
			// mgr:           mgr,
			eci:           new(egressv1beta1.EgressClusterInfo),
			log:           logr.Logger{},
			k8sPodCidr:    make(map[string]egressv1beta1.IPListPair),
			v4ClusterCidr: make([]string, 0),
			v6ClusterCidr: make([]string, 0),
		}

		// egci
		egci = &egressv1beta1.EgressClusterInfo{
			TypeMeta: metav1.TypeMeta{
				Kind:       "egressclusterinfos",
				APIVersion: "egressgateway.spidernet.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: egciName,
			},
			// Spec: egressv1beta1.EgressClusterInfoSpec{
			// 	ExtraCidr: []string{"10.10.0.0/16"},
			// },
		}

		// kube-controller-manager pod
		controllerManagerPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-controller-manager-test",
				Namespace: "kube-system",
				Labels:    kubeControllerManagerPodLabel,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Command: []string{
							"--cluster-cidr=172.40.0.0/16,fd40::/48",
							"--service-cluster-ip-range=172.41.0.0/16,fd41::/108",
						},
					},
				},
			},
		}

		controllerManagerPodV4 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-controller-manager-test",
				Namespace: "kube-system",
				Labels:    kubeControllerManagerPodLabel,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Command: []string{
							"--cluster-cidr=172.40.0.0/16",
							"--service-cluster-ip-range=172.41.0.0/16",
						},
					},
				},
			},
		}

		controllerManagerPodV6 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-controller-manager-test",
				Namespace: "kube-system",
				Labels:    kubeControllerManagerPodLabel,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Command: []string{
							"--cluster-cidr=fd40::/48",
							"--service-cluster-ip-range=fd41::/108",
						},
					},
				},
			},
		}

		controllerManagerPodNoCommand = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-controller-manager-test",
				Namespace: "kube-system",
				Labels:    kubeControllerManagerPodLabel,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Command: []string{},
					},
				},
			},
		}

		controllerManagerPodNoContainer = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-controller-manager-test",
				Namespace: "kube-system",
				Labels:    kubeControllerManagerPodLabel,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{},
			},
		}

		// test calico ippools
		calicoIPPoolV4 = &calicov1.IPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ippool-v4",
			},
			Spec: calicov1.IPPoolSpec{
				CIDR: "10.10.0.0/18",
			},
		}
		calicoIPPoolV6 = &calicov1.IPPool{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ippool-v6",
			},
			Spec: calicov1.IPPoolSpec{
				CIDR: "fdee:120::/120",
			},
		}

		// test node
		testNode = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node",
			},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{
						Type:    corev1.NodeInternalIP,
						Address: "10.10.10.1",
					},
					{
						Type:    corev1.NodeInternalIP,
						Address: "fddd:10::1",
					},
				},
			},
		}

		DeferCleanup(func() {
		})
	})

	// reconcileEgressClusterInfo
	Context("reconcileEgressClusterInfo", func() {
		It("when failed ParseKindWithReq", func() {
			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "badNamespace", Name: egciName}})
			Expect(err).To(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{}))
		})

		It("when failed getEgressClusterInfo", func() {
			// set client
			r.client = builder.Build()

			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindEGCI + "/", Name: egciName}})
			Expect(err).To(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{Requeue: true}))
		})

		It("when failed update egci", func() {
			objs = append(objs, egci)

			// set client without subresource
			builder.WithObjects(objs...)
			r.client = builder.Build()

			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindEGCI + "/", Name: egciName}})
			Expect(err).To(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{Requeue: true}))
		})

		It("will success when both egci.Spec.AutoDetect.NodeIP and isWatchingNode are true", func() {
			// when egci.Spec.AutoDetect.NodeIP is true
			egci.Spec.AutoDetect.NodeIP = true
			// when isWatchingNode is true
			r.isWatchingNode = true

			// set client
			objs = append(objs, egci)
			builder.WithObjects(objs...)
			builder.WithStatusSubresource(objs...)
			r.client = builder.Build()

			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindEGCI + "/", Name: egciName}})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{}))
		})

		DescribeTable("when egci.Spec.AutoDetec.PodCidrMode", func(isOK bool, prepare func(egci *egressv1beta1.EgressClusterInfo, r *eciReconciler)) {
			prepare(egci, r)
			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindEGCI + "/", Name: egciName}})
			if isOK {
				Expect(err).NotTo(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))
			} else {
				Expect(err).To(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true}))
			}
		},
			// when egci.Spec.AutoDetect.PodCidrMode is CniTypeCalico
			Entry("CniTypeCalico success", true, func(egci *egressv1beta1.EgressClusterInfo, r *eciReconciler) {
				// set egci
				egci.Spec.AutoDetect.PodCidrMode = egressv1beta1.CniTypeCalico

				// set eciReconciler
				r.isWatchingCalico = true
				objs = append(objs, egci)
				builder.WithObjects(objs...)
				builder.WithStatusSubresource(objs...)
				r.client = builder.Build()
			}),

			// when egci.Spec.AutoDetect.PodCidrMode is CniTypeK8s
			Entry("CniTypeK8s fail with no commands", false, func(egci *egressv1beta1.EgressClusterInfo, r *eciReconciler) {
				// set egci
				egci.Spec.AutoDetect.PodCidrMode = egressv1beta1.CniTypeK8s

				// set eciReconciler
				objs = []client.Object{controllerManagerPodNoCommand, egci}
				builder.WithObjects(objs...)
				builder.WithStatusSubresource(objs...)
				r.client = builder.Build()
			}),

			// when egci.Spec.AutoDetect.PodCidrMode is CniTypeK8s
			Entry("CniTypeK8s fail with no containers", false, func(egci *egressv1beta1.EgressClusterInfo, r *eciReconciler) {
				// set egci
				egci.Spec.AutoDetect.PodCidrMode = egressv1beta1.CniTypeK8s

				// set eciReconciler
				objs = []client.Object{controllerManagerPodNoContainer, egci}
				builder.WithObjects(objs...)
				builder.WithStatusSubresource(objs...)
				r.client = builder.Build()
			}),

			// when egci.Spec.AutoDetect.PodCidrMode is CniTypeK8s
			Entry("CniTypeK8s success dual", true, func(egci *egressv1beta1.EgressClusterInfo, r *eciReconciler) {
				// set egci
				egci.Spec.AutoDetect.PodCidrMode = egressv1beta1.CniTypeK8s

				// set eciReconciler
				objs = []client.Object{controllerManagerPod, egci}
				builder.WithObjects(objs...)
				builder.WithStatusSubresource(objs...)
				r.client = builder.Build()
			}),

			// when egci.Spec.AutoDetect.PodCidrMode is CniTypeK8s
			Entry("CniTypeK8s success v4", true, func(egci *egressv1beta1.EgressClusterInfo, r *eciReconciler) {
				// set egci
				egci.Spec.AutoDetect.PodCidrMode = egressv1beta1.CniTypeK8s

				// set eciReconciler
				objs = []client.Object{controllerManagerPodV4, egci}
				builder.WithObjects(objs...)
				builder.WithStatusSubresource(objs...)
				r.client = builder.Build()
			}),

			// when egci.Spec.AutoDetect.PodCidrMode is CniTypeK8s
			Entry("CniTypeK8s success v6", true, func(egci *egressv1beta1.EgressClusterInfo, r *eciReconciler) {
				// set egci
				egci.Spec.AutoDetect.PodCidrMode = egressv1beta1.CniTypeK8s

				// set eciReconciler
				objs = []client.Object{controllerManagerPodV6, egci}
				builder.WithObjects(objs...)
				builder.WithStatusSubresource(objs...)
				r.client = builder.Build()
			}),

			// when egci.Spec.AutoDetect.PodCidrMode is CniTypeEmpty
			Entry("CniTypeEmpty success", true, func(egci *egressv1beta1.EgressClusterInfo, r *eciReconciler) {
				// set egci
				egci.Spec.AutoDetect.PodCidrMode = egressv1beta1.CniTypeEmpty

				// set eciReconciler
				objs = append(objs, egci)
				builder.WithObjects(objs...)
				builder.WithStatusSubresource(objs...)
				r.client = builder.Build()
			}),

			// when egci.Spec.AutoDetect.PodCidrMode is unkownType
			Entry("unkownType fail", true, func(egci *egressv1beta1.EgressClusterInfo, r *eciReconciler) {
				// set egci
				egci.Spec.AutoDetect.PodCidrMode = "unkownType"

				// set eciReconciler
				objs = append(objs, egci)
				builder.WithObjects(objs...)
				builder.WithStatusSubresource(objs...)
				r.client = builder.Build()
			}),
		)

		It("will success when egci.Spec.AutoDetect.ClusterIP is true", func() {
			// when egci.Spec.AutoDetect.NodeIP is true
			egci.Spec.AutoDetect.ClusterIP = true

			// set client
			objs = []client.Object{controllerManagerPod, egci}
			builder.WithObjects(objs...)
			builder.WithStatusSubresource(objs...)
			r.client = builder.Build()

			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindEGCI + "/", Name: egciName}})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{}))
		})

		It("will success when egci.Spec.ExtraCidr is not nil", func() {
			// when egci.Spec.ExtraCidr is not nil
			egci.Spec.ExtraCidr = []string{"10.10.0.0/16"}

			// set client
			objs = append(objs, egci)
			builder.WithObjects(objs...)
			builder.WithStatusSubresource(objs...)
			r.client = builder.Build()

			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindEGCI + "/", Name: egciName}})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{}))
		})

	})

	// reconcileCalicoIPPool
	Context("reconcileCalicoIPPool", func() {
		It("will success when delete event", func() {
			// when egci.Spec.AutoDetect.PodCidrMode is CniTypeCalico
			egci.Spec.AutoDetect.PodCidrMode = egressv1beta1.CniTypeCalico

			// set client
			objs = append(objs, egci)
			builder.WithObjects(objs...)
			builder.WithStatusSubresource(objs...)
			r.client = builder.Build()

			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindCalicoIPPool + "/", Name: "xxx"}})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{}))
		})

		It("will success when update event", func() {
			egci.Spec.AutoDetect.PodCidrMode = egressv1beta1.CniTypeCalico

			// set client
			objs = []client.Object{egci, calicoIPPoolV4, calicoIPPoolV6}
			builder.WithObjects(objs...)
			builder.WithStatusSubresource(objs...)
			r.client = builder.Build()

			// reconcile calico ippools v4
			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindCalicoIPPool + "/", Name: calicoIPPoolV4.Name}})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{}))

			// reconcile calico ippools v6
			res, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindCalicoIPPool + "/", Name: calicoIPPoolV6.Name}})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{}))

			// test method listCalicoIPPools
			_, _ = r.listCalicoIPPools(ctx)

			// test method stopCheckCalico
			r.isCheckCalicoGoroutineRunning.Store(true)
			r.stopCheckChan = make(chan struct{})
			r.stopCheckCalico()

			// test method stopAllCheckGoroutine
			r.taskToken.Store(true)
			r.stopCheckChan = make(chan struct{})
			r.stopAllCheckGoroutine()

		})

	})

	// reconcileNode
	Context("reconcileNode", func() {
		It("will success when delete event", func() {
			// when egci.Spec.AutoDetect.NodeIP is true
			egci.Spec.AutoDetect.NodeIP = true

			// set client
			objs = append(objs, egci)
			builder.WithObjects(objs...)
			builder.WithStatusSubresource(objs...)
			r.client = builder.Build()

			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindNode + "/", Name: "xxx"}})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{}))
		})

		It("will success when update event", func() {
			// when egci.Spec.AutoDetect.NodeIP is true
			egci.Spec.AutoDetect.NodeIP = true

			// set client
			objs = []client.Object{egci, testNode}
			builder.WithObjects(objs...)
			builder.WithStatusSubresource(objs...)
			r.client = builder.Build()

			// reconcile node
			res, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: kindNode + "/", Name: testNode.Name}})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(reconcile.Result{}))

			// test method listNodeIPs
			_, _ = r.listNodeIPs(ctx)
		})
	})
})
