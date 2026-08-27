// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/spidernet-io/egressgateway/pkg/agent/vxlan"
	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

func TestInitTunnelPeerMapUsesTunnelNames(t *testing.T) {
	tunnels := []client.Object{
		&egressv1.EgressTunnel{
			ObjectMeta: metav1.ObjectMeta{Name: "node-a"},
			Status: egressv1.EgressTunnelStatus{
				Phase: egressv1.EgressTunnelReady,
				Tunnel: egressv1.Tunnel{
					IPv4: "192.200.0.1",
					MAC:  "66:00:00:00:00:01",
				},
			},
		},
		&egressv1.EgressTunnel{
			ObjectMeta: metav1.ObjectMeta{Name: "node-b"},
			Status: egressv1.EgressTunnelStatus{
				Phase: egressv1.EgressTunnelReady,
				Tunnel: egressv1.Tunnel{
					IPv4: "192.200.0.2",
					MAC:  "66:00:00:00:00:02",
				},
			},
		},
	}

	r := &vxlanReconciler{
		client: fake.NewClientBuilder().
			WithScheme(schema.GetScheme()).
			WithObjects(tunnels...).
			Build(),
		cfg: &config.Config{
			EnvConfig:  config.EnvConfig{NodeName: "node-a"},
			FileConfig: config.FileConfig{EnableIPv4: true},
		},
		peerMap: utils.NewSyncMap[string, vxlan.Peer](),
	}

	assert.NoError(t, r.initTunnelPeerMap())

	peerA, ok := r.peerMap.Load("node-a")
	if assert.True(t, ok) {
		assert.Equal(t, "192.200.0.1", peerA.IPv4.String())
		assert.Equal(t, "66:00:00:00:00:01", peerA.MAC.String())
	}

	peerB, ok := r.peerMap.Load("node-b")
	if assert.True(t, ok) {
		assert.Equal(t, "192.200.0.2", peerB.IPv4.String())
		assert.Equal(t, "66:00:00:00:00:02", peerB.MAC.String())
	}
}
