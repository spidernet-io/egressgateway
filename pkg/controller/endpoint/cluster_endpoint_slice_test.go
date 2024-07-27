// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package endpoint

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-logr/logr"
	"github.com/spidernet-io/egressgateway/pkg/coalescing"
	"github.com/spidernet-io/egressgateway/pkg/config"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

var errForMock = errors.New("mock err")

func Test_NewEgressClusterEpSliceController(t *testing.T) {
	cases := map[string]struct {
		patchFun func(*testing.T, reconcile.Reconciler, manager.Manager, logr.Logger) []gomonkey.Patches
		expErr   bool
	}{
		"failed NewRequestCache": {
			patchFun: mock_NewEgressClusterEpSliceController_NewRequestCache_err,
			expErr:   true,
		},
		"failed New controller": {
			patchFun: mock_NewEgressClusterEpSliceController_New_err,
			expErr:   true,
		},

		"failed controller watch pod": {
			patchFun: mock_NewEgressClusterEpSliceController_Watch_pod_err,
			expErr:   true,
		},

		"failed controller watch namespace": {
			patchFun: mock_NewEgressClusterEpSliceController_Watch_namespace_err,
			expErr:   true,
		},
		"failed controller watch egressClusterPolicy": {
			patchFun: mock_NewEgressClusterEpSliceController_Watch_clusterpolicy_err,
			expErr:   true,
		},
		"failed controller watch egressClusterEndpointSlice": {
			patchFun: mock_NewEgressClusterEpSliceController_Watch_clusterendpointslice_err,
			expErr:   true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
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
			r := &endpointClusterReconciler{
				client: mgr.GetClient(),
				log:    log,
				config: cfg,
			}

			// patch
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFun != nil {
				patches = tc.patchFun(t, r, mgr, log)
			}

			err = NewEgressClusterEpSliceController(mgr, log, cfg)
			if tc.expErr {
				assert.Error(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_Reconcile(t *testing.T) {
	cases := map[string]struct {
		setMaxNumberEndpointPerSlice func(*endpointClusterReconciler)
		setObjs                      func() ([]client.Object, reconcile.Request)
		patchFun                     func(*testing.T, *endpointClusterReconciler, manager.Manager, logr.Logger) []gomonkey.Patches
		expErr                       bool
	}{
		"failed Get clusterPolicy": {
			patchFun: mock_endpointClusterReconciler_Reconcile_Get_err,
			expErr:   true,
		},
		"failed Get clusterPolicy notFound": {
			patchFun: mock_endpointClusterReconciler_Reconcile_Get_err_notFound,
		},

		"failed listPodsByClusterPolicy": {
			patchFun: mock_listPodsByClusterPolicy_err,
			expErr:   true,
		},

		"failed listClusterEndpointSlices": {
			patchFun: mock_listClusterEndpointSlices_err,
			expErr:   true,
		},

		" needUpdateEndpoint true": {
			setObjs:  mock_ClusterPolicyObjs,
			patchFun: mock_needUpdateEndpoint_true,
		},

		"need to CreateEgressEndpointSlice": {
			setObjs: mock_ClusterPolicyObjs_need_create_endpoint,
		},
		"need to CreateEgressEndpointSlice less count": {
			setMaxNumberEndpointPerSlice: mock_MaxNumberEndpointPerSlice_less_count,
			setObjs:                      mock_ClusterPolicyObjs_need_create_endpoint,
		},

		"need to CreateEgressEndpointSlice but sliceNotChange": {
			setObjs: mock_ClusterPolicyObjs_not_change,
		},

		"need to delete endpoint": {
			setObjs: mock_ClusterPolicyObjs_need_delete_endpoint,
		},

		"failed to update endpoint": {
			setObjs:  mock_ClusterPolicyObjs,
			patchFun: mock_endpointClusterReconciler_Reconcile_Update_err,
			expErr:   true,
		},

		"failed to create endpoint": {
			setObjs:  mock_ClusterPolicyObjs_no_endpoint,
			patchFun: mock_endpointClusterReconciler_Reconcile_Create_err,
			expErr:   true,
		},

		"failed to delete endpoint": {
			setObjs:  mock_ClusterPolicyObjs_need_delete_endpoint,
			patchFun: mock_endpointClusterReconciler_Reconcile_Delete_err,
			expErr:   true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var initialObjects []client.Object
			var req reconcile.Request
			if tc.setObjs != nil {
				initialObjects, req = tc.setObjs()
			}

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
			r := &endpointClusterReconciler{
				client: mgr.GetClient(),
				log:    log,
				config: cfg,
			}

			if tc.setMaxNumberEndpointPerSlice != nil {
				tc.setMaxNumberEndpointPerSlice(r)
			}

			// patch
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFun != nil {
				patches = tc.patchFun(t, r, mgr, log)
			}

			_, err = r.Reconcile(context.TODO(), req)

			if tc.expErr {
				assert.Error(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_listPodsByClusterPolicy(t *testing.T) {
	cases := map[string]struct {
		setObjs   func() []client.Object
		setParams func() *v1beta1.EgressClusterPolicy
		patchFun  func(c client.Client) []gomonkey.Patches
		expErr    bool
	}{
		"failed LabelSelectorAsSelector when nil namespaceSelector": {
			setParams: mock_listPodsByClusterPolicy_nil_NamespaceSelector,
			patchFun:  mock_listPodsByClusterPolicy_LabelSelectorAsSelector_err,
			expErr:    true,
		},

		"failed List when nil namespaceSelector": {
			setParams: mock_listPodsByClusterPolicy_nil_NamespaceSelector,
			patchFun:  mock_listPodsByClusterPolicy_List_err,
			expErr:    true,
		},

		"failed LabelSelectorAsSelector when not nil namespaceSelector": {
			setParams: mock_listPodsByClusterPolicy_not_nil_NamespaceSelector,
			patchFun:  mock_listPodsByClusterPolicy_LabelSelectorAsSelector_err,
			expErr:    true,
		},

		"failed List when not nil namespaceSelector": {
			setParams: mock_listPodsByClusterPolicy_not_nil_NamespaceSelector,
			patchFun:  mock_listPodsByClusterPolicy_List_err,
			expErr:    true,
		},

		"failed LabelSelectorAsSelector when not nil namespaceSelector second": {
			setObjs:   mock_listPodsByClusterPolicy_Objs,
			setParams: mock_listPodsByClusterPolicy_not_nil_NamespaceSelector,
			patchFun:  mock_listPodsByClusterPolicy_LabelSelectorAsSelector_err_second,
			expErr:    true,
		},

		"succeeded listPodsByClusterPolicy": {
			setObjs:   mock_listPodsByClusterPolicy_Objs,
			setParams: mock_listPodsByClusterPolicy_not_nil_NamespaceSelector,
			// patchFun:  mock_listPodsByClusterPolicy_List_err_second,
			// expErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			if tc.setObjs != nil {
				builder.WithObjects(tc.setObjs()...)
				builder.WithStatusSubresource(tc.setObjs()...)
			}
			cli := builder.Build()

			// patch
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFun != nil {
				patches = tc.patchFun(cli)
			}

			policy := tc.setParams()
			_, err := listPodsByClusterPolicy(context.TODO(), cli, policy)

			if tc.expErr {
				assert.Error(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_listClusterEndpointSlices(t *testing.T) {
	t.Run("failed to LabelSelectorAsSelector", func(t *testing.T) {
		p := gomonkey.ApplyFuncReturn(metav1.LabelSelectorAsSelector, nil, errForMock)
		defer p.Reset()

		builder := fake.NewClientBuilder()
		builder.WithScheme(schema.GetScheme())
		cli := builder.Build()
		_, err := listClusterEndpointSlices(context.TODO(), cli, "testPolicy")
		assert.Error(t, err)
	})
}

func TestUpdateNamespace(t *testing.T) {
	cases := map[string]struct {
		in  event.UpdateEvent
		res bool
	}{
		"ObjectOld all nil": {
			in:  event.UpdateEvent{},
			res: false,
		},
		"ObjectNew nil": {
			in: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{},
			},
			res: false,
		},
		"ObjectNew Namespace label equal": {
			in: event.UpdateEvent{
				ObjectOld: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"foo": "bar"},
					},
				},
				ObjectNew: &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"foo": "bar"},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := nsPredicate{}
			got := p.Update(tc.in)
			if got != tc.res {
				t.Fatalf("got %v", got)
			}
		})
	}
}

func Test_Generic(t *testing.T) {
	t.Run("test Generic", func(t *testing.T) {
		p := nsPredicate{}
		e := event.GenericEvent{}
		res := p.Generic(e)
		assert.True(t, res)
	})
}

func Test_enqueueNS(t *testing.T) {
	cases := map[string]struct {
		in       *corev1.Namespace
		objs     []client.Object
		patchFun func(c client.Client) []gomonkey.Patches
		expErr   bool
	}{
		"failed List": {
			in:       &corev1.Namespace{},
			patchFun: mock_enqueueNS_List_err,
			expErr:   true,
		},
		"failed LabelSelectorAsSelector": {
			in: &corev1.Namespace{},
			objs: []client.Object{
				&v1beta1.EgressClusterPolicy{},
			},
			patchFun: mock_enqueueNS_LabelSelectorAsSelector_err,
			expErr:   true,
		},
		"succeeded enqueueNS": {
			in: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"foo": "bar"},
				},
			},
			objs: []client.Object{
				&v1beta1.EgressClusterPolicy{
					Spec: v1beta1.EgressClusterPolicySpec{
						AppliedTo: v1beta1.ClusterAppliedTo{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"foo": "bar"},
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			if tc.objs != nil {
				builder.WithObjects(tc.objs...)
				builder.WithStatusSubresource(tc.objs...)
			}
			cli := builder.Build()

			// patch
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFun != nil {
				patches = tc.patchFun(cli)
			}

			resF := enqueueNS(cli)
			res := resF(context.TODO(), tc.in)

			if tc.expErr {
				assert.Nil(t, res)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_enqueueEGCP(t *testing.T) {
	cases := map[string]struct {
		in       client.Object
		objs     []client.Object
		patchFun func(c client.Client) []gomonkey.Patches
		expErr   bool
	}{
		"failed not pod obj": {
			in: &corev1.Node{},
			// patchFun: mock_enqueueNS_List_err,
			expErr: true,
		},

		"failed to list policy": {
			in:       &corev1.Pod{},
			patchFun: mock_enqueueEGCP_List_err_one,
			expErr:   true,
		},
		"failed to list namespace": {
			in:       &corev1.Pod{},
			patchFun: mock_enqueueEGCP_List_err_two,
			expErr:   true,
		},
		"failed to Get namespace": {
			in: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testpod",
					Namespace: "testns",
					Labels:    map[string]string{"app": "testpod"},
				},
			},
			objs: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "testns",
						Labels: map[string]string{"name": "testns"},
					},
				},
				&v1beta1.EgressClusterPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testpolicy",
						Namespace: "testns",
					},
					Spec: v1beta1.EgressClusterPolicySpec{
						AppliedTo: v1beta1.ClusterAppliedTo{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "testpod"},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"name": "testns"},
							},
						},
					},
				},
			},
			patchFun: mock_enqueueEGCP_Get_err,
			expErr:   true,
		},

		"failed to LabelSelectorAsSelector first": {
			in: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testpod",
					Namespace: "testns",
					Labels:    map[string]string{"app": "testpod"},
				},
			},
			objs: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "testns",
						Labels: map[string]string{"name": "testns"},
					},
				},
				&v1beta1.EgressClusterPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testpolicy",
						Namespace: "testns",
					},
					Spec: v1beta1.EgressClusterPolicySpec{
						AppliedTo: v1beta1.ClusterAppliedTo{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "testpod"},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"name": "testns"},
							},
						},
					},
				},
			},
			patchFun: mock_enqueueEGCP_LabelSelectorAsSelector_err_first,
			expErr:   true,
		},

		"failed to LabelSelectorAsSelector second": {
			in: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testpod",
					Namespace: "testns",
					Labels:    map[string]string{"app": "testpod"},
				},
			},
			objs: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "testns",
						Labels: map[string]string{"name": "testns"},
					},
				},
				&v1beta1.EgressClusterPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testpolicy",
						Namespace: "testns",
					},
					Spec: v1beta1.EgressClusterPolicySpec{
						AppliedTo: v1beta1.ClusterAppliedTo{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "testpod"},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"name": "testns"},
							},
						},
					},
				},
			},
			patchFun: mock_enqueueEGCP_LabelSelectorAsSelector_err_second,
			expErr:   true,
		},

		"ns not match enqueueEGCP": {
			in: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testpod",
					Namespace: "testns1",
					Labels:    map[string]string{"app": "testpod"},
				},
			},
			objs: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "testns1",
						Labels: map[string]string{"name": "testns1"},
					},
				},
				&v1beta1.EgressClusterPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testpolicy",
					},
					Spec: v1beta1.EgressClusterPolicySpec{
						AppliedTo: v1beta1.ClusterAppliedTo{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "testpod"},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"name": "testns3"},
							},
						},
					},
				},
			},
		},

		"succeeded enqueueEGCP": {
			in: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testpod",
					Namespace: "testns",
					Labels:    map[string]string{"app": "testpod"},
				},
			},
			objs: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "testns1",
						Labels: map[string]string{"name": "testns1"},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "testns",
						Labels: map[string]string{"name": "testns"},
					},
				},
				&v1beta1.EgressClusterPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testpolicy",
						Namespace: "testns",
					},
					Spec: v1beta1.EgressClusterPolicySpec{
						AppliedTo: v1beta1.ClusterAppliedTo{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"app": "testpod"},
							},
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{"name": "testns"},
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			if tc.objs != nil {
				builder.WithObjects(tc.objs...)
				builder.WithStatusSubresource(tc.objs...)
			}
			cli := builder.Build()

			// patch
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFun != nil {
				patches = tc.patchFun(cli)
			}

			resF := enqueueEGCP(cli)
			res := resF(context.TODO(), tc.in)

			if tc.expErr {
				assert.Nil(t, res)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func mock_NewEgressClusterEpSliceController_NewRequestCache_err(t *testing.T, r reconcile.Reconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(coalescing.NewRequestCache, nil, errForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_NewEgressClusterEpSliceController_New_err(t *testing.T, r reconcile.Reconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(controller.New, nil, errForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_NewEgressClusterEpSliceController_Watch_pod_err(t *testing.T, r reconcile.Reconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	name := "test-controller"
	cache, err := coalescing.NewRequestCache(time.Second)
	assert.NoError(t, err)
	reduce := coalescing.NewReconciler(r, cache, log)

	c, err := controller.New(name, mgr, controller.Options{Reconciler: reduce})
	assert.NoError(t, err)
	patch1 := gomonkey.ApplyFuncReturn(controller.New, c, nil)
	patch2 := gomonkey.ApplyMethodReturn(c, "Watch", errForMock)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_NewEgressClusterEpSliceController_Watch_namespace_err(t *testing.T, r reconcile.Reconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	name := "test-controller"
	cache, err := coalescing.NewRequestCache(time.Second)
	assert.NoError(t, err)
	reduce := coalescing.NewReconciler(r, cache, log)

	c, err := controller.New(name, mgr, controller.Options{Reconciler: reduce})
	assert.NoError(t, err)
	patch1 := gomonkey.ApplyFuncReturn(controller.New, c, nil)
	patch2 := gomonkey.ApplyMethodSeq(c, "Watch", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{errForMock}, Times: 1},
	})
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_NewEgressClusterEpSliceController_Watch_clusterpolicy_err(t *testing.T, r reconcile.Reconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	name := "test-controller"
	cache, err := coalescing.NewRequestCache(time.Second)
	assert.NoError(t, err)
	reduce := coalescing.NewReconciler(r, cache, log)

	c, err := controller.New(name, mgr, controller.Options{Reconciler: reduce})
	assert.NoError(t, err)
	patch1 := gomonkey.ApplyFuncReturn(controller.New, c, nil)
	patch2 := gomonkey.ApplyMethodSeq(c, "Watch", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{errForMock}, Times: 1},
	})
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_NewEgressClusterEpSliceController_Watch_clusterendpointslice_err(t *testing.T, r reconcile.Reconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	name := "test-controller"
	cache, err := coalescing.NewRequestCache(time.Second)
	assert.NoError(t, err)
	reduce := coalescing.NewReconciler(r, cache, log)

	c, err := controller.New(name, mgr, controller.Options{Reconciler: reduce})
	assert.NoError(t, err)
	patch1 := gomonkey.ApplyFuncReturn(controller.New, c, nil)
	patch2 := gomonkey.ApplyMethodSeq(c, "Watch", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{errForMock}, Times: 1},
	})
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_endpointClusterReconciler_Reconcile_Get_err(t *testing.T, r *endpointClusterReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", errForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_endpointClusterReconciler_Reconcile_Get_err_notFound(t *testing.T, r *endpointClusterReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", errForMock)
	patch2 := gomonkey.ApplyFuncReturn(apierrors.IsNotFound, true)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_listPodsByClusterPolicy_err(t *testing.T, r *endpointClusterReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", nil)
	patch2 := gomonkey.ApplyFuncReturn(listPodsByClusterPolicy, nil, errForMock)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_listClusterEndpointSlices_err(t *testing.T, r *endpointClusterReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", nil)
	patch2 := gomonkey.ApplyFuncReturn(listPodsByClusterPolicy, nil, nil)
	patch3 := gomonkey.ApplyFuncReturn(listClusterEndpointSlices, nil, errForMock)

	return []gomonkey.Patches{*patch1, *patch2, *patch3}
}

func mock_needUpdateEndpoint_true(t *testing.T, r *endpointClusterReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(needUpdateEndpoint, true)

	return []gomonkey.Patches{*patch1}
}

func mock_ClusterPolicyObjs() ([]client.Object, reconcile.Request) {
	labels := map[string]string{"app": "nginx1"}
	labels2 := map[string]string{"app": "nginx2"}
	policyName := "policy1"
	policyName2 := "policy2"

	initialObjects := []client.Object{
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName,
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
					v1beta1.LabelPolicyName: policyName,
				},
			},
			Endpoints: []v1beta1.EgressEndpoint{
				{
					Namespace: "default",
					Pod:       "pod1",
					IPv4: []string{
						"10.6.0.1",
					},
					IPv6: []string{},
					Node: "",
				},
				{
					Namespace: "default",
					Pod:       "pod2",
					IPv4: []string{
						"10.6.0.2",
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
					{IP: "10.6.0.2"},
				},
			},
		},
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName2,
			},
			Spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels2},
				},
				DestSubnet: nil,
			},
		},
		&v1beta1.EgressClusterEndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name: "s2",
				Labels: map[string]string{
					v1beta1.LabelPolicyName: policyName2,
				},
			},
			Endpoints: []v1beta1.EgressEndpoint{
				{
					Namespace: "default",
					Pod:       "pod3",
					IPv4: []string{
						"10.6.0.3",
					},
					IPv6: []string{},
					Node: "",
				},
				{
					Namespace: "default",
					Pod:       "pod4",
					IPv4: []string{
						"10.6.0.4",
					},
					IPv6: []string{},
					Node: "",
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod3",
				Namespace: "default",
				Labels:    labels2,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.3"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod4",
				Namespace: "default",
				Labels:    labels2,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.14"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod5",
				Namespace: "default",
				Labels:    labels2,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.5"},
				},
			},
		},
		&v1beta1.EgressClusterEndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name: "s3",
				Labels: map[string]string{
					v1beta1.LabelPolicyName: "nopolicy",
				},
			},
			Endpoints: []v1beta1.EgressEndpoint{
				{
					Namespace: "default",
					Pod:       "nopod1",
					IPv4: []string{
						"10.7.0.1",
					},
					IPv6: []string{},
					Node: "",
				},
				{
					Namespace: "default",
					Pod:       "nopod2",
					IPv4: []string{
						"10.7.0.2",
					},
					IPv6: []string{},
					Node: "",
				},
			},
		},
	}
	return initialObjects, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "", Name: policyName}}
}

