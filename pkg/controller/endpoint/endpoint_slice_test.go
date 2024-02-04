// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package endpoint

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

type TestCaseEPS struct {
	initialObjects []client.Object
	reqs           []TestEgressGatewayPolicyReq
	config         *config.Config
}

type TestEgressGatewayPolicyReq struct {
	nn         types.NamespacedName
	expErr     bool
	expRequeue bool
}

func TestReconcilerEgressEndpointSlice(t *testing.T) {
	cases := map[string]TestCaseEPS{
		"caseAddEgressGatewayPolicy": caseAddPolicy(),
		"caseUpdatePod":              caseUpdatePod(),
		"caseDeletePod":              caseDeletePod(),
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			builder.WithObjects(c.initialObjects...)
			cli := builder.Build()
			reconciler := endpointReconciler{
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
				policy := new(v1beta1.EgressPolicy)
				err = cli.Get(ctx, req.nn, policy)
				if err != nil {
					t.Fatal(err)
				}

				epList, err := listEndpointSlices(ctx, cli, policy.Namespace, policy.Name)
				if err != nil {
					t.Fatal(err)
				}

				pods, err := listPodsByPolicy(ctx, cli, policy)
				if err != nil {
					t.Fatal(err)
				}

				err = checkPolicyIPsInEpSlice(pods, epList)
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func checkPolicyIPsInEpSlice(pods *corev1.PodList, eps *v1beta1.EgressEndpointSliceList) error {
	if pods == nil || eps == nil {
		return nil
	}
	podMap := make(map[string]struct{})
	epMap := make(map[string]struct{})
	for _, pod := range pods.Items {
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

func compareMaps(podMap, epMap map[string]struct{}) error {
	var missingPods, missingEps []string

	for k := range podMap {
		if _, ok := epMap[k]; !ok {
			missingEps = append(missingEps, k)
		}
	}

	for k := range epMap {
		if _, ok := podMap[k]; !ok {
			missingPods = append(missingPods, k)
		}
	}

	if len(missingPods) > 0 || len(missingEps) > 0 {
		var msg string
		if len(missingPods) > 0 {
			msg += fmt.Sprintf("missing endpoints for pods: %v\n", missingPods)
		}
		if len(missingEps) > 0 {
			msg += fmt.Sprintf("missing pods ip for endpoints: %v\n", missingEps)
		}
		return errors.New(msg)
	}

	return nil
}

func caseAddPolicy() TestCaseEPS {
	labels := map[string]string{"app": "nginx1"}
	initialObjects := []client.Object{
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "policy1",
				Namespace: "default",
			},
			Spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.AppliedTo{
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
			nn:         types.NamespacedName{Namespace: "default", Name: "policy1"},
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

func caseUpdatePod() TestCaseEPS {
	labels := map[string]string{"app": "nginx1"}
	initialObjects := []client.Object{
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "policy1",
				Namespace: "default",
			},
			Spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
				DestSubnet: nil,
			},
		},
		&v1beta1.EgressEndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "s1",
				Namespace: "default",
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
			nn:         types.NamespacedName{Namespace: "default", Name: "policy1"},
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

func caseDeletePod() TestCaseEPS {
	labels := map[string]string{"app": "nginx1"}
	initialObjects := []client.Object{
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "policy1",
				Namespace: "default",
			},
			Spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
				DestSubnet: nil,
			},
		},
		&v1beta1.EgressEndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "s1",
				Namespace: "default",
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
				Name:      "s2",
				Namespace: "default",
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
			nn:         types.NamespacedName{Namespace: "default", Name: "policy1"},
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

func TestPodPredicate(t *testing.T) {
	p := podPredicate{}
	if !p.Create(event.CreateEvent{
		Object: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
				Labels: map[string]string{
					"aa": "bb",
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.6.1.21",
				PodIPs: []corev1.PodIP{
					{IP: "10.6.1.21"},
				},
			},
		},
	}) {
		t.Fatal("got false")
	}

	if !p.Delete(event.DeleteEvent{}) {
		t.Fatal("got false")
	}

	if !p.Update(event.UpdateEvent{
		ObjectOld: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
				Labels: map[string]string{
					"aa": "bb",
				},
			},
		},
		ObjectNew: &corev1.Pod{
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

func TestEnqueuePod(t *testing.T) {
	labels := map[string]string{"app": "nginx1"}
	initialObjects := []client.Object{
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "p1",
			},
			Spec: v1beta1.EgressPolicySpec{
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
			},
		},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	builder.WithObjects(initialObjects...)
	cli := builder.Build()

	f := enqueuePod(cli)
	ctx := context.Background()

	f(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
			Labels:    labels,
		},
	})
}

func TestNewEgressEndpointSliceController(t *testing.T) {
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
	err = NewEgressEndpointSliceController(mgr, log, cfg)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_NewEgressEndpointSliceController(t *testing.T) {
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

		"failed controller watch egressPolilcy": {
			patchFun: mock_NewEgressClusterEpSliceController_Watch_namespace_err,
			expErr:   true,
		},
		"failed controller watch egressEndpointSlice": {
			patchFun: mock_NewEgressClusterEpSliceController_Watch_clusterpolicy_err,
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
			r := &endpointReconciler{
				client: mgr.GetClient(),
				log:    log,
				config: cfg,
			}

			// patch
			patches := make([]gomonkey.Patches, 0)
			if tc.patchFun != nil {
				patches = tc.patchFun(t, r, mgr, log)
			}

			err = NewEgressEndpointSliceController(mgr, log, cfg)

			if tc.expErr {
				assert.Error(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_endpointReconciler_Reconcile(t *testing.T) {
	cases := map[string]struct {
		setMaxNumberEndpointPerSlice func(*endpointReconciler)
		setObjs                      func() ([]client.Object, reconcile.Request)
		patchFun                     func(*testing.T, *endpointReconciler, manager.Manager, logr.Logger) []gomonkey.Patches
		expErr                       bool
	}{
		"failed Get policy": {
			patchFun: mock_endpointReconciler_Reconcile_Get_err,
			expErr:   true,
		},
		"failed Get policy notFound": {
			patchFun: mock_endpointReconciler_Reconcile_Get_err_notFound,
		},

		"failed listPodsByPolicy": {
			patchFun: mock_listPodsByPolicy_err,
			expErr:   true,
		},

		"failed listEndpointSlices": {
			patchFun: mock_listEndpointSlices_err,
			expErr:   true,
		},

		" needUpdateEndpoint true": {
			setObjs:  mock_policyObjs,
			patchFun: mock_endpointReconciler_needUpdateEndpoint_true,
		},

		"need to CreateEgressEndpointSlice": {
			setObjs: mock_policyObjs_need_create_endpoint,
		},

		"need to CreateEgressEndpointSlice less count": {
			setMaxNumberEndpointPerSlice: mock_endpointReconciler_MaxNumberEndpointPerSlice_less_count,
			setObjs:                      mock_policyObjs_need_create_endpoint,
		},

		"need to CreateEgressEndpointSlice but sliceNotChange": {
			setObjs: mock_policyObjs_not_change,
		},

		"need to delete endpoint": {
			setObjs: mock_policyObjs_need_delete_endpoint,
		},

		"failed to update endpoint": {
			setObjs:  mock_policyObjs,
			patchFun: mock_endpointReconciler_Reconcile_Update_err,
			expErr:   true,
		},

		"failed to create endpoint": {
			setObjs:  mock_policyObjs_no_endpoint,
			patchFun: mock_endpointReconciler_Reconcile_Create_err,
			expErr:   true,
		},

		"failed to delete endpoint": {
			setObjs:  mock_policyObjs_need_delete_endpoint,
			patchFun: mock_endpointReconciler_Reconcile_Delete_err,
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
			r := &endpointReconciler{
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

func Test_getEndpointSlicePrefix(t *testing.T) {
	t.Run("len(validation.NameIsDNSSubdomain(prefix, true)) is not zero", func(t *testing.T) {
		p := gomonkey.ApplyFuncReturn(validation.NameIsDNSSubdomain, []string{"1", "2"})
		defer p.Reset()
		getEndpointSlicePrefix("xxx")
	})
}

func Test_newEndpoint(t *testing.T) {
	t.Run("ipv6", func(t *testing.T) {
		newEndpoint(corev1.Pod{
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "fddd:dd::2"},
				},
			},
		})
	})
	t.Run("no ip", func(t *testing.T) {
		newEndpoint(corev1.Pod{})
	})
}

func Test_needUpdateEndpoint(t *testing.T) {
	t.Run("ipv6", func(t *testing.T) {
		needUpdateEndpoint(corev1.Pod{
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "fddd:dd::2"},
				},
			},
		}, &v1beta1.EgressEndpoint{})
	})
	t.Run("need update ipv4", func(t *testing.T) {
		needUpdateEndpoint(corev1.Pod{
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "10.10.0.2"},
				},
			},
		}, &v1beta1.EgressEndpoint{})
	})
	t.Run("need update ipv6", func(t *testing.T) {
		needUpdateEndpoint(corev1.Pod{
			Status: corev1.PodStatus{
				PodIPs: []corev1.PodIP{
					{IP: "fddd:dd::2"},
				},
			},
		}, &v1beta1.EgressEndpoint{})
	})
}

func Test_sliceEqual(t *testing.T) {
	t.Run("length not equal", func(t *testing.T) {
		sliceEqual([]string{"x"}, []string{"x", "xx"})
	})
	t.Run("slice not equal", func(t *testing.T) {
		sliceEqual([]string{"x1"}, []string{"x2"})
	})
}

func Test_initEndpoint(t *testing.T) {
	t.Run("length not equal", func(t *testing.T) {
		builder := fake.NewClientBuilder()
		builder.WithScheme(schema.GetScheme())
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
		r := &endpointReconciler{
			client: mgr.GetClient(),
			log:    log,
			config: cfg,
		}
		e := r.initEndpoint()
		assert.NoError(t, e)
	})
}

func Test_listPodsByPolicy(t *testing.T) {
	cases := map[string]struct {
		setObjs   func() []client.Object
		setParams func() *v1beta1.EgressPolicy
		patchFun  func(c client.Client) []gomonkey.Patches
		expErr    bool
	}{
		"failed LabelSelectorAsSelector when nil namespaceSelector": {
			setParams: mock_listPodsByPolicy,
			patchFun:  mock_listPodsByPolicy_LabelSelectorAsSelector_err,
			expErr:    true,
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
			_, err := listPodsByPolicy(context.TODO(), cli, policy)

			if tc.expErr {
				assert.Error(t, err)
			}
			for _, p := range patches {
				p.Reset()
			}
		})
	}
}

func Test_listEndpointSlices(t *testing.T) {
	t.Run("failed to LabelSelectorAsSelector", func(t *testing.T) {
		p := gomonkey.ApplyFuncReturn(metav1.LabelSelectorAsSelector, nil, errForMock)
		defer p.Reset()

		builder := fake.NewClientBuilder()
		builder.WithScheme(schema.GetScheme())
		cli := builder.Build()
		_, err := listEndpointSlices(context.TODO(), cli, "testns", "testPolicy")
		assert.Error(t, err)
	})
}

func Test_podPredicate_Create(t *testing.T) {
	cases := map[string]struct {
		in  event.CreateEvent
		res bool
	}{
		"createEvent not pod": {
			in: event.CreateEvent{},
		},
		"pod no ip": {
			in: event.CreateEvent{
				Object: &corev1.Pod{},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := podPredicate{}

			ok := p.Create(tc.in)
			if tc.res {
				assert.True(t, ok)
			} else {
				assert.False(t, ok)
			}
		})
	}
}

func Test_podPredicate_Update(t *testing.T) {
	cases := map[string]struct {
		in  event.UpdateEvent
		res bool
	}{
		"ObjectOld not pod": {
			in: event.UpdateEvent{},
		},
		"ObjectNew not pod": {
			in: event.UpdateEvent{
				ObjectOld: &corev1.Pod{},
			},
		},
		"nodeName not equal": {
			in: event.UpdateEvent{
				ObjectOld: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"foo": "bar"},
					},
					Spec: corev1.PodSpec{
						NodeName: "node1",
					},
				},
				ObjectNew: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"foo": "bar"},
					},
					Spec: corev1.PodSpec{
						NodeName: "node2",
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p := podPredicate{}

			ok := p.Update(tc.in)
			if tc.res {
				assert.True(t, ok)
			} else {
				assert.False(t, ok)
			}
		})
	}
}

