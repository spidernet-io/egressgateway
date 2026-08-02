// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package clusterinfo

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	calicov1 "github.com/tigera/operator/pkg/apis/crd.projectcalico.org/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

func TestReconcileNodeUpdatesK8sPodCIDRInAutoMode(t *testing.T) {
	info := &egressv1.EgressClusterInfo{
		ObjectMeta: metav1.ObjectMeta{Name: defaultName},
		Spec: egressv1.EgressClusterInfoSpec{
			AutoDetect: egressv1.AutoDetect{
				NodeIP:      true,
				PodCidrMode: egressv1.CniTypeAuto,
			},
		},
	}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Spec: corev1.NodeSpec{
			PodCIDR:  "10.244.1.0/24",
			PodCIDRs: []string{"10.244.1.0/24", "fd00:10:244:1::/64"},
		},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "10.20.10.21"},
			},
		},
	}

	objects := []client.Object{info, node}
	builder := fake.NewClientBuilder().
		WithScheme(schema.GetScheme()).
		WithObjects(objects...).
		WithStatusSubresource(info)
	reconciler := &clusterInfo{cli: builder.Build()}

	err := reconciler.reconcileNode(
		context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: node.Name}},
		logr.Discard(),
	)
	if !assert.NoError(t, err) {
		return
	}

	updated := new(egressv1.EgressClusterInfo)
	err = reconciler.cli.Get(context.Background(), client.ObjectKey{Name: defaultName}, updated)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, []string{"10.244.1.0/24"}, updated.Status.PodCIDR[node.Name].IPv4)
	assert.Equal(t, []string{"fd00:10:244:1::/64"}, updated.Status.PodCIDR[node.Name].IPv6)
}

func TestReconcileNodeUpdatesK8sPodCIDRWhenNodeIPDetectionIsDisabled(t *testing.T) {
	info := &egressv1.EgressClusterInfo{
		ObjectMeta: metav1.ObjectMeta{Name: defaultName},
		Spec: egressv1.EgressClusterInfoSpec{
			AutoDetect: egressv1.AutoDetect{
				NodeIP:      false,
				PodCidrMode: egressv1.CniTypeK8s,
			},
		},
	}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Spec: corev1.NodeSpec{
			PodCIDR: "10.244.1.0/24",
		},
	}

	objects := []client.Object{info, node}
	builder := fake.NewClientBuilder().
		WithScheme(schema.GetScheme()).
		WithObjects(objects...).
		WithStatusSubresource(info)
	reconciler := &clusterInfo{cli: builder.Build()}

	err := reconciler.reconcileNode(
		context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: node.Name}},
		logr.Discard(),
	)
	if !assert.NoError(t, err) {
		return
	}

	updated := new(egressv1.EgressClusterInfo)
	err = reconciler.cli.Get(context.Background(), client.ObjectKey{Name: defaultName}, updated)
	if !assert.NoError(t, err) {
		return
	}

	assert.Empty(t, updated.Status.NodeIP)
	assert.Equal(t, []string{"10.244.1.0/24"}, updated.Status.PodCIDR[node.Name].IPv4)
}

func TestReconcileNodeDoesNotAddK8sPodCIDRInCalicoMode(t *testing.T) {
	info := &egressv1.EgressClusterInfo{
		ObjectMeta: metav1.ObjectMeta{Name: defaultName},
		Spec: egressv1.EgressClusterInfoSpec{
			AutoDetect: egressv1.AutoDetect{
				NodeIP:      true,
				PodCidrMode: egressv1.CniTypeCalico,
			},
		},
	}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Spec:       corev1.NodeSpec{PodCIDRs: []string{"10.244.1.0/24"}},
		Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{
			{Type: corev1.NodeInternalIP, Address: "10.20.10.21"},
		}},
	}

	objects := []client.Object{info, node}
	builder := fake.NewClientBuilder().
		WithScheme(schema.GetScheme()).
		WithObjects(objects...).
		WithStatusSubresource(info)
	reconciler := &clusterInfo{cli: builder.Build()}

	err := reconciler.reconcileNode(
		context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: node.Name}},
		logr.Discard(),
	)
	if !assert.NoError(t, err) {
		return
	}

	updated := new(egressv1.EgressClusterInfo)
	err = reconciler.cli.Get(context.Background(), client.ObjectKey{Name: defaultName}, updated)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, []string{"10.20.10.21"}, updated.Status.NodeIP[node.Name].IPv4)
	assert.Empty(t, updated.Status.PodCIDR)
}

