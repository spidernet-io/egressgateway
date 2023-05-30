// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"github.com/spidernet-io/e2eframework/framework"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/err"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func GenerateEgressPolicyYaml(name string, labels map[string]string, dest []string) *egressv1.EgressPolicy {
	return &egressv1.EgressPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: egressv1.EgressGatewayPolicySpec{
			AppliedTo: egressv1.AppliedTo{
				PodSelector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
			},
			DestSubnet: dest,
		},
	}
}

func CreateEgressPolicy(f *framework.Framework, policy *egressv1.EgressPolicy, opts ...client.CreateOption) error {
	return f.CreateResource(policy, opts...)
}

func GetEgressPolicy(f *framework.Framework, name string, policy *egressv1.EgressPolicy) error {
	key := client.ObjectKey{
		Name: name,
	}
	return f.GetResource(key, policy)
}

func DeleteEgressPolicy(f *framework.Framework, policy *egressv1.EgressPolicy, opts ...client.DeleteOption) error {
	return f.DeleteResource(policy, opts...)
}

// DeleteEgressPolicyIfExists delete egressPolicy if its exists
func DeleteEgressPolicyIfExists(f *framework.Framework, name string, duration time.Duration) error {
	policy := new(egressv1.EgressPolicy)
	e := GetEgressPolicy(f, name, policy)
	if e == nil {
		return DeleteEgressPolicyUntilFinish(f, policy, duration)
	}
	return nil
}

func DeleteEgressPolicyUntilFinish(f *framework.Framework, policy *egressv1.EgressPolicy, duration time.Duration, opts ...client.DeleteOption) error {
	e := DeleteEgressPolicy(f, policy, opts...)
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return err.TIME_OUT
		default:
			e = GetEgressPolicy(f, policy.Name, policy)
			if e != nil {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func EditEgressPolicy(f *framework.Framework, policy *egressv1.EgressPolicy, labels map[string]string, dst []string, opts ...client.UpdateOption) error {
	if labels == nil && dst == nil {
		return INVALID_INPUT
	}
	policy.Spec.DestSubnet = dst
	policy.Spec.AppliedTo.PodSelector.MatchLabels = labels
	return f.UpdateResource(policy, opts...)
}