func mock_ClusterPolicyObjs_no_endpoint() ([]client.Object, reconcile.Request) {
	labels1 := map[string]string{"app": "nginx1"}
	policyName := "policy1"
	initialObjects := []client.Object{
		// &v1beta1.EgressClusterEndpointSlice{
		// 	ObjectMeta: metav1.ObjectMeta{
		// 		Name: "s1",
		// 		Labels: map[string]string{
		// 			v1beta1.LabelPolicyName: policyName,
		// 		},
		// 	},
		// 	Endpoints: []v1beta1.EgressEndpoint{
		// 		{
		// 			Namespace: "defaultxx",
		// 			Pod:       "pod1",
		// 			IPv4: []string{
		// 				"10.6.0.1", "10.6.0.2",
		// 			},
		// 			IPv6: []string{},
		// 			Node: "",
		// 		},
		// 	},
		// },
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName,
			},
			Spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels1},
				},
				DestSubnet: nil,
			},
		},

		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
				Labels:    labels1,
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
				Labels:    labels1,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.3"},
					{IP: "10.6.0.4"},
				},
			},
		},
	}
	return initialObjects, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "", Name: policyName}}
}

func mock_ClusterPolicyObjs_need_create_endpoint() ([]client.Object, reconcile.Request) {
	labels1 := map[string]string{"app": "nginx1"}
	labels2 := map[string]string{"app": "nginx2"}
	policyName := "policy1"
	policyName2 := "policy2"
	policyName3 := "policy3"
	initialObjects := []client.Object{
		&v1beta1.EgressClusterEndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name: "s1",
				Labels: map[string]string{
					v1beta1.LabelPolicyName: policyName,
				},
			},
			Endpoints: []v1beta1.EgressEndpoint{
				{
					Namespace: "defaultxx",
					Pod:       "pod1",
					IPv4: []string{
						"10.6.0.1", "10.6.0.2",
					},
					IPv6: []string{},
					Node: "",
				},
			},
		},
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName,
			},
			Spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels2},
				},
				DestSubnet: nil,
			},
		},
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName2,
			},
			Spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels1},
				},
				DestSubnet: nil,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod1",
				Namespace: "default",
				Labels:    labels1,
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
				Labels:    labels1,
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
				Labels:    labels1,
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
				Labels:    labels2,
			},
			Spec: corev1.PodSpec{},
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.6.0.5"},
					{IP: "10.6.0.6"},
				},
			},
		},
		&v1beta1.EgressClusterEndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name: "s3",
				Labels: map[string]string{
					v1beta1.LabelPolicyName: policyName3,
				},
			},
			Endpoints: []v1beta1.EgressEndpoint{
				{
					Namespace: "default",
					Pod:       "podxxx",
					IPv4: []string{
						"10.6.0.1", "10.6.0.2",
					},
					IPv6: []string{},
					Node: "",
				},
			},
		},
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName3,
			},
			Spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels2},
				},
				DestSubnet: nil,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "podxxx",
				Namespace: "default",
				Labels:    labels2,
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
	return initialObjects, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "", Name: policyName}}
}

