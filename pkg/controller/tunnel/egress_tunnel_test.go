// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package tunnel

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/cilium/ipam/service/ipallocator"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/markallocator"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

type TestNodeReq struct {
	nn         types.NamespacedName
	expErr     bool
	expRequeue bool
}

func TestEgressTunnelCtrlForEgressTunnel(t *testing.T) {
	cfg := &config.Config{
		EnvConfig:  config.EnvConfig{},
		FileConfig: config.FileConfig{EnableIPv4: true, EnableIPv6: false},
	}

	initialObjects := []client.Object{
		&corev1.Node{ObjectMeta: v1.ObjectMeta{Name: "node1"}},
		&egressv1.EgressTunnel{
			ObjectMeta: v1.ObjectMeta{Name: "node1"},
			Status:     egressv1.EgressTunnelStatus{},
		},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	builder.WithObjects(initialObjects...)
	builder.WithStatusSubresource(initialObjects...)

	mark, err := markallocator.NewAllocatorMarkRange("0x26000000")
	if err != nil {
		t.Fatal(err)
	}

	_, cidr, err := net.ParseCIDR("10.6.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	allocatorV4, err := ipallocator.NewCIDRRange(cidr)
	if err != nil {
		t.Fatal(err)
	}

	reconciler := egReconciler{
		client:      builder.Build(),
		log:         logger.NewLogger(cfg.EnvConfig.Logger),
		config:      cfg,
		mark:        mark,
		allocatorV4: allocatorV4,
		allocatorV6: nil,
		initDone:    make(chan struct{}, 1),
	}

	reqs := []TestNodeReq{
		{
			nn:         types.NamespacedName{Namespace: "EgressTunnel/", Name: "node1"},
			expErr:     false,
			expRequeue: false,
		},
	}
	ctx := context.Background()
	for _, req := range reqs {
		res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: req.nn})
		if !req.expErr {
			assert.NoError(t, err)
		}
		assert.Equal(t, req.expRequeue, res.Requeue)
	}

	egressTunnel := &egressv1.EgressTunnel{}

	err = reconciler.client.Get(ctx, types.NamespacedName{Name: "node1"}, egressTunnel)
	if err != nil {
		t.Fatal(err)
	}

	if egressTunnel.Status.Mark == "" {
		t.Fatal("mark is empty")
	}
	if egressTunnel.Status.Tunnel.MAC == "" {
		t.Fatal("mac is empty")
	}
	if egressTunnel.Status.Tunnel.IPv4 == "" {
		t.Fatal("ipv4 is empty")
	}

	err = reconciler.client.Delete(ctx, egressTunnel)
	if err != nil {
		t.Fatal(err)
	}

	err = reconciler.client.Get(ctx, types.NamespacedName{Name: "node1"}, egressTunnel)
	if err != nil {
	} else {
		t.Fatal("expect deleted egress tunnel, but got one")
	}
}

func TestEgressTunnelCtrlForNode(t *testing.T) {
	cfg := &config.Config{}
	node := &corev1.Node{ObjectMeta: v1.ObjectMeta{Name: "node1"}}
	initialObjects := []client.Object{node}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	builder.WithObjects(initialObjects...)

	mark, err := markallocator.NewAllocatorMarkRange("0x26000000")
	if err != nil {
		t.Fatal(err)
	}

	_, cidr, _ := net.ParseCIDR("10.6.0.0/24")
	allocatorV4, _ := ipallocator.NewCIDRRange(cidr)
	_, cidr, _ = net.ParseCIDR("fd00::/24")
	allocatorV6, _ := ipallocator.NewCIDRRange(cidr)

	reconciler := egReconciler{
		client:      builder.Build(),
		log:         logger.NewLogger(cfg.EnvConfig.Logger),
		config:      cfg,
		mark:        mark,
		allocatorV4: allocatorV4,
		allocatorV6: allocatorV6,
		initDone:    make(chan struct{}, 1),
	}

	reqs := []TestNodeReq{
		{
			nn:         types.NamespacedName{Namespace: "Node/", Name: "node1"},
			expErr:     false,
			expRequeue: false,
		},
	}
	ctx := context.Background()
	for _, req := range reqs {
		res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: req.nn})
		if !req.expErr {
			assert.NoError(t, err)
		}
		assert.Equal(t, req.expRequeue, res.Requeue)
	}

	egressTunnel := &egressv1.EgressTunnel{}
	err = reconciler.client.Get(ctx, types.NamespacedName{Name: "node1"}, egressTunnel)
	if err != nil {
		t.Fatal(err)
	}

	err = reconciler.client.Delete(ctx, node)
	if err != nil {
		t.Fatal(err)
	}

	for _, req := range reqs {
		res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: req.nn})
		if !req.expErr {
			assert.NoError(t, err)
		}
		assert.Equal(t, req.expRequeue, res.Requeue)
	}

	err = reconciler.client.Get(ctx, types.NamespacedName{Name: "node1"}, egressTunnel)
	if err != nil {
	} else if egressTunnel.DeletionTimestamp.IsZero() {
		t.Fatal("expect deleted egress tunnel, but got one")
	}
}

