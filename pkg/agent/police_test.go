// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/ipset"
	testing2 "github.com/spidernet-io/egressgateway/pkg/ipset/testing"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
	"github.com/spidernet-io/egressgateway/pkg/logger"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/spidernet-io/egressgateway/pkg/utils"
)

type TestCaseEGP struct {
	initialObjects  []client.Object
	initialV4IPSets map[string][]string
	initialV6IPSets map[string][]string
	reqs            []TestEgressGatewayPolicyReq
	config          *config.Config
}

type TestEgressGatewayPolicyReq struct {
	nn         types.NamespacedName
	expErr     bool
	expRequeue bool
	expIPSets  map[string][]string
}

func TestReconcilerEgressGatewayPolicy(t *testing.T) {
	cases := map[string]TestCaseEGP{
		"caseAddEgressGatewayPolicy":    caseAddEgressGatewayPolicy(),
		"caseDelEgressGatewayPolicy":    caseDelEgressGatewayPolicy(),
		"caseUpdateEgressGatewayPolicy": caseUpdateEgressGatewayPolicy(),
		"caseAddPodUpdatePolicy":        caseAddPodUpdatePolicy(),
		"caseDelPodUpdatePolicy":        caseDelPodUpdatePolicy(),
	}
	for name, c := range cases {
		log := logger.NewStdoutLogger(os.Getenv("LOG_LEVEL"))
		t.Run(name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			builder.WithObjects(c.initialObjects...)
			ipsetClient := testing2.NewFake("")
			ipsetMap := utils.NewSyncMap[string, *ipset.IPSet]()
			reconciler := policeReconciler{
				client:   builder.Build(),
				log:      log,
				cfg:      c.config,
				ipsetMap: ipsetMap,
				ipset:    ipsetClient,
			}

			for name, ips := range c.initialV4IPSets {
				err := initIPSet(ipsetClient, ipsetMap, name, "IPv4", ips)
				assert.NoError(t, err)
			}
			for name, ips := range c.initialV6IPSets {
				err := initIPSet(ipsetClient, ipsetMap, name, "IPv6", ips)
				assert.NoError(t, err)
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
				sets, err := reconciler.ipset.ListSets()
				assert.NoError(t, err)
				for _, set := range sets {
					expSet, ok := req.expIPSets[set]
					if !ok {
						t.Fatalf("find not expected IPSet: %s", set)
					}
					entries, err := reconciler.ipset.ListEntries(set)
					assert.NoError(t, err)

					if !assert.ElementsMatch(t, entries, expSet) {
						fmt.Printf("%s got:\n%v\n", set, entries)
						fmt.Printf("%s exp:\n%v\n", set, expSet)
					}
					delete(req.expIPSets, set)
				}
				if len(req.expIPSets) > 0 {
					list := make([]string, len(req.expIPSets))
					for key := range req.expIPSets {
						list = append(list, key)
					}
					t.Fatalf("exp ip set list not found in got list: %v", list)
				}
			}
		})
	}
}

func initIPSet(cli ipset.Interface, setMap *utils.SyncMap[string, *ipset.IPSet], name, family string, ips []string) error {
	set := &ipset.IPSet{
		Name:       name,
		SetType:    ipset.HashNet,
		HashFamily: family,
	}
	err := cli.CreateSet(set, false)
	if err != nil {
		return err
	}
	setMap.Store(name, set)
	for _, ip := range ips {
		err = cli.AddEntry(ip, set, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func caseAddEgressGatewayPolicy() TestCaseEGP {
	return TestCaseEGP{
		initialObjects: []client.Object{
			&egressv1.EgressGatewayPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy1",
				},
				Spec: egressv1.EgressGatewayPolicySpec{
					AppliedTo: egressv1.AppliedTo{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "2048",
							},
						},
					},
				},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod1",
					Labels: map[string]string{
						"app": "2048",
					},
				},
				Spec: corev1.PodSpec{},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{IP: "10.245.0.2"},
						{IP: "10.245.0.3"},
						{IP: "10.245.0.4"},
						{IP: "10.245.0.5"},
						{IP: "2000::2"},
						{IP: "2000::3"},
						{IP: "2000::4"},
						{IP: "2000::5"},
					},
				},
			},
		},
		reqs: []TestEgressGatewayPolicyReq{
			{
				nn: types.NamespacedName{
					Namespace: "EgressGatewayPolicy/",
					Name:      "policy1",
				},
				expErr:     false,
				expRequeue: false,
				expIPSets: map[string][]string{
					formatIPSetName("egress-src-v4-", "policy1"): {
						"10.245.0.2", "10.245.0.3", "10.245.0.4", "10.245.0.5",
					},
					formatIPSetName("egress-dst-v4-", "policy1"): {},
				},
			},
		},
		config: &config.Config{
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
	}
}