func mock_MaxNumberEndpointPerSlice_less_count(r *endpointClusterReconciler) {
	r.config.FileConfig.MaxNumberEndpointPerSlice = 1
}

func mock_ClusterPolicyObjs_not_change() ([]client.Object, reconcile.Request) {
	labels := map[string]string{"app": "nginx1"}
	policyName := "policy1"

	initialObjects := []client.Object{
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName,
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
					v1beta1.LabelPolicyName: policyName,
				},
			},
			Endpoints: []v1beta1.EgressEndpoint{
				{
					Namespace: "default",
					Pod:       "pod1",
					IPv4: []string{
						"10.6.0.1",
					},
					IPv6: []string{},
					Node: "",
				},
				{
					Namespace: "default",
					Pod:       "pod2",
					IPv4: []string{
						"10.6.0.2",
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
					{IP: "10.6.0.2"},
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
					{IP: "10.6.0.3"},
				},
			},
		},
	}
	return initialObjects, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "", Name: policyName}}
}

func mock_ClusterPolicyObjs_need_delete_endpoint() ([]client.Object, reconcile.Request) {
	labels := map[string]string{"app": "nginx1"}
	policyName := "policy1"

	initialObjects := []client.Object{
		&v1beta1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName,
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
					v1beta1.LabelPolicyName: policyName,
				},
			},
			Endpoints: []v1beta1.EgressEndpoint{
				{
					Namespace: "default",
					Pod:       "pod1",
					IPv4: []string{
						"10.6.0.1",
					},
					IPv6: []string{},
					Node: "",
				},
				{
					Namespace: "default",
					Pod:       "pod2",
					IPv4: []string{
						"10.6.0.2",
					},
					IPv6: []string{},
					Node: "",
				},
			},
		},
	}
	return initialObjects, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "", Name: policyName}}
}

