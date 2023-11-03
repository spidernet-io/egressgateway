// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/google/uuid"

	econfig "github.com/spidernet-io/egressgateway/pkg/config"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateEgressPolicyNew(ctx context.Context, cli client.Client, cfg econfig.FileConfig,
	egw string, podLabel map[string]string) (*egressv1.EgressPolicy, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	res := &egressv1.EgressPolicy{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "policy-" + uuid.NewString(), Namespace: "default"},
		Spec: egressv1.EgressPolicySpec{
			EgressGatewayName: egw,
			AppliedTo: egressv1.AppliedTo{PodSelector: &metav1.LabelSelector{
				MatchLabels: podLabel,
			}},
			DestSubnet: []string{},
		},
	}

	err := cli.Create(ctx, res)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			_ = DeleteObj(context.Background(), cli, res)
			return nil, fmt.Errorf("create EgressPolicy time out")
		default:
			err := cli.Get(ctx, types.NamespacedName{Namespace: res.Namespace, Name: res.Name}, res)
			if err != nil {
				return nil, err
			}

			cond1 := cfg.EnableIPv4 && res.Status.Eip.Ipv4 != ""
			cond2 := cfg.EnableIPv6 && res.Status.Eip.Ipv6 != ""

			if cond1 && cond2 {
				return res, nil
			}

			if cfg.EnableIPv4 && cond1 {
				return res, nil
			}

			if cfg.EnableIPv6 && cond2 {
				return res, nil
			}

			time.Sleep(time.Second / 2)
		}
	}
}

func CreateEgressClusterPolicy(ctx context.Context, cli client.Client, cfg econfig.FileConfig, egw string, podLabel map[string]string) (*egressv1.EgressClusterPolicy, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	res := &egressv1.EgressClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "cluster-policy-" + uuid.NewString()},
		Spec: egressv1.EgressClusterPolicySpec{
			EgressGatewayName: egw,
			AppliedTo: egressv1.ClusterAppliedTo{PodSelector: &metav1.LabelSelector{
				MatchLabels: podLabel,
			}},
			DestSubnet: []string{},
		},
	}

	if err := cli.Create(ctx, res); err != nil {
		return nil, err
	}

	for {
		select {
		case <-ctx.Done():
			_ = DeleteObj(context.Background(), cli, res)
			return nil, fmt.Errorf("create EgressClusterPolicy time out")
		default:
			err := cli.Get(ctx, types.NamespacedName{Namespace: res.Namespace, Name: res.Name}, res)
			if err != nil {
				return nil, err
			}

			cond1 := cfg.EnableIPv4 && res.Status.Eip.Ipv4 != ""
			cond2 := cfg.EnableIPv6 && res.Status.Eip.Ipv6 != ""

			if cond1 && cond2 {
				return res, nil
			}

			if cfg.EnableIPv4 && cond1 {
				return res, nil
			}

			if cfg.EnableIPv6 && cond2 {
				return res, nil
			}

			time.Sleep(time.Second / 2)
		}
	}
}

func CreateEgressPolicyCustom(ctx context.Context, cli client.Client, setUp func(egp *egressv1.EgressPolicy)) (*egressv1.EgressPolicy, error) {
	name := "egp-" + uuid.NewString()
	ns := "default"
	res := &egressv1.EgressPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
	}

	setUp(res)

	err := cli.Create(ctx, res)
	if err != nil {
		return nil, fmt.Errorf("error:\n%w\npolicy yaml:\n%s\n", err, GetObjYAML(res))
	}
	return res, nil
}

