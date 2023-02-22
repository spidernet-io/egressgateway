// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

type TestCaseEGN struct {
	initialObjects []client.Object
	reqs           []TestEGNReq
	config         *config.Config
}

type TestEGNReq struct {
	nn         types.NamespacedName
	expErr     bool
	expRequeue bool
	expEGN     []egressv1.EgressGateway
}

func TestReconcilerEgressGateway(t *testing.T) {
	cases := map[string]TestCaseEGN{
		"caseNodeReadyButVxlanNotReady":            caseNodeReadyButVxlanNotReady(),
		"caseNodeReadVxlanReady":                   caseNodeReadVxlanReady(),
		"caseNodeReadyVxlanReadyDualStack":         caseNodeReadyVxlanReadyDualStack(),
		"caseNodeReadyVxlanReadyDualStackNotReady": caseNodeReadyVxlanReadyDualStackNotReady(),
		"caseMoreNodeBeSelected":                   caseMoreNodeBeSelected(),
		"caseMoreNodeBeSelectedActiveActive":       caseMoreNodeBeSelectedActiveActive(),
		"caseChangeEgressGatewayLabel":             caseChangeEgressGatewayLabel(),
		"caseDeleteEgressNode":                     caseDeleteEgressNode(),
		"caseNodeReadyToNotReady":                  caseNodeReadyToNotReady(),
		"caseNodeNotReadyToReady":                  caseNodeNotReadyToReady(),
	}
	for name, c := range cases {
		log := logger.NewStdoutLogger(os.Getenv("LOG_LEVEL"))

		t.Run(name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			builder.WithObjects(c.initialObjects...)
			reconciler := egnReconciler{
				client: builder.Build(),
				log:    log,
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

				for _, expEgn := range req.expEGN {
					gotEgn := new(egressv1.EgressGateway)
					err := reconciler.client.Get(context.Background(), types.NamespacedName{
						Name: expEgn.Name,
					}, gotEgn)
					assert.NoError(t, err)

					hasDiff := difference(gotEgn.Status.NodeList, expEgn.Status.NodeList,
						func(a, b egressv1.SelectedEgressNode) bool {
							if a.Name != b.Name {
								return true
							}
							if a.Active != b.Active {
								return true
							}
							if a.Ready != b.Ready {
								return true
							}
							return false
						})
					if hasDiff {
						msg := "can get exp egress gateway node:\ngot:\n"
						for _, item := range gotEgn.Status.NodeList {
							msg = msg + fmt.Sprintf("- name:%v ready:%v active:%v\n", item.Name, item.Ready, item.Active)
						}
						msg += "exp:\n"
						for _, item := range expEgn.Status.NodeList {
							msg = msg + fmt.Sprintf("- name:%v ready:%v active:%v\n", item.Name, item.Ready, item.Active)
						}
						t.Fatal(msg)
					}
				}
			}
		})
	}
}

