// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"reflect"
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
	"github.com/spidernet-io/egressgateway/pkg/iptables"
	"github.com/spidernet-io/egressgateway/pkg/iptables/testutils"
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
	expMangle  []map[string][]string
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
		log := logger.NewStdoutLogger("error")
		t.Run(name, func(t *testing.T) {
			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			builder.WithObjects(c.initialObjects...)
			ipsetCli := testing2.NewFake("")
			ipsetMap := utils.NewSyncMap[string, *ipset.IPSet]()

			mangles := make([]*iptables.Table, 0)

			opt := iptables.Options{
				XTablesLock:              &mockMutex{},
				BackendMode:              c.config.FileConfig.IPTables.BackendMode,
				InsertMode:               "insert",
				RefreshInterval:          time.Second * time.Duration(c.config.FileConfig.IPTables.RefreshIntervalSecond),
				InitialPostWriteInterval: time.Second * time.Duration(c.config.FileConfig.IPTables.InitialPostWriteIntervalSecond),
				RestoreSupportsLock:      false,
				LockTimeout:              time.Second * time.Duration(c.config.FileConfig.IPTables.LockTimeoutSecond),
				LockProbeInterval:        time.Millisecond * time.Duration(c.config.FileConfig.IPTables.LockProbeIntervalMillis),
				LookPathOverride:         testutils.LookPathAll,
			}
			manglesStore := make([]*testutils.MockDataplane, 0)

			if c.config.FileConfig.EnableIPv4 {
				tmpOpt := opt
				dataplane := testutils.NewMockDataplane("mangle", map[string][]string{
					"PREROUTING":  {"-m comment --comment \"cali:6gwbT8clXdHdC1b1\" -j cali-PREROUTING"},
					"POSTROUTING": {"-m comment --comment \"cali:O3lYWMrLQYEMJtB5\" -j cali-POSTROUTING"},
					"FORWARD":     {},
				}, "legacy")
				tmpOpt.NewCmdOverride = dataplane.NewCmd
				tmpOpt.SleepOverride = dataplane.Sleep
				tmpOpt.NowOverride = dataplane.Now
				table, err := iptables.NewTable("mangle", 4, "egw-", tmpOpt, log)
				assert.NoError(t, err)
				mangles = append(mangles, table)
				manglesStore = append(manglesStore, dataplane)
			}
			if c.config.FileConfig.EnableIPv6 {
				tmpOpt := opt
				dataplane := testutils.NewMockDataplane("mangle", map[string][]string{
					"PREROUTING":  {"-m comment --comment \"cali:6gwbT8clXdHdC1b1\" -j cali-PREROUTING"},
					"POSTROUTING": {"-m comment --comment \"cali:O3lYWMrLQYEMJtB5\" -j cali-POSTROUTING"},
				}, "legacy")
				tmpOpt.NewCmdOverride = dataplane.NewCmd
				tmpOpt.SleepOverride = dataplane.Sleep
				tmpOpt.NowOverride = dataplane.Now
				table, err := iptables.NewTable("mangle", 6, "egw-", tmpOpt, log)
				assert.NoError(t, err)
				mangles = append(mangles, table)
			}

			reconciler := policeReconciler{
				client:       builder.Build(),
				log:          log,
				cfg:          c.config,
				ipsetMap:     ipsetMap,
				ipset:        ipsetCli,
				mangleTables: mangles,
				ruleV4Map:    utils.NewSyncMap[string, iptables.Rule](),
				ruleV6Map:    utils.NewSyncMap[string, iptables.Rule](),
			}

			for name, ips := range c.initialV4IPSets {
				err := initIPSet(ipsetCli, ipsetMap, name, "IPv4", ips)
				assert.NoError(t, err)
			}
			for name, ips := range c.initialV6IPSets {
				err := initIPSet(ipsetCli, ipsetMap, name, "IPv6", ips)
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
				// do check
				for i := range manglesStore {
					for name, gotRules := range manglesStore[i].Chains {
						if expRules, ok := req.expMangle[i][name]; ok {
							if !reflect.DeepEqual(gotRules, expRules) {
								fmt.Println("chain name:", name)
								fmt.Printf("gotRules:\n%v\n", gotRules)
								fmt.Printf("expRules:\n%v\n", expRules)
								t.Fatal("gotRules != expRules")
							}
						}
					}
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
			&egressv1.EgressGateway{
				ObjectMeta: metav1.ObjectMeta{Name: "gateway"},
				Spec:       egressv1.EgressGatewaySpec{},
				Status: egressv1.EgressGatewayStatus{NodeList: []egressv1.SelectedEgressNode{{
					Name:   "node1",
					Active: true,
				}}},
			},
			&egressv1.EgressGatewayPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: "policy1"},
				Spec: egressv1.EgressGatewayPolicySpec{
					AppliedTo: egressv1.AppliedTo{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "2048"},
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
				expMangle: []map[string][]string{
					{
						"EGRESSGATEWAY-MARK-REQUEST": []string{"" +
							"-m comment --comment \"egw-Mio2l_qQ2kKhiz2d\" -m set --match-set egress-src-v4-c0eb42ab804da452e src -m set --match-set egress-dst-v4-c0eb42ab804da452e dst --jump MARK --set-mark 0x11000000/0xffffffff",
						},
						"PREROUTING": []string{
							"-m comment --comment \"egw-HAe35Kaffr8R0mLj\" --jump EGRESSGATEWAY-MARK-REQUEST",
							"-m comment --comment \"cali:6gwbT8clXdHdC1b1\" -j cali-PREROUTING",
						},
						"POSTROUTING": []string{
							"-m comment --comment \"egw-1ecaPSxhNEc4Ylv_\" -m mark --mark 0x11000000/0xffffffff --jump ACCEPT",
							"-m comment --comment \"cali:O3lYWMrLQYEMJtB5\" -j cali-POSTROUTING",
						},
						"FORWARD": []string{
							"-m comment --comment \"egw-ogGr4WMsrCe4gJFJ\" -m mark --mark 0x11000000/0xffffffff --jump MARK --set-mark 0x12000000/0x12000000",
						},
					},
				},
			},
		},
		config: &config.Config{
			EnvConfig: config.EnvConfig{
				NodeName: "",
			},
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
			&egressv1.EgressGateway{
				ObjectMeta: metav1.ObjectMeta{Name: "gateway"},
				Spec:       egressv1.EgressGatewaySpec{},
				Status: egressv1.EgressGatewayStatus{NodeList: []egressv1.SelectedEgressNode{{
					Name:   "node1",
					Active: true,
				}}},
			},
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
				expMangle: []map[string][]string{
					{
						"PREROUTING": []string{
							"-m comment --comment \"egw-HAe35Kaffr8R0mLj\" --jump EGRESSGATEWAY-MARK-REQUEST",
							"-m comment --comment \"cali:6gwbT8clXdHdC1b1\" -j cali-PREROUTING",
						},
						"POSTROUTING": []string{
							"-m comment --comment \"egw-1ecaPSxhNEc4Ylv_\" -m mark --mark 0x11000000/0xffffffff --jump ACCEPT",
							"-m comment --comment \"cali:O3lYWMrLQYEMJtB5\" -j cali-POSTROUTING",
						},
						"FORWARD": []string{
							"-m comment --comment \"egw-ogGr4WMsrCe4gJFJ\" -m mark --mark 0x11000000/0xffffffff --jump MARK --set-mark 0x12000000/0x12000000",
						},
					},
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

func caseUpdateEgressGatewayPolicy() TestCaseEGP {
	return TestCaseEGP{
		initialObjects: []client.Object{
			&egressv1.EgressGateway{
				ObjectMeta: metav1.ObjectMeta{Name: "gateway"},
				Spec:       egressv1.EgressGatewaySpec{},
				Status: egressv1.EgressGatewayStatus{NodeList: []egressv1.SelectedEgressNode{{
					Name:   "node1",
					Active: true,
				}}},
			},
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
				expMangle: []map[string][]string{
					{
						"EGRESSGATEWAY-MARK-REQUEST": []string{"" +
							"-m comment --comment \"egw-Mio2l_qQ2kKhiz2d\" -m set --match-set egress-src-v4-c0eb42ab804da452e src -m set --match-set egress-dst-v4-c0eb42ab804da452e dst --jump MARK --set-mark 0x11000000/0xffffffff",
						},
						"PREROUTING": []string{
							"-m comment --comment \"egw-HAe35Kaffr8R0mLj\" --jump EGRESSGATEWAY-MARK-REQUEST",
							"-m comment --comment \"cali:6gwbT8clXdHdC1b1\" -j cali-PREROUTING",
						},
						"POSTROUTING": []string{
							"-m comment --comment \"egw-1ecaPSxhNEc4Ylv_\" -m mark --mark 0x11000000/0xffffffff --jump ACCEPT",
							"-m comment --comment \"cali:O3lYWMrLQYEMJtB5\" -j cali-POSTROUTING",
						},
						"FORWARD": []string{
							"-m comment --comment \"egw-ogGr4WMsrCe4gJFJ\" -m mark --mark 0x11000000/0xffffffff --jump MARK --set-mark 0x12000000/0x12000000",
						},
					},
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
			&egressv1.EgressGateway{
				ObjectMeta: metav1.ObjectMeta{Name: "gateway"},
				Spec:       egressv1.EgressGatewaySpec{},
				Status: egressv1.EgressGatewayStatus{NodeList: []egressv1.SelectedEgressNode{{
					Name:   "node1",
					Active: true,
				}}},
			},
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
				expMangle: []map[string][]string{
					{
						"EGRESSGATEWAY-MARK-REQUEST": []string{"" +
							"-m comment --comment \"egw-Mio2l_qQ2kKhiz2d\" -m set --match-set egress-src-v4-c0eb42ab804da452e src -m set --match-set egress-dst-v4-c0eb42ab804da452e dst --jump MARK --set-mark 0x11000000/0xffffffff",
						},
						"PREROUTING": []string{
							"-m comment --comment \"egw-HAe35Kaffr8R0mLj\" --jump EGRESSGATEWAY-MARK-REQUEST",
							"-m comment --comment \"cali:6gwbT8clXdHdC1b1\" -j cali-PREROUTING",
						},
						"POSTROUTING": []string{
							"-m comment --comment \"egw-1ecaPSxhNEc4Ylv_\" -m mark --mark 0x11000000/0xffffffff --jump ACCEPT",
							"-m comment --comment \"cali:O3lYWMrLQYEMJtB5\" -j cali-POSTROUTING",
						},
						"FORWARD": []string{
							"-m comment --comment \"egw-ogGr4WMsrCe4gJFJ\" -m mark --mark 0x11000000/0xffffffff --jump MARK --set-mark 0x12000000/0x12000000",
						},
					},
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
			&egressv1.EgressGateway{
				ObjectMeta: metav1.ObjectMeta{Name: "gateway"},
				Spec:       egressv1.EgressGatewaySpec{},
				Status: egressv1.EgressGatewayStatus{NodeList: []egressv1.SelectedEgressNode{{
					Name:   "node1",
					Active: true,
				}}},
			},
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
				expMangle: []map[string][]string{
					{
						"EGRESSGATEWAY-MARK-REQUEST": []string{"" +
							"-m comment --comment \"egw-Mio2l_qQ2kKhiz2d\" -m set --match-set egress-src-v4-c0eb42ab804da452e src -m set --match-set egress-dst-v4-c0eb42ab804da452e dst --jump MARK --set-mark 0x11000000/0xffffffff",
						},
						"PREROUTING": []string{
							"-m comment --comment \"egw-HAe35Kaffr8R0mLj\" --jump EGRESSGATEWAY-MARK-REQUEST",
							"-m comment --comment \"cali:6gwbT8clXdHdC1b1\" -j cali-PREROUTING",
						},
						"POSTROUTING": []string{
							"-m comment --comment \"egw-1ecaPSxhNEc4Ylv_\" -m mark --mark 0x11000000/0xffffffff --jump ACCEPT",
							"-m comment --comment \"cali:O3lYWMrLQYEMJtB5\" -j cali-POSTROUTING",
						},
						"FORWARD": []string{
							"-m comment --comment \"egw-ogGr4WMsrCe4gJFJ\" -m mark --mark 0x11000000/0xffffffff --jump MARK --set-mark 0x12000000/0x12000000",
						},
					},
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

type mockMutex struct {
	Held     bool
	WasTaken bool
}

func (m *mockMutex) Lock() {
	if m.Held {
		panic("mutex already held")
	}
	m.Held = true
	m.WasTaken = true
}

func (m *mockMutex) Unlock() {
	if !m.Held {
		panic("mutex not held")
	}
	m.Held = false
}

func TestGetPodIPs(t *testing.T) {
	tests := []struct {
		name         string
		args         []corev1.PodIP
		wantIpv4List []string
		wantIpv6List []string
	}{
		{
			name: "ipv4 with ipv6 list",
			args: []corev1.PodIP{
				{
					IP: "10.21.180.91",
				},
				{
					IP: "some invalid IP address",
				},
				{
					IP: "fd00:21::a203:748a:5f1a:c780",
				},
				{
					IP: "fd00:21::a203:748a:5f1a:c781",
				},
			},
			wantIpv4List: []string{
				"10.21.180.91",
			},
			wantIpv6List: []string{
				"fd00:21::a203:748a:5f1a:c780",
				"fd00:21::a203:748a:5f1a:c781",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			gotIpv4List, gotIpv6List := getPodIPsBy(tt.args)
			assert.Equalf(t, tt.wantIpv4List, gotIpv4List, "getPodIPsBy(%v)", tt.args)
			assert.Equalf(t, tt.wantIpv6List, gotIpv6List, "getPodIPsBy(%v)", tt.args)
		})
	}
}