func caseDelEgressGatewayPolicy() TestCaseEGP {
	return TestCaseEGP{
		initialObjects: []client.Object{
			&egressv1.EgressGatewayPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "policy1",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: egressv1.EgressGatewayPolicySpec{
					AppliedTo: egressv1.AppliedTo{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "2048",
							},
						},
					},
				},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod1",
					Labels: map[string]string{
						"app": "2048",
					},
				},
				Spec: corev1.PodSpec{},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{IP: "10.245.0.2"},
						{IP: "10.245.0.3"},
						{IP: "10.245.0.4"},
						{IP: "10.245.0.5"},
						{IP: "2000::2"},
						{IP: "2000::3"},
						{IP: "2000::4"},
						{IP: "2000::5"},
					},
				},
			},
		},
		initialV4IPSets: map[string][]string{
			formatIPSetName("egress-src-v4-", "policy1"): {
				"10.245.0.2", "10.245.0.3", "10.245.0.4", "10.245.0.5",
			},
			formatIPSetName("egress-dst-v4-", "policy1"): {},
		},
		reqs: []TestEgressGatewayPolicyReq{
			{
				nn: types.NamespacedName{
					Namespace: "EgressGatewayPolicy/",
					Name:      "policy1",
				},
				expErr:     false,
				expRequeue: false,
				expIPSets:  map[string][]string{},
			},
		},
		config: &config.Config{
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
	}
}

func caseUpdateEgressGatewayPolicy() TestCaseEGP {
	return TestCaseEGP{
		initialObjects: []client.Object{
			&egressv1.EgressGatewayPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy1",
				},
				Spec: egressv1.EgressGatewayPolicySpec{
					AppliedTo: egressv1.AppliedTo{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "app2",
							},
						},
					},
				},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod1",
					Labels: map[string]string{
						"app": "app1",
					},
				},
				Spec: corev1.PodSpec{},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{IP: "10.245.0.2"},
						{IP: "10.245.0.3"},
						{IP: "10.245.0.4"},
						{IP: "10.245.0.5"},
						{IP: "2000::2"},
						{IP: "2000::3"},
						{IP: "2000::4"},
						{IP: "2000::5"},
					},
				},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod2",
					Labels: map[string]string{
						"app": "app2",
					},
				},
				Spec: corev1.PodSpec{},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{IP: "10.245.0.6"},
						{IP: "10.245.0.7"},
						{IP: "10.245.0.8"},
						{IP: "10.245.0.9"},
						{IP: "2000::6"},
						{IP: "2000::7"},
						{IP: "2000::8"},
						{IP: "2000::9"},
					},
				},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod3",
					Labels: map[string]string{
						"app": "app2",
					},
				},
				Spec: corev1.PodSpec{},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{IP: "10.245.0.10"},
						{IP: "10.245.0.11"},
					},
				},
			},
		},
		initialV4IPSets: map[string][]string{
			formatIPSetName("egress-src-v4-", "policy1"): {
				"10.245.0.2", "10.245.0.3", "10.245.0.4", "10.245.0.5",
			},
			formatIPSetName("egress-dst-v4-", "policy1"): {},
		},
		initialV6IPSets: map[string][]string{},
		reqs: []TestEgressGatewayPolicyReq{
			{
				nn: types.NamespacedName{
					Namespace: "EgressGatewayPolicy/",
					Name:      "policy1",
				},
				expErr:     false,
				expRequeue: false,
				expIPSets: map[string][]string{
					formatIPSetName("egress-src-v4-", "policy1"): {
						"10.245.0.6",
						"10.245.0.7",
						"10.245.0.8",
						"10.245.0.9",
						"10.245.0.10",
						"10.245.0.11",
					},
					formatIPSetName("egress-dst-v4-", "policy1"): {},
				},
			},
		},
		config: &config.Config{
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
	}
}

