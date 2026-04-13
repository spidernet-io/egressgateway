// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

type statusClient struct {
	client.Client
	status client.SubResourceWriter
}

func (c *statusClient) Status() client.SubResourceWriter {
	return c.status
}

type blockingStatusWriter struct {
	client.SubResourceWriter

	entered chan struct{}
	release chan struct{}

	mu          sync.Mutex
	updateCalls int
	active      int
	maxActive   int
}

func newBlockingStatusWriter(base client.SubResourceWriter) *blockingStatusWriter {
	return &blockingStatusWriter{
		SubResourceWriter: base,
		entered:           make(chan struct{}),
		release:           make(chan struct{}),
	}
}

func (w *blockingStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	w.mu.Lock()
	w.updateCalls++
	callNum := w.updateCalls
	w.active++
	if w.active > w.maxActive {
		w.maxActive = w.active
	}
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.active--
	}()

	if callNum == 1 {
		close(w.entered)
		<-w.release
	}

	return w.SubResourceWriter.Update(ctx, obj, opts...)
}

func (w *blockingStatusWriter) stats() (int, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.updateCalls, w.maxActive
}

func newTestReconciler(t *testing.T, retryInterval time.Duration) (*vxlanReconciler, *blockingStatusWriter) {
	t.Helper()

	tunnel := &egressv1.EgressTunnel{
		ObjectMeta: metav1.ObjectMeta{Name: "node1"},
		Status:     egressv1.EgressTunnelStatus{},
	}

	builder := fake.NewClientBuilder()
	builder.WithScheme(schema.GetScheme())
	builder.WithObjects(tunnel)
	builder.WithStatusSubresource(tunnel)

	baseClient := builder.Build()
	writer := newBlockingStatusWriter(baseClient.Status())
	reconciler := &vxlanReconciler{
		client: &statusClient{
			Client: baseClient,
			status: writer,
		},
		log: logger.NewLogger(logger.Config{}),
		cfg: &config.Config{
			EnvConfig: config.EnvConfig{NodeName: "node1"},
			FileConfig: config.FileConfig{
				GatewayFailover: config.GatewayFailover{EipEvictionTimeout: 1},
			},
		},
		updateRetryInterval: retryInterval,
	}

	return reconciler, writer
}

func TestTriggerTunnelStatusUpdateSerializesConcurrentCalls(t *testing.T) {
	reconciler, writer := newTestReconciler(t, 10*time.Millisecond)

	firstErr := make(chan error, 1)
	go func() {
		firstErr <- reconciler.triggerTunnelStatusUpdate(context.Background())
	}()

	select {
	case <-writer.entered:
	case <-time.After(time.Second):
		t.Fatal("first tunnel status update did not start")
	}

	secondErr := make(chan error, 1)
	start := time.Now()
	go func() {
		secondErr <- reconciler.triggerTunnelStatusUpdate(context.Background())
	}()

	select {
	case err := <-secondErr:
		t.Fatalf("second tunnel status update should wait for the running task, got: %v", err)
	case <-time.After(30 * time.Millisecond):
	}

	close(writer.release)

	if err := <-firstErr; err != nil {
		t.Fatalf("first tunnel status update failed: %v", err)
	}
	if err := <-secondErr; err != nil {
		t.Fatalf("second tunnel status update failed: %v", err)
	}

	updateCalls, maxActive := writer.stats()
	if updateCalls != 2 {
		t.Fatalf("expected 2 serialized status updates, got %d", updateCalls)
	}
	if maxActive != 1 {
		t.Fatalf("expected tunnel status updates to run one at a time, max active=%d", maxActive)
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("expected second update to wait before running, elapsed=%s", elapsed)
	}

	tunnel := new(egressv1.EgressTunnel)
	if err := reconciler.client.Get(context.Background(), client.ObjectKey{Name: "node1"}, tunnel); err != nil {
		t.Fatalf("get tunnel: %v", err)
	}
	if tunnel.Status.LastHeartbeatTime.IsZero() {
		t.Fatal("expected tunnel heartbeat time to be updated")
	}
}

func TestTriggerTunnelStatusUpdateReturnsContextErrorWhileWaiting(t *testing.T) {
	reconciler, writer := newTestReconciler(t, 50*time.Millisecond)

	firstErr := make(chan error, 1)
	go func() {
		firstErr <- reconciler.triggerTunnelStatusUpdate(context.Background())
	}()

	select {
	case <-writer.entered:
	case <-time.After(time.Second):
		t.Fatal("first tunnel status update did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := reconciler.triggerTunnelStatusUpdate(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded while waiting for running update, got %v", err)
	}

	close(writer.release)
	if err := <-firstErr; err != nil {
		t.Fatalf("first tunnel status update failed: %v", err)
	}
}
