// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1"
	"github.com/vishvananda/netlink"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/spidernet-io/egressgateway/pkg/agent/route"
	"github.com/spidernet-io/egressgateway/pkg/agent/vxlan"
	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

type TestCaseVXLAN struct {
	initialObjects []client.Object
	reqs           []TestReqVXLAN
	config         *config.Config
}

type TestReqVXLAN struct {
	nn         types.NamespacedName
	expErr     bool
	expRequeue bool
}

func TestReconcilerEgressNode(t *testing.T) {
	cases := map[string]TestCaseVXLAN{
		//"caseAddEgressNode": caseAddEgressNode(),
	}

	getParent := vxlan.GetParentByDefaultRoute(vxlan.NetLink{
		RouteListFiltered: netlink.RouteListFiltered,
		LinkByIndex:       netlink.LinkByIndex,
		AddrList:          netlink.AddrList,
		LinkByName:        netlink.LinkByName,
	})

	for name, c := range cases {
		log := logger.NewStdoutLogger("error")

		t.Run(name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			builder.WithObjects(c.initialObjects...)
			ctx := context.Background()
			ruleRoute := route.NewRuleRoute(log)
			reconciler := vxlanReconciler{
				client:         builder.Build(),
				log:            log,
				cfg:            c.config,
				getParent:      getParent,
				ruleRoute:      ruleRoute,
				ruleRouteCache: utils.NewSyncMap[string, []net.IP](),
			}

			for _, req := range c.reqs {
				res, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: req.nn})
				if !req.expErr {
					assert.NoError(t, err)
				}
				assert.Equal(t, req.expRequeue, res.Requeue)
			}
		})
	}
}

func caseAddEgressNode() TestCaseVXLAN {

	_, ipn, err := net.ParseCIDR("192.168.100.1/24")
	if err != nil {
		panic(err)
	}

	cfg := &config.Config{
		EnvConfig: config.EnvConfig{
			NodeName: "workstation1",
		},
		FileConfig: config.FileConfig{
			EnableIPv4: true,
			EnableIPv6: false,

			TunnelIPv4Net: ipn,
			TunnelIPv6Net: nil,
		},
	}

	return TestCaseVXLAN{
		initialObjects: []client.Object{
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{Name: "workstation1"},
				Spec:       egressv1.EgressNodeSpec{},
				Status:     egressv1.EgressNodeStatus{},
			},
		},
		reqs: []TestReqVXLAN{
			{
				nn: types.NamespacedName{
					Namespace: "EgressNode/",
					Name:      "workstation1",
				},
				expErr:     false,
				expRequeue: false,
			},
		},
		config: cfg,
	}
}