func TestReconcileNodeRemovesK8sPodCIDRWhenNodeIsDeleted(t *testing.T) {
	info := &egressv1.EgressClusterInfo{
		ObjectMeta: metav1.ObjectMeta{Name: defaultName},
		Spec: egressv1.EgressClusterInfoSpec{
			AutoDetect: egressv1.AutoDetect{
				NodeIP:      true,
				PodCidrMode: egressv1.CniTypeAuto,
			},
		},
		Status: egressv1.EgressClusterInfoStatus{
			NodeIP: map[string]egressv1.IPListPair{
				"worker-1": {IPv4: []string{"10.20.10.21"}},
			},
			PodCIDR: map[string]egressv1.IPListPair{
				"worker-1": {IPv4: []string{"10.244.1.0/24"}},
			},
		},
	}

	builder := fake.NewClientBuilder().
		WithScheme(schema.GetScheme()).
		WithObjects(info).
		WithStatusSubresource(info)
	reconciler := &clusterInfo{cli: builder.Build()}

	err := reconciler.reconcileNode(
		context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: "worker-1"}},
		logr.Discard(),
	)
	if !assert.NoError(t, err) {
		return
	}

	updated := new(egressv1.EgressClusterInfo)
	err = reconciler.cli.Get(context.Background(), client.ObjectKey{Name: defaultName}, updated)
	if !assert.NoError(t, err) {
		return
	}

	_, hasNodeIP := updated.Status.NodeIP["worker-1"]
	_, hasPodCIDR := updated.Status.PodCIDR["worker-1"]
	assert.False(t, hasNodeIP)
	assert.False(t, hasPodCIDR)
}

func TestReconcileInfoSyncsExistingK8sPodCIDRs(t *testing.T) {
	info := &egressv1.EgressClusterInfo{
		ObjectMeta: metav1.ObjectMeta{Name: defaultName},
		Spec: egressv1.EgressClusterInfoSpec{
			AutoDetect: egressv1.AutoDetect{PodCidrMode: egressv1.CniTypeAuto},
		},
	}
	nodes := []client.Object{
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
			Spec:       corev1.NodeSpec{PodCIDRs: []string{"10.244.1.0/24"}},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "worker-2"},
			Spec:       corev1.NodeSpec{PodCIDRs: []string{"10.244.2.0/24"}},
		},
	}
	objects := append([]client.Object{info}, nodes...)
	builder := fake.NewClientBuilder().
		WithScheme(schema.GetScheme()).
		WithObjects(objects...).
		WithStatusSubresource(info)
	reconciler := &clusterInfo{
		cli:         builder.Build(),
		watchCalico: func() {},
	}

	err := reconciler.reconcileInfo(
		context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: defaultName}},
		logr.Discard(),
	)
	if !assert.NoError(t, err) {
		return
	}

	updated := new(egressv1.EgressClusterInfo)
	err = reconciler.cli.Get(context.Background(), client.ObjectKey{Name: defaultName}, updated)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, []string{"10.244.1.0/24"}, updated.Status.PodCIDR["worker-1"].IPv4)
	assert.Equal(t, []string{"10.244.2.0/24"}, updated.Status.PodCIDR["worker-2"].IPv4)
	assert.Equal(t, egressv1.CniTypeK8s, updated.Status.PodCidrMode)
}

