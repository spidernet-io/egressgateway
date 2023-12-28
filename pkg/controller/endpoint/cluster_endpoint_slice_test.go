// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package endpoint

import (
	"context"
	"github.com/spidernet-io/egressgateway/pkg/config"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"testing"

	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestNamespacePredicate(t *testing.T) {
	p := nsPredicate{}
	if !p.Create(event.CreateEvent{}) {
		t.Fatal("got false")
	}

	if !p.Delete(event.DeleteEvent{}) {
		t.Fatal("got false")
	}

	if !p.Update(event.UpdateEvent{
		ObjectOld: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
				Labels: map[string]string{
					"aa": "bb",
				},
			},
		},
		ObjectNew: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
				Labels: map[string]string{
					"aa": "cc",
				},
			},
		},
	}) {
		t.Fatal("got false")
	}
}

func TestReconcilerEgressClusterEndpointSlice(t *testing.T) {
	cases := map[string]TestCaseEPS{
		"caseAddEgressGatewayPolicy": caseAddClusterPolicy(),
		"caseClusterPolicyUpdatePod": caseClusterPolicyUpdatePod(),
		"caseClusterPolicyDeletePod": caseClusterPolicyDeletePod(),
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			builder.WithObjects(c.initialObjects...)
			cli := builder.Build()
			reconciler := endpointClusterReconciler{
				client: cli,
				log:    logger.NewLogger(c.config.EnvConfig.Logger),
				config: c.config,
			}

			for _, req := range c.reqs {
				res, err := reconciler.Reconcile(
					context.Background(),
					reconcile.Request{NamespacedName: req.nn},
				)
				if !req.expErr {
					assert.NoError(t, err)
				}
				assert.Equal(t, req.expRequeue, res.Requeue)

				ctx := context.Background()
				policy := new(v1beta1.EgressClusterPolicy)
				err = cli.Get(ctx, req.nn, policy)
				if err != nil {
					t.Fatal(err)
				}

				epList, err := listClusterEndpointSlices(ctx, cli, policy.Name)
				if err != nil {
					t.Fatal(err)
				}

				pods, err := listPodsByClusterPolicy(ctx, cli, policy)
				if err != nil {
					t.Fatal(err)
				}

				err = checkClusterPolicyIPsInEpSlice(pods, epList)
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func checkClusterPolicyIPsInEpSlice(pods []corev1.Pod, eps *v1beta1.EgressClusterEndpointSliceList) error {
	if pods == nil || eps == nil {
		return nil
	}
	podMap := make(map[string]struct{})
	epMap := make(map[string]struct{})
	for _, pod := range pods {
		for _, ip := range pod.Status.PodIPs {
			podMap[ip.IP] = struct{}{}
		}
	}

	for _, item := range eps.Items {
		for _, endpoint := range item.Endpoints {
			for _, ip := range endpoint.IPv4 {
				epMap[ip] = struct{}{}
			}
			for _, ip := range endpoint.IPv6 {
				epMap[ip] = struct{}{}
			}
		}
	}

	return compareMaps(podMap, epMap)
}

func caseAddClusterPolicy() TestCaseEPS {
	labels := map[string]string{"app": "nginx1"}
	initialObjects := []client.Object{
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "policy1",
			},
			Spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
				DestSubnet: nil,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
				Labels:    labels,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.1"},
					{IP: "10.6.0.2"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "default",
				Labels:    labels,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.3"},
					{IP: "10.6.0.4"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod3",
				Namespace: "default",
				Labels:    labels,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.5"},
					{IP: "10.6.0.6"},
				},
			},
		},
	}
	reqs := []TestEgressGatewayPolicyReq{
		{
			nn:         types.NamespacedName{Namespace: "", Name: "policy1"},
			expErr:     false,
			expRequeue: false,
		},
	}

	conf := &config.Config{
		FileConfig: config.FileConfig{
			MaxNumberEndpointPerSlice: 2,
		},
	}

	return TestCaseEPS{
		initialObjects: initialObjects,
		reqs:           reqs,
		config:         conf,
	}
}

func caseClusterPolicyUpdatePod() TestCaseEPS {
	labels := map[string]string{"app": "nginx1"}
	initialObjects := []client.Object{
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "policy1",
			},
			Spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
				DestSubnet: nil,
			},
		},
		&v1beta1.EgressClusterEndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name: "s1",
				Labels: map[string]string{
					v1beta1.LabelPolicyName: "policy1",
				},
			},
			Endpoints: []v1beta1.EgressEndpoint{
				{
					Namespace: "default",
					Pod:       "pod1",
					IPv4: []string{
						"10.6.0.1", "10.6.0.2",
					},
					IPv6: []string{},
					Node: "",
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
				Labels:    labels,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.1"},
					{IP: "10.6.0.2"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "default",
				Labels:    labels,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.3"},
					{IP: "10.6.0.4"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod3",
				Namespace: "default",
				Labels:    labels,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.5"},
					{IP: "10.6.0.6"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod4",
				Namespace: "default",
				Labels:    labels,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.7"},
					{IP: "10.6.0.8"},
				},
			},
		},
	}
	reqs := []TestEgressGatewayPolicyReq{
		{
			nn:         types.NamespacedName{Namespace: "", Name: "policy1"},
			expErr:     false,
			expRequeue: false,
		},
	}

	conf := &config.Config{
		FileConfig: config.FileConfig{
			MaxNumberEndpointPerSlice: 2,
		},
	}

	return TestCaseEPS{
		initialObjects: initialObjects,
		reqs:           reqs,
		config:         conf,
	}
}

