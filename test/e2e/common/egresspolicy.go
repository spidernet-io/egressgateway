// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spidernet-io/e2eframework/framework"
	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/test/e2e/err"
	"k8s.io/apimachinery/pkg/api/errors"
)

func GenerateEgressPolicyYaml(policyName, gatewayName, namespace string, egressIP v1beta1.EgressIP, labels map[string]string, podSubnet, dest []string) *v1beta1.EgressPolicy {
	return &v1beta1.EgressPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyName,
			Namespace: namespace,
		},
		Spec: v1beta1.EgressPolicySpec{
			EgressGatewayName: gatewayName,
			EgressIP:          egressIP,
			AppliedTo: v1beta1.AppliedTo{
				PodSelector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				PodSubnet: podSubnet,
			},
			DestSubnet: dest,
		},
	}
}

func GenerateEgressClusterPolicyYaml(policyName, gatewayName string, egressIP v1beta1.EgressIP, labels map[string]string, podSubnet, dest []string) *v1beta1.EgressClusterPolicy {
	return &v1beta1.EgressClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: policyName,
		},
		Spec: v1beta1.EgressClusterPolicySpec{
			EgressGatewayName: gatewayName,
			EgressIP:          egressIP,
			AppliedTo: v1beta1.ClusterAppliedTo{
				PodSelector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				PodSubnet: &podSubnet,
			},
			DestSubnet: dest,
		},
	}
}

func CreateEgressPolicy(f *framework.Framework, policy client.Object, opts ...client.CreateOption) error {
	return f.CreateResource(policy, opts...)
}

func GetEgressPolicy(f *framework.Framework, name, namespace string, policy client.Object) error {
	key := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	return f.GetResource(key, policy)
}

func DeleteEgressPolicy(f *framework.Framework, policy client.Object, opts ...client.DeleteOption) error {
	return f.DeleteResource(policy, opts...)
}

// DeleteEgressPolicyIfExists delete egressPolicy if its exists
func DeleteEgressPolicyIfExists(f *framework.Framework, name, namespace string, policy client.Object, duration time.Duration) error {
	e := GetEgressPolicy(f, name, namespace, policy)
	if e == nil {
		return DeleteEgressPolicyUntilFinish(f, name, namespace, policy, duration)
	}
	if errors.IsNotFound(e) {
		return nil
	}
	return e
}

func DeleteEgressPolicyUntilFinish(f *framework.Framework, name, namespace string, policy client.Object, duration time.Duration, opts ...client.DeleteOption) error {
	e := DeleteEgressPolicy(f, policy, opts...)
	if errors.IsNotFound(e) {
		return nil
	}
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
			e = GetEgressPolicy(f, name, namespace, policy)
			if errors.IsNotFound(e) {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func EditEgressPolicy(f *framework.Framework, policy *v1beta1.EgressPolicy, labels map[string]string, dst []string, opts ...client.UpdateOption) error {
	if dst != nil {
		policy.Spec.DestSubnet = dst
	}
	if labels != nil {
		if policy.Spec.AppliedTo.PodSelector == nil {
			policy.Spec.AppliedTo.PodSelector = new(metav1.LabelSelector)
			policy.Spec.AppliedTo.PodSelector.MatchLabels = labels
		} else {
			policy.Spec.AppliedTo.PodSelector.MatchLabels = labels
		}
	}
	return f.UpdateResource(policy, opts...)
}

func EditEgressClusterPolicy(f *framework.Framework, policy *v1beta1.EgressClusterPolicy, labels map[string]string, dst []string, opts ...client.UpdateOption) error {
	if dst != nil {
		policy.Spec.DestSubnet = dst
	}
	if labels != nil {
		if policy.Spec.AppliedTo.PodSelector == nil {
			policy.Spec.AppliedTo.PodSelector = new(metav1.LabelSelector)
			policy.Spec.AppliedTo.PodSelector.MatchLabels = labels
		} else {
			policy.Spec.AppliedTo.PodSelector.MatchLabels = labels

		}
	}
	return f.UpdateResource(policy, opts...)
}

func WaitEgressPolicyEipUpdated(f *framework.Framework, name, namespace, expectV4Eip, expectV6Eip string, enableV4, enableV6 bool, timeout time.Duration) (v4Eip, v6Eip, allocatorPolicy string, useNodeIP bool, e error) {
	policy := new(v1beta1.EgressPolicy)
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return "", "", "", true, ERR_TIMEOUT
		default:
			e = GetEgressPolicy(f, name, namespace, policy)
			if e != nil {
				return "", "", "", true, e
			}
			if enableV4 {
				if len(expectV4Eip) != 0 {
					if policy.Status.Eip.Ipv4 != expectV4Eip {
						time.Sleep(time.Millisecond * 100)
						break
					}
				} else {
					if len(policy.Status.Eip.Ipv4) == 0 {
						time.Sleep(time.Millisecond * 100)
						break
					}
				}
				v4Eip = policy.Status.Eip.Ipv4
			}

			if enableV6 {
				if len(expectV6Eip) != 0 {
					if policy.Status.Eip.Ipv6 != expectV6Eip {
						time.Sleep(time.Millisecond * 100)
						break
					}
				} else {
					if len(policy.Status.Eip.Ipv6) == 0 {
						time.Sleep(time.Millisecond * 100)
						break
					}
				}
				v6Eip = policy.Status.Eip.Ipv6
			}

			allocatorPolicy = policy.Spec.EgressIP.AllocatorPolicy
			useNodeIP = policy.Spec.EgressIP.UseNodeIP
			return
		}
	}
}

func WaitEgressClusterPolicyEipUpdated(f *framework.Framework, name, expectV4Eip, expectV6Eip string, enableV4, enableV6 bool, timeout time.Duration) (v4Eip, v6Eip, allocatorPolicy string, useNodeIP bool, e error) {
	policy := new(v1beta1.EgressClusterPolicy)
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return "", "", "", true, ERR_TIMEOUT
		default:
			e = GetEgressPolicy(f, name, "", policy)
			if e != nil {
				return "", "", "", true, e
			}
			if enableV4 {
				if len(expectV4Eip) != 0 {
					if policy.Status.Eip.Ipv4 != expectV4Eip {
						time.Sleep(time.Millisecond * 100)
						break
					}
				} else {
					if len(policy.Status.Eip.Ipv4) == 0 {
						time.Sleep(time.Millisecond * 100)
						break
					}
				}
				v4Eip = policy.Status.Eip.Ipv4
			}

			if enableV6 {
				if len(expectV6Eip) != 0 {
					if policy.Status.Eip.Ipv6 != expectV6Eip {
						time.Sleep(time.Millisecond * 100)
						break
					}
				} else {
					if len(policy.Status.Eip.Ipv6) == 0 {
						time.Sleep(time.Millisecond * 100)
						break
					}
				}
				v6Eip = policy.Status.Eip.Ipv6
			}

			allocatorPolicy = policy.Spec.EgressIP.AllocatorPolicy
			useNodeIP = policy.Spec.EgressIP.UseNodeIP
			return
		}
	}
}