func CreateEgressClusterPolicyCustom(ctx context.Context, cli client.Client, setUp func(egcp *egressv1.EgressClusterPolicy)) (*egressv1.EgressClusterPolicy, error) {
	name := "egcp-" + uuid.NewString()
	res := &egressv1.EgressClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}

	setUp(res)

	err := cli.Create(ctx, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func CheckEgressPolicyStatusSynced(ctx context.Context, cli client.Client, egp *egressv1.EgressPolicy, expectStatus *egressv1.EgressPolicyStatus, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("check EgressPolicyStatus synced timeout")
		default:
			key := types.NamespacedName{Name: egp.Name, Namespace: egp.Namespace}
			err := cli.Get(ctx, key, egp)
			if err != nil {
				return nil
			}
			if reflect.DeepEqual(*expectStatus, egp.Status) {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func CheckEgressClusterPolicyStatusSynced(ctx context.Context, cli client.Client, egcp *egressv1.EgressClusterPolicy, expectStatus *egressv1.EgressPolicyStatus, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("check EgressPolicyStatus synced timeout")
		default:
			key := types.NamespacedName{Name: egcp.Name}
			err := cli.Get(ctx, key, egcp)
			if err != nil {
				return nil
			}
			if reflect.DeepEqual(*expectStatus, egcp.Status) {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

// DeleteEgressPolicies delete egressPolicies
func DeleteEgressPolicies(ctx context.Context, cli client.Client, egps []*egressv1.EgressPolicy) error {
	for _, egp := range egps {
		err := DeleteObj(ctx, cli, egp)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteEgressClusterPolicies delete egressClusterPolicies
func DeleteEgressClusterPolicies(ctx context.Context, cli client.Client, egcps []*egressv1.EgressClusterPolicy) error {
	for _, egcp := range egcps {
		err := DeleteObj(ctx, cli, egcp)
		if err != nil {
			return err
		}
	}
	return nil
}

func WaitEgressPoliciesDeleted(ctx context.Context, cli client.Client, egps []*egressv1.EgressPolicy, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout to wait egressPolicies deleted")
		default:
			err := DeleteEgressPolicies(ctx, cli, egps)
			if err != nil {
				time.Sleep(time.Second / 2)
				continue
			}
			for _, egp := range egps {
				err := cli.Get(ctx, types.NamespacedName{Name: egp.Name, Namespace: egp.Namespace}, egp)
				if err == nil {
					time.Sleep(time.Second / 2)
					continue
				}
			}
			return nil
		}
	}
}

func WaitEgressClusterPoliciesDeleted(ctx context.Context, cli client.Client, egcps []*egressv1.EgressClusterPolicy, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout to wait egressPolicies deleted")
		default:
			err := DeleteEgressClusterPolicies(ctx, cli, egcps)
			if err != nil {
				time.Sleep(time.Second / 2)
				continue
			}
			for _, egcp := range egcps {
				err := cli.Get(ctx, types.NamespacedName{Name: egcp.Name}, egcp)
				if err == nil {
					time.Sleep(time.Second / 2)
					continue
				}
			}
			return nil
		}
	}
}

// WaitEgressPolicyStatusReady waits for the EgressPolicy status.Eip to be allocated after the EgressPolicy is created
func WaitEgressPolicyStatusReady(ctx context.Context, cli client.Client, egp *egressv1.EgressPolicy, v4Enabled, v6Enabled bool, timeout time.Duration) error {
	if !v4Enabled && !v6Enabled {
		return fmt.Errorf("both v4 and v6 are not enabled")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var v4Ok, v6Ok bool

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout to wait egressPolicy status ready")
		default:
			err := cli.Get(ctx, types.NamespacedName{Name: egp.Name, Namespace: egp.Namespace}, egp)
			if err != nil {
				time.Sleep(time.Second / 2)
				continue
			}
			if egp.Spec.EgressIP.UseNodeIP {
				if v4Enabled && len(egp.Status.Eip.Ipv4) != 0 {
					v4Ok = true
				}
				if v6Enabled && len(egp.Status.Eip.Ipv6) != 0 {
					v6Ok = true
				}
			} else {
				if len(egp.Status.Eip.Ipv4) == 0 && len(egp.Status.Eip.Ipv6) == 0 {
					return nil
				}
			}
			if v4Enabled && v6Enabled {
				if v4Ok && v6Ok {
					return nil
				}
			} else if v4Enabled && v4Ok {
				return nil
			} else if v6Enabled && v6Ok {
				return nil
			}
			time.Sleep(time.Second / 2)
		}
	}
}

// CreateEgressPolicyWithEipAllocatorRR  creates an egressPolicy  and sets Spec.EgressIP.AllocatorPolicy to "rr"
func CreateEgressPolicyWithEipAllocatorRR(ctx context.Context, cli client.Client, egw *egressv1.EgressGateway, labels map[string]string) (*egressv1.EgressPolicy, error) {
	return CreateEgressPolicyCustom(ctx, cli,
		func(egp *egressv1.EgressPolicy) {
			egp.Spec.EgressGatewayName = egw.Name
			egp.Spec.EgressIP.AllocatorPolicy = egressv1.EipAllocatorRR
			// egp.Spec.EgressIP.AllocatorPolicy = egressv1.EipAllocatorDefault
			egp.Spec.AppliedTo.PodSelector = &metav1.LabelSelector{MatchLabels: labels}
		})
}

// CreateEgressPoliciesForPods
func CreateEgressPoliciesForPods(ctx context.Context, cli client.Client, egw *egressv1.EgressGateway, pods []*corev1.Pod, v4Enabled, v6Enabled bool, timeout time.Duration) (
	[]*egressv1.EgressPolicy, map[*corev1.Pod]*egressv1.EgressPolicy, error) {
	egps := make([]*egressv1.EgressPolicy, 0)
	pod2Policy := make(map[*corev1.Pod]*egressv1.EgressPolicy, 0)
	for _, pod := range pods {
		egp, err := CreateEgressPolicyWithEipAllocatorRR(ctx, cli, egw, pod.Labels)
		if err != nil {
			return nil, nil, err
		}
		// wait egressPolicy status updated
		err = WaitEgressPolicyStatusReady(ctx, cli, egp, v4Enabled, v6Enabled, timeout)
		if err != nil {
			return nil, nil, err
		}
		// get egp after egressPolicy updated
		err = cli.Get(ctx, types.NamespacedName{Name: egp.Name, Namespace: egp.Namespace}, egp)
		if err != nil {
			return nil, nil, err
		}
		egps = append(egps, egp)
		pod2Policy[pod] = egp
	}

	return egps, pod2Policy, nil
}