func caseClusterPolicyDeletePod() TestCaseEPS {
	labels := map[string]string{"app": "nginx1"}
	initialObjects := []client.Object{
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "policy1",
			},
			Spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
				DestSubnet: nil,
			},
		},
		&v1beta1.EgressClusterEndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name: "s1",
				Labels: map[string]string{
					v1beta1.LabelPolicyName: "policy1",
				},
			},
			Endpoints: []v1beta1.EgressEndpoint{
				{
					Namespace: "default",
					Pod:       "pod1",
					IPv4: []string{
						"10.6.0.1",
						"10.6.0.2",
					},
					IPv6: []string{},
					Node: "",
				},
				{
					Namespace: "default",
					Pod:       "pod2",
					IPv4: []string{
						"10.6.0.3",
						"10.6.0.4",
					},
					IPv6: []string{},
					Node: "",
				},
			},
		},
		&v1beta1.EgressEndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name: "s2",
				Labels: map[string]string{
					v1beta1.LabelPolicyName: "policy1",
				},
			},
			Endpoints: []v1beta1.EgressEndpoint{
				{
					Namespace: "default",
					Pod:       "pod3",
					IPv4: []string{
						"10.6.0.5",
						"10.6.0.6",
					},
					IPv6: []string{},
					Node: "",
				},
				{
					Namespace: "default",
					Pod:       "pod4",
					IPv4: []string{
						"10.6.0.7",
						"10.6.0.8",
					},
					IPv6: []string{},
					Node: "",
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
				Labels:    labels,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.1"},
					{IP: "10.6.0.2"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod2",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.3"},
					{IP: "10.6.0.4"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod3",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.5"},
					{IP: "10.6.0.6"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod4",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.7"},
					{IP: "10.6.0.8"},
				},
			},
		},
	}
	reqs := []TestEgressGatewayPolicyReq{
		{
			nn:         types.NamespacedName{Name: "policy1"},
			expErr:     false,
			expRequeue: false,
		},
	}

	conf := &config.Config{
		FileConfig: config.FileConfig{
			MaxNumberEndpointPerSlice: 2,
		},
	}

	return TestCaseEPS{
		initialObjects: initialObjects,
		reqs:           reqs,
		config:         conf,
	}
}

func TestEnqueueNS(t *testing.T) {
	initialObjects := []client.Object{
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "p1",
			},
			Spec: v1beta1.EgressClusterPolicySpec{},
		},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	builder.WithObjects(initialObjects...)
	cli := builder.Build()

	f := enqueueNS(cli)
	ctx := context.Background()

	f(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{},
	})
}

func TestEnqueueEGCP(t *testing.T) {
	labels := map[string]string{"app": "nginx1"}
	initialObjects := []client.Object{
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "p1",
			},
			Spec: v1beta1.EgressClusterPolicySpec{
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
			},
		},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	builder.WithObjects(initialObjects...)
	cli := builder.Build()

	f := enqueueEGCP(cli)
	ctx := context.Background()

	f(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
			Labels:    labels,
		},
	})
}

func TestNewEgressClusterEpSliceController(t *testing.T) {
	var initialObjects []client.Object

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	builder.WithObjects(initialObjects...)
	cli := builder.Build()

	mgrOpts := manager.Options{
		Scheme: schema.GetScheme(),
		NewClient: func(config *rest.Config, options client.Options) (client.Client, error) {
			return cli, nil
		},
	}

	cfg := &config.Config{
		KubeConfig: &rest.Config{},
		FileConfig: config.FileConfig{
			MaxNumberEndpointPerSlice: 100,
			IPTables: config.IPTables{
				RefreshIntervalSecond:   90,
				PostWriteIntervalSecond: 1,
				LockTimeoutSecond:       0,
				LockProbeIntervalMillis: 50,
				LockFilePath:            "/run/xtables.lock",
				RestoreSupportsLock:     true,
			},
			Mark: "0x26000000",
			GatewayFailover: config.GatewayFailover{
				Enable:              true,
				TunnelMonitorPeriod: 5,
				TunnelUpdatePeriod:  5,
				EipEvictionTimeout:  15,
			},
		},
	}
	log := logger.NewLogger(cfg.EnvConfig.Logger)
	mgr, err := ctrl.NewManager(cfg.KubeConfig, mgrOpts)
	if err != nil {
		t.Fatal(err)
	}
	err = NewEgressClusterEpSliceController(mgr, log, cfg)
	if err != nil {
		t.Fatal(err)
	}
}