func mock_endpointClusterReconciler_Reconcile_Update_err(t *testing.T, r *endpointClusterReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Update", errForMock)
	patch2 := gomonkey.ApplyFuncReturn(needUpdateEndpoint, true)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_endpointClusterReconciler_Reconcile_Create_err(t *testing.T, r *endpointClusterReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Create", errForMock)
	patch2 := gomonkey.ApplyFuncReturn(needUpdateEndpoint, false)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_endpointClusterReconciler_Reconcile_Delete_err(t *testing.T, r *endpointClusterReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Delete", errForMock)
	patch2 := gomonkey.ApplyFuncReturn(needUpdateEndpoint, false)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_listPodsByClusterPolicy_nil_NamespaceSelector() *v1beta1.EgressClusterPolicy {
	return &v1beta1.EgressClusterPolicy{
		Spec: v1beta1.EgressClusterPolicySpec{
			AppliedTo: v1beta1.ClusterAppliedTo{
				PodSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"foo": "bar"},
				},
			},
		},
	}
}

func mock_listPodsByClusterPolicy_LabelSelectorAsSelector_err(c client.Client) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(metav1.LabelSelectorAsSelector, nil, errForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_listPodsByClusterPolicy_List_err(c client.Client) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(c, "List", errForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_listPodsByClusterPolicy_not_nil_NamespaceSelector() *v1beta1.EgressClusterPolicy {
	return &v1beta1.EgressClusterPolicy{
		Spec: v1beta1.EgressClusterPolicySpec{
			AppliedTo: v1beta1.ClusterAppliedTo{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"name": "ns1"},
				},
				PodSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"foo": "bar"},
				},
			},
		},
	}
}