func TestReconcileInfoRemovesK8sPodCIDRsWhenSwitchingToCalicoMode(t *testing.T) {
	info := &egressv1.EgressClusterInfo{
		ObjectMeta: metav1.ObjectMeta{Name: defaultName},
		Spec: egressv1.EgressClusterInfoSpec{
			AutoDetect: egressv1.AutoDetect{PodCidrMode: egressv1.CniTypeCalico},
		},
		Status: egressv1.EgressClusterInfoStatus{
			PodCIDR: map[string]egressv1.IPListPair{
				"worker-1":          {IPv4: []string{"10.244.1.0/24"}},
				"default-ipv4-pool": {IPv4: []string{"10.244.0.0/16"}},
			},
		},
	}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Spec:       corev1.NodeSpec{PodCIDRs: []string{"10.244.1.0/24"}},
	}
	pool := &calicov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{Name: "default-ipv4-pool"},
		Spec:       calicov1.IPPoolSpec{CIDR: "10.244.0.0/16"},
	}
	builder := fake.NewClientBuilder().
		WithScheme(schema.GetScheme()).
		WithObjects(info, node, pool).
		WithStatusSubresource(info)
	reconciler := &clusterInfo{
		cli:         builder.Build(),
		watchCalico: func() {},
	}

	err := reconciler.reconcileInfo(
		context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: defaultName}},
		logr.Discard(),
	)
	if !assert.NoError(t, err) {
		return
	}

	updated := new(egressv1.EgressClusterInfo)
	err = reconciler.cli.Get(context.Background(), client.ObjectKey{Name: defaultName}, updated)
	if !assert.NoError(t, err) {
		return
	}

	_, hasNodePodCIDR := updated.Status.PodCIDR["worker-1"]
	assert.False(t, hasNodePodCIDR)
	assert.Equal(t, []string{"10.244.0.0/16"}, updated.Status.PodCIDR["default-ipv4-pool"].IPv4)
	assert.Equal(t, egressv1.CniTypeCalico, updated.Status.PodCidrMode)
}

func TestReconcileInfoAutoPrefersCalicoIPPool(t *testing.T) {
	info := &egressv1.EgressClusterInfo{
		ObjectMeta: metav1.ObjectMeta{Name: defaultName},
		Spec: egressv1.EgressClusterInfoSpec{
			AutoDetect: egressv1.AutoDetect{PodCidrMode: egressv1.CniTypeAuto},
		},
		Status: egressv1.EgressClusterInfoStatus{
			PodCidrMode: egressv1.CniTypeK8s,
			PodCIDR: map[string]egressv1.IPListPair{
				"worker-1": {IPv4: []string{"10.244.1.0/24"}},
			},
		},
	}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Spec:       corev1.NodeSpec{PodCIDRs: []string{"10.244.1.0/24"}},
	}
	pool := &calicov1.IPPool{
		ObjectMeta: metav1.ObjectMeta{Name: "default-ipv4-pool"},
		Spec:       calicov1.IPPoolSpec{CIDR: "192.168.0.0/16"},
	}
	builder := fake.NewClientBuilder().
		WithScheme(schema.GetScheme()).
		WithObjects(info, node, pool).
		WithStatusSubresource(info)
	reconciler := &clusterInfo{
		cli:         builder.Build(),
		watchCalico: func() {},
	}

	err := reconciler.reconcileInfo(
		context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: defaultName}},
		logr.Discard(),
	)
	if !assert.NoError(t, err) {
		return
	}

	updated := new(egressv1.EgressClusterInfo)
	err = reconciler.cli.Get(context.Background(), client.ObjectKey{Name: defaultName}, updated)
	if !assert.NoError(t, err) {
		return
	}

	_, hasNodePodCIDR := updated.Status.PodCIDR["worker-1"]
	assert.False(t, hasNodePodCIDR)
	assert.Equal(t, []string{"192.168.0.0/16"}, updated.Status.PodCIDR["default-ipv4-pool"].IPv4)
	assert.Equal(t, egressv1.CniTypeCalico, updated.Status.PodCidrMode)
}

func TestReconcileCalicoIPPoolFallsBackToK8sAfterLastPoolIsDeleted(t *testing.T) {
	info := &egressv1.EgressClusterInfo{
		ObjectMeta: metav1.ObjectMeta{Name: defaultName},
		Spec: egressv1.EgressClusterInfoSpec{
			AutoDetect: egressv1.AutoDetect{PodCidrMode: egressv1.CniTypeAuto},
		},
		Status: egressv1.EgressClusterInfoStatus{
			PodCidrMode: egressv1.CniTypeCalico,
			PodCIDR: map[string]egressv1.IPListPair{
				"default-ipv4-pool": {IPv4: []string{"192.168.0.0/16"}},
			},
		},
	}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "worker-1"},
		Spec:       corev1.NodeSpec{PodCIDRs: []string{"10.244.1.0/24"}},
	}
	builder := fake.NewClientBuilder().
		WithScheme(schema.GetScheme()).
		WithObjects(info, node).
		WithStatusSubresource(info)
	reconciler := &clusterInfo{cli: builder.Build()}

	err := reconciler.reconcileCalicoIPPool(
		context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: "default-ipv4-pool"}},
		logr.Discard(),
	)
	if !assert.NoError(t, err) {
		return
	}

	updated := new(egressv1.EgressClusterInfo)
	err = reconciler.cli.Get(context.Background(), client.ObjectKey{Name: defaultName}, updated)
	if !assert.NoError(t, err) {
		return
	}

	_, hasCalicoPool := updated.Status.PodCIDR["default-ipv4-pool"]
	assert.False(t, hasCalicoPool)
	assert.Equal(t, []string{"10.244.1.0/24"}, updated.Status.PodCIDR["worker-1"].IPv4)
	assert.Equal(t, egressv1.CniTypeK8s, updated.Status.PodCidrMode)
}

