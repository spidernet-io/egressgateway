// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"net"
	"testing"

	"github.com/cilium/ipam/service/ipallocator"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

func TestEgressNodeCtrlForEgressNode(t *testing.T) {
	cfg := &config.Config{
		EnvConfig:  config.EnvConfig{},
		FileConfig: config.FileConfig{EnableIPv4: true, EnableIPv6: false},
	}

	initialObjects := []client.Object{
		&corev1.Node{ObjectMeta: v1.ObjectMeta{Name: "node1"}},
		&egressv1.EgressTunnel{
			ObjectMeta: v1.ObjectMeta{Name: "node1"},
			Status:     egressv1.EgressNodeStatus{},
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

	egressNode := &egressv1.EgressTunnel{}

	err = reconciler.client.Get(ctx, types.NamespacedName{Name: "node1"}, egressNode)
	if err != nil {
		t.Fatal(err)
	}

	if egressNode.Status.Mark == "" {
		t.Fatal("mark is empty")
	}
	if egressNode.Status.Tunnel.MAC == "" {
		t.Fatal("mac is empty")
	}
	if egressNode.Status.Tunnel.IPv4 == "" {
		t.Fatal("ipv4 is empty")
	}

	err = reconciler.client.Delete(ctx, egressNode)
	if err != nil {
		t.Fatal(err)
	}

	err = reconciler.client.Get(ctx, types.NamespacedName{Name: "node1"}, egressNode)
	if err != nil {
	} else {
		t.Fatal("expect deleted egress node, but got one")
	}
}

func TestEgressNodeCtrlForNode(t *testing.T) {
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

	egressNode := &egressv1.EgressTunnel{}
	err = reconciler.client.Get(ctx, types.NamespacedName{Name: "node1"}, egressNode)
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

	err = reconciler.client.Get(ctx, types.NamespacedName{Name: "node1"}, egressNode)
	if err != nil {
	} else if egressNode.DeletionTimestamp.IsZero() {
		t.Fatal("expect deleted egress node, but got one")
	}
}