func mock_listPodsByClusterPolicy_Objs() []client.Object {

	initialObjects := []client.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "ns1",
				Labels: map[string]string{"name": "ns1"},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "ns2",
				Labels: map[string]string{"name": "ns2"},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "ns3",
				Labels: map[string]string{"name": "ns3"},
			},
		},
	}
	return initialObjects
}

func mock_listPodsByClusterPolicy_LabelSelectorAsSelector_err_second(c client.Client) []gomonkey.Patches {
	patch := gomonkey.ApplyFuncSeq(metav1.LabelSelectorAsSelector, []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil, nil}, Times: 1},
		{Values: gomonkey.Params{nil, errForMock}, Times: 1},
	})
	return []gomonkey.Patches{*patch}
}

func mock_listPodsByClusterPolicy_List_err_second(c client.Client) []gomonkey.Patches {
	patch := gomonkey.ApplyMethodSeq(c, "List", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{errForMock}, Times: 1},
	})
	return []gomonkey.Patches{*patch}
}

func mock_enqueueNS_List_err(c client.Client) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(c, "List", errForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_enqueueNS_LabelSelectorAsSelector_err(c client.Client) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(metav1.LabelSelectorAsSelector, nil, errForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_enqueueEGCP_List_err_one(c client.Client) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(c, "List", errForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_enqueueEGCP_List_err_two(c client.Client) []gomonkey.Patches {
	patch := gomonkey.ApplyMethodSeq(c, "List", []gomonkey.OutputCell{
		{Values: gomonkey.Params{nil}, Times: 1},
		{Values: gomonkey.Params{errForMock}, Times: 1},
	})
	return []gomonkey.Patches{*patch}
}

func mock_enqueueEGCP_Get_err(c client.Client) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(c, "Get", errForMock)

	return []gomonkey.Patches{*patch1}
}

func mock_enqueueEGCP_LabelSelectorAsSelector_err_first(c client.Client) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(metav1.LabelSelectorAsSelector, nil, errForMock)

	return []gomonkey.Patches{*patch1}
}

func mock_enqueueEGCP_LabelSelectorAsSelector_err_second(c client.Client) []gomonkey.Patches {
	sel, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "testpod"},
	})
	patch := gomonkey.ApplyFuncSeq(metav1.LabelSelectorAsSelector, []gomonkey.OutputCell{
		{Values: gomonkey.Params{sel, nil}, Times: 1},
		{Values: gomonkey.Params{nil, errForMock}, Times: 1},
	})
	return []gomonkey.Patches{*patch}
}