func TestReconcileInfoDisablesPodCIDRDetection(t *testing.T) {
	info := &egressv1.EgressClusterInfo{
		ObjectMeta: metav1.ObjectMeta{Name: defaultName},
		Spec: egressv1.EgressClusterInfoSpec{
			AutoDetect: egressv1.AutoDetect{PodCidrMode: egressv1.CniTypeEmpty},
		},
		Status: egressv1.EgressClusterInfoStatus{
			PodCidrMode: egressv1.CniTypeK8s,
			PodCIDR: map[string]egressv1.IPListPair{
				"worker-1": {IPv4: []string{"10.244.1.0/24"}},
			},
		},
	}
	builder := fake.NewClientBuilder().
		WithScheme(schema.GetScheme()).
		WithObjects(info).
		WithStatusSubresource(info)
	reconciler := &clusterInfo{cli: builder.Build()}

	err := reconciler.reconcileInfo(
		context.Background(),
		reconcile.Request{NamespacedName: types.NamespacedName{Name: defaultName}},
		logr.Discard(),
	)
	if !assert.NoError(t, err) {
		return
	}

	updated := new(egressv1.EgressClusterInfo)
	err = reconciler.cli.Get(context.Background(), client.ObjectKey{Name: defaultName}, updated)
	if !assert.NoError(t, err) {
		return
	}

	assert.Empty(t, updated.Status.PodCIDR)
	assert.Equal(t, egressv1.CniTypeEmpty, updated.Status.PodCidrMode)
}

func TestGetNodePodCIDRList(t *testing.T) {
	tests := []struct {
		name         string
		node         *corev1.Node
		expectedIPv4 []string
		expectedIPv6 []string
	}{
		{
			name: "dual stack pod CIDRs",
			node: &corev1.Node{Spec: corev1.NodeSpec{
				PodCIDRs: []string{"10.244.1.0/24", "fd00:10:244:1::/64"},
			}},
			expectedIPv4: []string{"10.244.1.0/24"},
			expectedIPv6: []string{"fd00:10:244:1::/64"},
		},
		{
			name:         "legacy singular pod CIDR",
			node:         &corev1.Node{Spec: corev1.NodeSpec{PodCIDR: "10.244.2.0/24"}},
			expectedIPv4: []string{"10.244.2.0/24"},
		},
		{
			name: "invalid CIDRs are ignored",
			node: &corev1.Node{Spec: corev1.NodeSpec{
				PodCIDRs: []string{"invalid", "10.244.3.0/24"},
			}},
			expectedIPv4: []string{"10.244.3.0/24"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipv4, ipv6 := getNodePodCIDRList(tt.node)
			assert.Equal(t, tt.expectedIPv4, ipv4)
			assert.Equal(t, tt.expectedIPv6, ipv6)
		})
	}
}

func TestNodePredicateDetectsPodCIDRChanges(t *testing.T) {
	oldNode := &corev1.Node{
		Spec: corev1.NodeSpec{PodCIDRs: []string{"10.244.1.0/24"}},
	}
	newNode := oldNode.DeepCopy()
	newNode.Spec.PodCIDR = "10.244.2.0/24"
	newNode.Spec.PodCIDRs = []string{"10.244.2.0/24"}

	predicate := nodePredicate{}
	assert.True(t, predicate.Update(event.UpdateEvent{ObjectOld: oldNode, ObjectNew: newNode}))
	assert.False(t, predicate.Update(event.UpdateEvent{ObjectOld: oldNode, ObjectNew: oldNode.DeepCopy()}))
}