func caseAddPodUpdatePolicy() TestCaseEGP {
	return TestCaseEGP{
		initialObjects: []client.Object{
			&egressv1.EgressGatewayPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy1",
				},
				Spec: egressv1.EgressGatewayPolicySpec{
					AppliedTo: egressv1.AppliedTo{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "app1",
							},
						},
					},
				},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "pod1",
					Labels: map[string]string{
						"app": "app1",
					},
				},
				Spec: corev1.PodSpec{},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{IP: "10.245.0.2"},
						{IP: "10.245.0.3"},
					},
				},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "pod2",
					Labels: map[string]string{
						"app": "app1",
					},
				},
				Spec: corev1.PodSpec{},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{IP: "10.245.0.4"},
						{IP: "10.245.0.5"},
					},
				},
			},
		},
		initialV4IPSets: map[string][]string{
			formatIPSetName("egress-src-v4-", "policy1"): {
				"10.245.0.2",
				"10.245.0.3",
			},
			formatIPSetName("egress-dst-v4-", "policy1"): {},
		},
		initialV6IPSets: map[string][]string{},

		reqs: []TestEgressGatewayPolicyReq{
			{
				nn: types.NamespacedName{
					Namespace: "Pod/default",
					Name:      "pod2",
				},
				expErr:     false,
				expRequeue: false,
				expIPSets: map[string][]string{
					formatIPSetName("egress-src-v4-", "policy1"): {
						"10.245.0.2",
						"10.245.0.3",
						"10.245.0.4",
						"10.245.0.5",
					},
					formatIPSetName("egress-dst-v4-", "policy1"): {},
				},
			},
		},
		config: &config.Config{
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
	}
}

func caseDelPodUpdatePolicy() TestCaseEGP {
	return TestCaseEGP{
		initialObjects: []client.Object{
			&egressv1.EgressGatewayPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy1",
				},
				Spec: egressv1.EgressGatewayPolicySpec{
					AppliedTo: egressv1.AppliedTo{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "app1",
							},
						},
					},
				},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod1",
					Labels: map[string]string{
						"app": "app1",
					},
				},
				Spec: corev1.PodSpec{},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{IP: "10.245.0.2"},
						{IP: "10.245.0.3"},
					},
				},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod2",
					Labels: map[string]string{
						"app": "app1",
					},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
				Spec: corev1.PodSpec{},
				Status: corev1.PodStatus{
					PodIPs: []corev1.PodIP{
						{IP: "10.245.0.4"},
						{IP: "10.245.0.5"},
					},
				},
			},
		},
		initialV4IPSets: map[string][]string{
			formatIPSetName("egress-src-v4-", "policy1"): {
				"10.245.0.2",
				"10.245.0.3",
			},
			formatIPSetName("egress-dst-v4-", "policy1"): {},
		},
		initialV6IPSets: map[string][]string{},

		reqs: []TestEgressGatewayPolicyReq{
			{
				nn: types.NamespacedName{
					Namespace: "Pod/default",
					Name:      "pod2",
				},
				expErr:     false,
				expRequeue: false,
				expIPSets: map[string][]string{
					formatIPSetName("egress-src-v4-", "policy1"): {
						"10.245.0.2",
						"10.245.0.3",
					},
					formatIPSetName("egress-dst-v4-", "policy1"): {},
				},
			},
		},
		config: &config.Config{
			FileConfig: config.FileConfig{
				EnableIPv4: true,
				EnableIPv6: false,
			},
		},
	}
}