func caseNodeReadyButVxlanNotReady() TestCaseEGN {
	return TestCaseEGN{
		config: &config.Config{
			EnvConfig: config.EnvConfig{},
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
		initialObjects: []client.Object{
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&egressv1.EgressGateway{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "egress1",
				},
				Spec: egressv1.EgressGatewaySpec{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"egress": "true",
						},
					},
				},
				Status: egressv1.EgressGatewayStatus{},
			},
		},
		reqs: []TestEGNReq{
			{
				nn: types.NamespacedName{
					Namespace: "EgressGateway/",
					Name:      "egress1",
				},
				expErr:     false,
				expRequeue: false,
				expEGN: []egressv1.EgressGateway{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "egress1",
						},

						Spec: egressv1.EgressGatewaySpec{},
						Status: egressv1.EgressGatewayStatus{
							NodeList: []egressv1.SelectedEgressNode{
								{
									Name:            "node1",
									Ready:           false,
									Active:          false,
									InterfaceStatus: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

func caseNodeReadVxlanReady() TestCaseEGN {
	return TestCaseEGN{
		config: &config.Config{
			EnvConfig: config.EnvConfig{},
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
		initialObjects: []client.Object{
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.1",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1d",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressGateway{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "egress1",
				},
				Spec: egressv1.EgressGatewaySpec{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"egress": "true",
						},
					},
				},
				Status: egressv1.EgressGatewayStatus{},
			},
		},
		reqs: []TestEGNReq{
			{
				nn: types.NamespacedName{
					Namespace: "EgressGateway/",
					Name:      "egress1",
				},
				expErr:     false,
				expRequeue: false,
				expEGN: []egressv1.EgressGateway{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "egress1",
						},

						Spec: egressv1.EgressGatewaySpec{},
						Status: egressv1.EgressGatewayStatus{
							NodeList: []egressv1.SelectedEgressNode{
								{
									Name:            "node1",
									Ready:           true,
									Active:          true,
									InterfaceStatus: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

func caseNodeReadyVxlanReadyDualStack() TestCaseEGN {
	return TestCaseEGN{
		config: &config.Config{
			EnvConfig: config.EnvConfig{},
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: true,
			},
		},
		initialObjects: []client.Object{
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.1",
					VxlanIPv6:             "1010:0000:0000:0000:0000:0000:0000:00001",
					TunnelMac:             "00:50:56:b4:02:1d",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "1020:0000:0000:0000:0000:0000:0000:0001",
				},
			},
			&egressv1.EgressGateway{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "egress1",
				},
				Spec: egressv1.EgressGatewaySpec{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"egress": "true",
						},
					},
				},
				Status: egressv1.EgressGatewayStatus{},
			},
		},
		reqs: []TestEGNReq{
			{
				nn: types.NamespacedName{
					Namespace: "EgressGateway/",
					Name:      "egress1",
				},
				expErr:     false,
				expRequeue: false,
				expEGN: []egressv1.EgressGateway{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "egress1",
						},

						Spec: egressv1.EgressGatewaySpec{},
						Status: egressv1.EgressGatewayStatus{
							NodeList: []egressv1.SelectedEgressNode{
								{
									Name:            "node1",
									Ready:           true,
									Active:          true,
									InterfaceStatus: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

func caseNodeReadyVxlanReadyDualStackNotReady() TestCaseEGN {
	return TestCaseEGN{
		config: &config.Config{
			EnvConfig: config.EnvConfig{},
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: true,
			},
		},
		initialObjects: []client.Object{
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.1",
					VxlanIPv6:             "1010:0000:0000:0000:0000:0000:0000:00001",
					TunnelMac:             "00:50:56:b4:02:1d",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressGateway{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "egress1",
				},
				Spec: egressv1.EgressGatewaySpec{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"egress": "true",
						},
					},
				},
				Status: egressv1.EgressGatewayStatus{},
			},
		},
		reqs: []TestEGNReq{
			{
				nn: types.NamespacedName{
					Namespace: "EgressGateway/",
					Name:      "egress1",
				},
				expErr:     false,
				expRequeue: false,
				expEGN: []egressv1.EgressGateway{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "egress1",
						},

						Spec: egressv1.EgressGatewaySpec{},
						Status: egressv1.EgressGatewayStatus{
							NodeList: []egressv1.SelectedEgressNode{
								{
									Name:            "node1",
									Ready:           false,
									Active:          false,
									InterfaceStatus: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

func caseMoreNodeBeSelected() TestCaseEGN {
	return TestCaseEGN{
		config: &config.Config{
			EnvConfig: config.EnvConfig{},
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
		initialObjects: []client.Object{
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.1",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1d",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.1",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1d",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "",
					VxlanIPv6:             "",
					TunnelMac:             "",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressGateway{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "egress1",
				},
				Spec: egressv1.EgressGatewaySpec{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"egress": "true",
						},
					},
				},
				Status: egressv1.EgressGatewayStatus{},
			},
		},
		reqs: []TestEGNReq{
			{
				nn: types.NamespacedName{
					Namespace: "EgressGateway/",
					Name:      "egress1",
				},
				expErr:     false,
				expRequeue: false,
				expEGN: []egressv1.EgressGateway{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "egress1",
						},
						Spec: egressv1.EgressGatewaySpec{},
						Status: egressv1.EgressGatewayStatus{
							NodeList: []egressv1.SelectedEgressNode{
								{
									Name:            "node1",
									Ready:           true,
									Active:          true,
									InterfaceStatus: nil,
								},
								{
									Name:            "node2",
									Ready:           true,
									Active:          false,
									InterfaceStatus: nil,
								},
								{
									Name:            "node3",
									Ready:           false,
									Active:          false,
									InterfaceStatus: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

func caseMoreNodeBeSelectedActiveActive() TestCaseEGN {
	return TestCaseEGN{
		config: &config.Config{
			EnvConfig: config.EnvConfig{},
			FileConfig: config.FileConfig{
				ForwardMethod: config.ForwardMethodActiveActive,
				EnableIPv4:    true,
				EnableIPv6:    false,
			},
		},
		initialObjects: []client.Object{
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.1",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1d",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.1",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1d",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "",
					VxlanIPv6:             "",
					TunnelMac:             "",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressGateway{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "egress1",
				},
				Spec: egressv1.EgressGatewaySpec{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"egress": "true",
						},
					},
				},
				Status: egressv1.EgressGatewayStatus{},
			},
		},
		reqs: []TestEGNReq{
			{
				nn: types.NamespacedName{
					Namespace: "EgressGateway/",
					Name:      "egress1",
				},
				expErr:     false,
				expRequeue: false,
				expEGN: []egressv1.EgressGateway{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "egress1",
						},
						Spec: egressv1.EgressGatewaySpec{},
						Status: egressv1.EgressGatewayStatus{
							NodeList: []egressv1.SelectedEgressNode{
								{
									Name:            "node1",
									Ready:           true,
									Active:          true,
									InterfaceStatus: nil,
								},
								{
									Name:            "node2",
									Ready:           true,
									Active:          true,
									InterfaceStatus: nil,
								},
								{
									Name:            "node3",
									Ready:           false,
									Active:          false,
									InterfaceStatus: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

func caseChangeEgressGatewayLabel() TestCaseEGN {
	return TestCaseEGN{
		config: &config.Config{
			EnvConfig: config.EnvConfig{},
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
		initialObjects: []client.Object{
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.1",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1a",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.2",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1b",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.2",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.3",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1c",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.3",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressGateway{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "egress1",
				},
				Spec: egressv1.EgressGatewaySpec{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"egress": "true",
						},
					},
				},
				Status: egressv1.EgressGatewayStatus{
					NodeList: []egressv1.SelectedEgressNode{
						{
							Name:            "node1",
							Ready:           true,
							Active:          true,
							InterfaceStatus: nil,
						},
						{
							Name:            "node2",
							Ready:           true,
							Active:          false,
							InterfaceStatus: nil,
						},
						{
							Name:            "node3",
							Ready:           false,
							Active:          false,
							InterfaceStatus: nil,
						},
					},
				},
			},
		},
		reqs: []TestEGNReq{
			{
				nn: types.NamespacedName{
					Namespace: "EgressGateway/",
					Name:      "egress1",
				},
				expErr:     false,
				expRequeue: false,
				expEGN: []egressv1.EgressGateway{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "egress1",
						},
						Spec: egressv1.EgressGatewaySpec{},
						Status: egressv1.EgressGatewayStatus{
							NodeList: []egressv1.SelectedEgressNode{
								{
									Name:            "node2",
									Ready:           true,
									Active:          true,
									InterfaceStatus: nil,
								},
								{
									Name:            "node3",
									Ready:           true,
									Active:          false,
									InterfaceStatus: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

func caseDeleteEgressNode() TestCaseEGN {
	return TestCaseEGN{
		config: &config.Config{
			EnvConfig: config.EnvConfig{},
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
		initialObjects: []client.Object{
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "node1",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.1",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1a",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.2",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1b",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.2",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.3",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1c",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.3",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressGateway{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "egress1",
				},
				Spec: egressv1.EgressGatewaySpec{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"egress": "true",
						},
					},
				},
				Status: egressv1.EgressGatewayStatus{
					NodeList: []egressv1.SelectedEgressNode{
						{
							Name:            "node1",
							Ready:           true,
							Active:          true,
							InterfaceStatus: nil,
						},
						{
							Name:            "node2",
							Ready:           true,
							Active:          false,
							InterfaceStatus: nil,
						},
						{
							Name:            "node3",
							Ready:           true,
							Active:          false,
							InterfaceStatus: nil,
						},
					},
				},
			},
		},
		reqs: []TestEGNReq{
			{
				nn: types.NamespacedName{
					Namespace: "EgressNode/",
					Name:      "node1",
				},
				expErr:     false,
				expRequeue: false,
				expEGN: []egressv1.EgressGateway{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "egress1",
						},
						Spec: egressv1.EgressGatewaySpec{},
						Status: egressv1.EgressGatewayStatus{
							NodeList: []egressv1.SelectedEgressNode{
								{
									Name:            "node2",
									Ready:           true,
									Active:          true,
									InterfaceStatus: nil,
								},
								{
									Name:            "node3",
									Ready:           true,
									Active:          false,
									InterfaceStatus: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

func caseNodeReadyToNotReady() TestCaseEGN {
	return TestCaseEGN{
		config: &config.Config{
			EnvConfig: config.EnvConfig{},
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
		initialObjects: []client.Object{
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec: corev1.NodeSpec{},
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type: corev1.NodeReady,
						},
					},
				},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "node1",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.1",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1a",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.2",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1b",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.2",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.3",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1c",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.3",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressGateway{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "egress1",
				},
				Spec: egressv1.EgressGatewaySpec{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"egress": "true",
						},
					},
				},
				Status: egressv1.EgressGatewayStatus{
					NodeList: []egressv1.SelectedEgressNode{
						{
							Name:            "node1",
							Ready:           true,
							Active:          true,
							InterfaceStatus: nil,
						},
						{
							Name:            "node2",
							Ready:           true,
							Active:          false,
							InterfaceStatus: nil,
						},
						{
							Name:            "node3",
							Ready:           true,
							Active:          false,
							InterfaceStatus: nil,
						},
					},
				},
			},
		},
		reqs: []TestEGNReq{
			{
				nn: types.NamespacedName{
					Namespace: "Node/",
					Name:      "node1",
				},
				expErr:     false,
				expRequeue: false,
				expEGN: []egressv1.EgressGateway{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "egress1",
						},
						Spec: egressv1.EgressGatewaySpec{},
						Status: egressv1.EgressGatewayStatus{
							NodeList: []egressv1.SelectedEgressNode{
								{
									Name:            "node1",
									Ready:           false,
									Active:          false,
									InterfaceStatus: nil,
								},
								{
									Name:            "node2",
									Ready:           true,
									Active:          true,
									InterfaceStatus: nil,
								},
								{
									Name:            "node3",
									Ready:           true,
									Active:          false,
									InterfaceStatus: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

func caseNodeNotReadyToReady() TestCaseEGN {
	return TestCaseEGN{
		config: &config.Config{
			EnvConfig: config.EnvConfig{},
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
		initialObjects: []client.Object{
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&corev1.Node{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
					Labels: map[string]string{
						"egress": "true",
					},
				},
				Spec:   corev1.NodeSpec{},
				Status: corev1.NodeStatus{},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "node1",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.1",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1a",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.1",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node2",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.2",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1b",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.2",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node3",
				},
				Spec: egressv1.EgressNodeSpec{},
				Status: egressv1.EgressNodeStatus{
					VxlanIPv4:             "10.6.0.3",
					VxlanIPv6:             "",
					TunnelMac:             "00:50:56:b4:02:1c",
					Phase:                 "Succeeded",
					PhysicalInterface:     "eth0",
					PhysicalInterfaceIPv4: "172.16.0.3",
					PhysicalInterfaceIPv6: "",
				},
			},
			&egressv1.EgressGateway{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: "egress1",
				},
				Spec: egressv1.EgressGatewaySpec{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"egress": "true",
						},
					},
				},
				Status: egressv1.EgressGatewayStatus{
					NodeList: []egressv1.SelectedEgressNode{
						{
							Name:            "node1",
							Ready:           false,
							Active:          false,
							InterfaceStatus: nil,
						},
						{
							Name:            "node2",
							Ready:           true,
							Active:          true,
							InterfaceStatus: nil,
						},
						{
							Name:            "node3",
							Ready:           true,
							Active:          false,
							InterfaceStatus: nil,
						},
					},
				},
			},
		},
		reqs: []TestEGNReq{
			{
				nn: types.NamespacedName{
					Namespace: "Node/",
					Name:      "node1",
				},
				expErr:     false,
				expRequeue: false,
				expEGN: []egressv1.EgressGateway{
					{
						TypeMeta: metav1.TypeMeta{},
						ObjectMeta: metav1.ObjectMeta{
							Name: "egress1",
						},
						Spec: egressv1.EgressGatewaySpec{},
						Status: egressv1.EgressGatewayStatus{
							NodeList: []egressv1.SelectedEgressNode{
								{
									Name:            "node1",
									Ready:           true,
									Active:          false,
									InterfaceStatus: nil,
								},
								{
									Name:            "node2",
									Ready:           true,
									Active:          true,
									InterfaceStatus: nil,
								},
								{
									Name:            "node3",
									Ready:           true,
									Active:          false,
									InterfaceStatus: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestNewEgressGatewayController(t *testing.T) {
	type Case struct {
		mgr    manager.Manager
		cfg    *config.Config
		log    *zap.Logger
		expErr bool
	}

	cases := map[string]Case{
		"cfg is nil": {
			nil,
			nil,
			nil,
			true,
		},
		"log is nil": {
			nil,
			&config.Config{},
			nil,
			true,
		},
		"normal": {
			nil,
			&config.Config{},
			nil,
			true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := newEgressGatewayController(c.mgr, c.log, c.cfg)
			if !c.expErr {
				assert.NoError(t, err)
			}
		})
	}
}
