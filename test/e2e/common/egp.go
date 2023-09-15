// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"fmt"
	"time"

	"github.com/go-faker/faker/v4"
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
		ObjectMeta: metav1.ObjectMeta{GenerateName: "policy-" + faker.Word(), Namespace: "default"},
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
		ObjectMeta: metav1.ObjectMeta{GenerateName: "policy-" + faker.Word()},
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