func Test_podPredicate_Generic(t *testing.T) {
	t.Run("test Generic", func(t *testing.T) {
		p := podPredicate{}
		e := event.GenericEvent{}
		res := p.Generic(e)
		assert.True(t, res)
	})
}
func Test_enqueuePod(t *testing.T) {
	cases := map[string]struct {
		in       client.Object
		objs     []client.Object
		patchFun func(c client.Client) []gomonkey.Patches
		expErr   bool
	}{
		"failed not pod obj": {
			in:     &corev1.Namespace{},
			expErr: true,
		},
		"failed List": {
			in:       &corev1.Pod{},
			patchFun: mock_enqueuePod_List_err,
			expErr:   true,
		},
		"failed LabelSelectorAsSelector": {
			in: &corev1.Pod{},
			objs: []client.Object{
				&v1beta1.EgressPolicy{},
			},
			patchFun: mock_enqueuePod_LabelSelectorAsSelector_err,
			expErr:   true,
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

			resF := enqueuePod(cli)
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

func mock_endpointReconciler_Reconcile_Get_err(t *testing.T, r *endpointReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", errForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_endpointReconciler_Reconcile_Get_err_notFound(t *testing.T, r *endpointReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", errForMock)
	patch2 := gomonkey.ApplyFuncReturn(apierrors.IsNotFound, true)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_listPodsByPolicy_err(t *testing.T, r *endpointReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", nil)
	patch2 := gomonkey.ApplyFuncReturn(listPodsByPolicy, nil, errForMock)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_listEndpointSlices_err(t *testing.T, r *endpointReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Get", nil)
	patch2 := gomonkey.ApplyFuncReturn(listPodsByPolicy, &corev1.PodList{}, nil)
	patch3 := gomonkey.ApplyFuncReturn(listEndpointSlices, nil, errForMock)

	return []gomonkey.Patches{*patch1, *patch2, *patch3}
}

func mock_policyObjs() ([]client.Object, reconcile.Request) {
	labels := map[string]string{"app": "nginx1"}
	labels2 := map[string]string{"app": "nginx2"}
	policyName := "policy1"
	policyName2 := "policy2"

	initialObjects := []client.Object{
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName,
			},
			Spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
				DestSubnet: nil,
			},
		},
		&v1beta1.EgressEndpointSlice{
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
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName2,
			},
			Spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels2},
				},
				DestSubnet: nil,
			},
		},
		&v1beta1.EgressEndpointSlice{
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
		&v1beta1.EgressEndpointSlice{
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

func mock_endpointReconciler_needUpdateEndpoint_true(t *testing.T, r *endpointReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(needUpdateEndpoint, true)

	return []gomonkey.Patches{*patch1}
}

func mock_policyObjs_need_create_endpoint() ([]client.Object, reconcile.Request) {
	labels1 := map[string]string{"app": "nginx1"}
	labels2 := map[string]string{"app": "nginx2"}
	policyName := "policy1"
	policyName2 := "policy2"
	policyName3 := "policy3"
	initialObjects := []client.Object{
		&v1beta1.EgressEndpointSlice{
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
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName,
			},
			Spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels2},
				},
				DestSubnet: nil,
			},
		},
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName2,
			},
			Spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.AppliedTo{
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
		&v1beta1.EgressEndpointSlice{
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
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName3,
			},
			Spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.AppliedTo{
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

func mock_endpointReconciler_MaxNumberEndpointPerSlice_less_count(r *endpointReconciler) {
	r.config.FileConfig.MaxNumberEndpointPerSlice = 1
}

func mock_policyObjs_not_change() ([]client.Object, reconcile.Request) {
	labels := map[string]string{"app": "nginx1"}
	policyName := "policy1"

	initialObjects := []client.Object{
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName,
			},
			Spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
				DestSubnet: nil,
			},
		},
		&v1beta1.EgressEndpointSlice{
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

func mock_policyObjs_need_delete_endpoint() ([]client.Object, reconcile.Request) {
	labels := map[string]string{"app": "nginx1"}
	policyName := "policy1"

	initialObjects := []client.Object{
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName,
			},
			Spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
				DestSubnet: nil,
			},
		},
		&v1beta1.EgressEndpointSlice{
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

func mock_endpointReconciler_Reconcile_Update_err(t *testing.T, r *endpointReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Update", errForMock)
	patch2 := gomonkey.ApplyFuncReturn(needUpdateEndpoint, true)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_policyObjs_no_endpoint() ([]client.Object, reconcile.Request) {
	labels1 := map[string]string{"app": "nginx1"}
	policyName := "policy1"
	initialObjects := []client.Object{
		&v1beta1.EgressPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: policyName,
			},
			Spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.AppliedTo{
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

func mock_endpointReconciler_Reconcile_Create_err(t *testing.T, r *endpointReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Create", errForMock)
	patch2 := gomonkey.ApplyFuncReturn(needUpdateEndpoint, false)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_endpointReconciler_Reconcile_Delete_err(t *testing.T, r *endpointReconciler, mgr manager.Manager, log logr.Logger) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(r.client, "Delete", errForMock)
	patch2 := gomonkey.ApplyFuncReturn(needUpdateEndpoint, false)
	return []gomonkey.Patches{*patch1, *patch2}
}

func mock_listPodsByPolicy() *v1beta1.EgressPolicy {
	return &v1beta1.EgressPolicy{
		Spec: v1beta1.EgressPolicySpec{
			AppliedTo: v1beta1.AppliedTo{
				PodSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"foo": "bar"},
				},
			},
		},
	}
}

func mock_listPodsByPolicy_LabelSelectorAsSelector_err(c client.Client) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(metav1.LabelSelectorAsSelector, nil, errForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_enqueuePod_List_err(c client.Client) []gomonkey.Patches {
	patch1 := gomonkey.ApplyMethodReturn(c, "List", errForMock)
	return []gomonkey.Patches{*patch1}
}

func mock_enqueuePod_LabelSelectorAsSelector_err(c client.Client) []gomonkey.Patches {
	patch1 := gomonkey.ApplyFuncReturn(metav1.LabelSelectorAsSelector, nil, errForMock)
	return []gomonkey.Patches{*patch1}
}