func TestCleanFinalizers(t *testing.T) {
	tests := []struct {
		name           string
		node           *egressv1.EgressTunnel
		wantFinalizers []string
	}{
		{
			name: "remove egressTunnelFinalizers",
			node: &egressv1.EgressTunnel{
				ObjectMeta: v1.ObjectMeta{
					Finalizers: []string{
						"keep-this",
						"keep-that",
						"egressgateway.spidernet.io/egresstunnel",
					},
				},
			},
			wantFinalizers: []string{"keep-this", "keep-that"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanFinalizers(tc.node)
			if !slicesEqual(tc.node.Finalizers, tc.wantFinalizers) {
				t.Errorf("cleanFinalizers() got = %v, want %v", tc.node.Finalizers, tc.wantFinalizers)
			}
		})
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestHealthCheck(t *testing.T) {
	cfg := &config.Config{
		FileConfig: config.FileConfig{
			GatewayFailover: config.GatewayFailover{
				Enable:              true,
				TunnelMonitorPeriod: 1,
			},
		},
	}
	initialObjects := []client.Object{
		&egressv1.EgressTunnel{
			ObjectMeta: v1.ObjectMeta{Name: "node1"},
		},
		&egressv1.EgressTunnel{
			ObjectMeta: v1.ObjectMeta{Name: "node2"},
		},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	builder.WithObjects(initialObjects...)

	mark, err := markallocator.NewAllocatorMarkRange("0x26000000")
	if err != nil {
		t.Fatal(err)
	}

	_, cidr, _ := net.ParseCIDR("10.6.0.0/24")
	allocatorV4, _ := ipallocator.NewCIDRRange(cidr)
	_, cidr, _ = net.ParseCIDR("fd00::/24")
	allocatorV6, _ := ipallocator.NewCIDRRange(cidr)

	reconciler := &egReconciler{
		client:      builder.Build(),
		log:         logger.NewLogger(cfg.EnvConfig.Logger),
		config:      cfg,
		mark:        mark,
		allocatorV4: allocatorV4,
		allocatorV6: allocatorV6,
		initDone:    make(chan struct{}, 1),
	}

	reconciler.initDone <- struct{}{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err = reconciler.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 6)
}

func TestNewEgressTunnelController(t *testing.T) {
	labels := map[string]string{"app": "nginx1"}
	initialObjects := []client.Object{
		&egressv1.EgressClusterPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name: "p1",
			},
			Spec: egressv1.EgressClusterPolicySpec{
				AppliedTo: egressv1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{MatchLabels: labels},
				},
			},
		},
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
			TunnelIpv4Subnet:          "10.6.1.21/24",
			TunnelIpv6Subnet:          "fd00::/126",
			EnableIPv4:                true,
			EnableIPv6:                true,
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
	err = NewEgressTunnelController(mgr, log, cfg)
	if err != nil {
		t.Fatal(err)
	}
}
