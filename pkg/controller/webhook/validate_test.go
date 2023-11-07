// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

func TestValidateEgressGateway(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		existingResources []runtime.Object
		newResource       *v1beta1.EgressGateway
		expAllow          bool
		expErrMessage     string
	}{
		"EgressGateway the EIP format is incorrect": {
			existingResources: nil,
			newResource: &v1beta1.EgressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eg-test",
				},
				Spec: v1beta1.EgressGatewaySpec{
					Ippools: v1beta1.Ippools{
						IPv4: []string{"1.1.1.1x"},
					},
				},
			},
			expAllow: false,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			marshalledRequestObject, err := json.Marshal(c.newResource)
			assert.NoError(t, err)

			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			cli := builder.Build()
			conf := &config.Config{
				FileConfig: config.FileConfig{
					EnableIPv4: true,
					EnableIPv6: false,
				},
			}

			validator := ValidateHook(cli, conf)
			resp := validator.Handle(ctx, admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Name: c.newResource.Name,
					Kind: metav1.GroupVersionKind{
						Kind: "EgressGateway",
					},
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: marshalledRequestObject,
					},
				},
			})

			assert.Equal(t, c.expAllow, resp.Allowed)
			if c.expErrMessage != "" {
				assert.Equal(t, c.expErrMessage, resp.AdmissionResponse.Result.Message)
			}
		})
	}
}

func TestValidateEgressPolicy(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		existingResources []client.Object
		spec              v1beta1.EgressPolicySpec
		expAllow          bool
		expErrMessage     string
	}{
		"case, valid": {
			existingResources: []client.Object{
				&v1beta1.EgressGateway{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
					Spec: v1beta1.EgressGatewaySpec{
						Ippools: v1beta1.Ippools{
							IPv4: []string{"172.18.1.2-172.18.1.5"},
							IPv6: []string{"fc00:f853:ccd:e793:a::3-fc00:f853:ccd:e793:a::6"},
						},
					},
				},
			},
			spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "test",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: false,
					IPv4:      "",
					IPv6:      "",
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
				DestSubnet: []string{
					"192.168.1.1/24",
					"1.1.1.1/32",
					"10.0.6.1/16",
					"fd00::21/112",
				},
			},
			expAllow: true,
		},
		"case1, not valid": {
			existingResources: nil,
			spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "test",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: false,
					IPv4:      "",
					IPv6:      "",
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
				DestSubnet: []string{
					"1.1.1.1",
				},
			},
			expAllow: false,
		},
		"case2, not valid": {
			existingResources: nil,
			spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "test",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: false,
					IPv4:      "",
					IPv6:      "",
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
				DestSubnet: []string{
					"1.1.1.1999/24",
				},
			},
			expAllow: false,
		},
		"case3, not valid": {
			existingResources: nil,
			spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "test",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: false,
					IPv4:      "",
					IPv6:      "",
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
				DestSubnet: []string{
					"---",
				},
			},
			expAllow: false,
		},
		"case4 empty EgressGatewayName": {
			existingResources: nil,
			spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo:         v1beta1.AppliedTo{},
				DestSubnet:        []string{},
			},
			expAllow: false,
		},
		"case5, create with eip": {
			existingResources: []client.Object{
				&v1beta1.EgressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1beta1.EgressGatewaySpec{
						Ippools: v1beta1.Ippools{
							IPv4: []string{"10.6.1.21"},
							IPv6: []string{"fd00::1"},
						},
						NodeSelector: v1beta1.NodeSelector{},
					},
					Status: v1beta1.EgressGatewayStatus{},
				},
			},
			spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "test",
				EgressIP: v1beta1.EgressIP{
					IPv4: "10.6.1.21",
					IPv6: "fd00::1",
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: []string{},
			},
			expAllow: true,
		},
		"case6 empty AppliedTo": {
			existingResources: nil,
			spec: v1beta1.EgressPolicySpec{
				EgressGatewayName: "test",
				AppliedTo:         v1beta1.AppliedTo{},
				DestSubnet:        []string{},
			},
			expAllow: false,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {

			policy := &v1beta1.EgressPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy",
				},
				Spec: c.spec,
			}

			marshalledRequestObject, err := json.Marshal(policy)
			assert.NoError(t, err)

			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			builder.WithObjects(c.existingResources...)
			builder.WithStatusSubresource(c.existingResources...)
			cli := builder.Build()
			conf := &config.Config{
				FileConfig: config.FileConfig{
					EnableIPv4: true,
					EnableIPv6: true,
				},
			}

			validator := ValidateHook(cli, conf)
			resp := validator.Handle(ctx, admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Name: policy.Name,
					Kind: metav1.GroupVersionKind{
						Kind: "EgressPolicy",
					},
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: marshalledRequestObject,
					},
				},
			})

			assert.Equal(t, c.expAllow, resp.Allowed)
			if c.expErrMessage != "" {
				assert.Equal(t, c.expErrMessage, resp.AdmissionResponse.Result.Message)
			}
		})
	}
}

func TestUpdateEgressPolicy(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		existingResources []runtime.Object
		old               v1beta1.EgressPolicySpec
		new               v1beta1.EgressPolicySpec
		expAllow          bool
		expErrMessage     string
	}{
		"test change ipv4": {
			existingResources: nil,
			old: v1beta1.EgressPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					IPv4:      "10.6.1.21",
					IPv6:      "",
					UseNodeIP: false,
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
				Priority: 0,
			},
			new: v1beta1.EgressPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					IPv4:            "10.6.1.22",
					IPv6:            "",
					UseNodeIP:       false,
					AllocatorPolicy: "",
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
				Priority: 0,
			},
			expAllow:      false,
			expErrMessage: "",
		},
		"change useNodeIP": {
			existingResources: nil,
			old: v1beta1.EgressPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: true,
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			new: v1beta1.EgressPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: false,
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			expAllow:      false,
			expErrMessage: "",
		},
		"change egress gateway name": {
			existingResources: nil,
			old: v1beta1.EgressPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: true,
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			new: v1beta1.EgressPolicySpec{
				EgressGatewayName: "b",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: true,
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			expAllow:      false,
			expErrMessage: "",
		},
		"change ipv6": {
			existingResources: nil,
			old: v1beta1.EgressPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					IPv6: "fd00::1",
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			new: v1beta1.EgressPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					IPv6: "fd00::2",
				},
				AppliedTo: v1beta1.AppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			expAllow:      false,
			expErrMessage: "",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			oldPolicy := &v1beta1.EgressPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy",
				},
				Spec: c.old,
			}

			newPolicy := &v1beta1.EgressPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy",
				},
				Spec: c.new,
			}

			oldObj, err := json.Marshal(oldPolicy)
			assert.NoError(t, err)

			newObj, err := json.Marshal(newPolicy)
			assert.NoError(t, err)

			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			cli := builder.Build()
			conf := &config.Config{
				FileConfig: config.FileConfig{
					EnableIPv4: true,
					EnableIPv6: true,
				},
			}

			validator := ValidateHook(cli, conf)
			resp := validator.Handle(ctx, admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Name: oldPolicy.Name,
					Kind: metav1.GroupVersionKind{
						Kind: "EgressPolicy",
					},
					Operation: admissionv1.Update,
					OldObject: runtime.RawExtension{
						Raw: oldObj,
					},
					Object: runtime.RawExtension{
						Raw: newObj,
					},
				},
			})

			assert.Equal(t, c.expAllow, resp.Allowed)
			if c.expErrMessage != "" {
				assert.Equal(t, c.expErrMessage, resp.AdmissionResponse.Result.Message)
			}
		})
	}
}

func TestValidateEgressTunnel(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		newResource   *v1beta1.EgressTunnel
		expAllow      bool
		expErrMessage string
	}{
		"all valid": {
			newResource: &v1beta1.EgressTunnel{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec: v1beta1.EgressTunnelSpec{},
			},
			expAllow: true,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			marshalledRequestObject, err := json.Marshal(c.newResource)
			assert.NoError(t, err)

			req := admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Name: c.newResource.Name,
					Kind: metav1.GroupVersionKind{
						Kind: "EgressTunnel",
					},
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: marshalledRequestObject,
					},
				},
			}

			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			cli := builder.Build()
			conf := &config.Config{
				FileConfig: config.FileConfig{
					EnableIPv4: true,
					EnableIPv6: true,
				},
			}
			validator := ValidateHook(cli, conf)
			resp := validator.Handle(ctx, req)

			assert.Equal(t, c.expAllow, resp.Allowed)
			if c.expErrMessage != "" {
				assert.Equal(t, c.expErrMessage, resp.AdmissionResponse.Result.Message)
			}
		})
	}
}

func TestValidateEgressClusterPolicy(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		existingResources []client.Object
		spec              v1beta1.EgressClusterPolicySpec
		expAllow          bool
		expErrMessage     string
	}{
		"case, valid": {
			existingResources: []client.Object{
				&v1beta1.EgressGateway{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
					Spec: v1beta1.EgressGatewaySpec{
						Ippools: v1beta1.Ippools{
							IPv4: []string{"172.18.1.2-172.18.1.5"},
							IPv6: []string{"fc00:f853:ccd:e793:a::3-fc00:f853:ccd:e793:a::6"},
						},
					},
				},
			},
			spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "test",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: false,
					IPv4:      "",
					IPv6:      "",
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
				DestSubnet: []string{
					"192.168.1.1/24",
					"1.1.1.1/32",
					"10.0.6.1/16",
					"fd00::21/112",
				},
			},
			expAllow: true,
		},
		"case1: Not valid when both PodSelector and DestSubnet exist": {
			existingResources: nil,
			spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "test",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: false,
					IPv4:      "",
					IPv6:      "",
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					PodSubnet: &[]string{"10.10.0.0/16"},
				},
				DestSubnet: []string{
					"192.12.0.0/16",
				},
			},
			expAllow: false,
		},
		"case2: Not valid when DestSubnet format is invalid": {
			existingResources: nil,
			spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "test",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: false,
					IPv4:      "",
					IPv6:      "",
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
				DestSubnet: []string{
					"1.1.1.1999/24",
				},
			},
			expAllow: false,
		},
		"case3: Not valid when empty EgressGatewayName": {
			existingResources: nil,
			spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "",
				EgressIP:          v1beta1.EgressIP{},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				},
				DestSubnet: []string{},
			},
			expAllow: false,
		},
		"case4: create with eip": {
			existingResources: []client.Object{
				&v1beta1.EgressGateway{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: v1beta1.EgressGatewaySpec{
						Ippools: v1beta1.Ippools{
							IPv4: []string{"10.6.1.21"},
							IPv6: []string{"fd00::1"},
						},
						NodeSelector: v1beta1.NodeSelector{},
					},
					Status: v1beta1.EgressGatewayStatus{},
				},
			},
			spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "test",
				EgressIP: v1beta1.EgressIP{
					IPv4: "10.6.1.21",
					IPv6: "fd00::1",
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: []string{},
			},
			expAllow: true,
		},
		"case5 empty AppliedTo": {
			existingResources: nil,
			spec: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "test",
				AppliedTo:         v1beta1.ClusterAppliedTo{},
				DestSubnet:        []string{},
			},
			expAllow: false,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {

			policy := &v1beta1.EgressClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy",
				},
				Spec: c.spec,
			}

			marshalledRequestObject, err := json.Marshal(policy)
			assert.NoError(t, err)

			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			builder.WithObjects(c.existingResources...)
			builder.WithStatusSubresource(c.existingResources...)
			cli := builder.Build()
			conf := &config.Config{
				FileConfig: config.FileConfig{
					EnableIPv4: true,
					EnableIPv6: true,
				},
			}

			validator := ValidateHook(cli, conf)
			resp := validator.Handle(ctx, admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Name: policy.Name,
					Kind: metav1.GroupVersionKind{
						Kind: "EgressClusterPolicy",
					},
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: marshalledRequestObject,
					},
				},
			})

			assert.Equal(t, c.expAllow, resp.Allowed)
			if c.expErrMessage != "" {
				assert.Equal(t, c.expErrMessage, resp.AdmissionResponse.Result.Message)
			}
		})
	}
}

func TestUpdateEgressClusterPolicy(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		existingResources []runtime.Object
		old               v1beta1.EgressClusterPolicySpec
		new               v1beta1.EgressClusterPolicySpec
		expAllow          bool
		expErrMessage     string
	}{
		"test change ipv4": {
			existingResources: nil,
			old: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					IPv4:      "10.6.1.21",
					IPv6:      "",
					UseNodeIP: false,
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
				Priority: 0,
			},
			new: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					IPv4:            "10.6.1.22",
					IPv6:            "",
					UseNodeIP:       false,
					AllocatorPolicy: "",
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
				Priority: 0,
			},
			expAllow:      false,
			expErrMessage: "",
		},
		"change useNodeIP": {
			existingResources: nil,
			old: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: true,
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			new: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: false,
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			expAllow:      false,
			expErrMessage: "",
		},
		"change egress gateway name": {
			existingResources: nil,
			old: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: true,
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			new: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "b",
				EgressIP: v1beta1.EgressIP{
					UseNodeIP: true,
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			expAllow:      false,
			expErrMessage: "",
		},
		"change ipv6": {
			existingResources: nil,
			old: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					IPv6: "fd00::1",
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			new: v1beta1.EgressClusterPolicySpec{
				EgressGatewayName: "a",
				EgressIP: v1beta1.EgressIP{
					IPv6: "fd00::2",
				},
				AppliedTo: v1beta1.ClusterAppliedTo{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
				}, DestSubnet: nil,
			},
			expAllow:      false,
			expErrMessage: "",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			oldPolicy := &v1beta1.EgressClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy",
				},
				Spec: c.old,
			}

			newPolicy := &v1beta1.EgressClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy",
				},
				Spec: c.new,
			}

			oldObj, err := json.Marshal(oldPolicy)
			assert.NoError(t, err)

			newObj, err := json.Marshal(newPolicy)
			assert.NoError(t, err)

			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			cli := builder.Build()
			conf := &config.Config{
				FileConfig: config.FileConfig{
					EnableIPv4: true,
					EnableIPv6: true,
				},
			}

			validator := ValidateHook(cli, conf)
			resp := validator.Handle(ctx, admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Name: oldPolicy.Name,
					Kind: metav1.GroupVersionKind{
						Kind: "EgressClusterPolicy",
					},
					Operation: admissionv1.Update,
					OldObject: runtime.RawExtension{
						Raw: oldObj,
					},
					Object: runtime.RawExtension{
						Raw: newObj,
					},
				},
			})

			assert.Equal(t, c.expAllow, resp.Allowed)
			if c.expErrMessage != "" {
				assert.Equal(t, c.expErrMessage, resp.AdmissionResponse.Result.Message)
			}
		})
	}
}

func TestCountAvailableIP(t *testing.T) {
	tests := []struct {
		name  string
		args  *egressv1.EgressGateway
		want4 int
		want6 int
	}{
		{
			name: "case1-ipv4",
			args: &egressv1.EgressGateway{
				Spec: egressv1.EgressGatewaySpec{
					Ippools: egressv1.Ippools{
						IPv4: []string{
							"10.6.1.21-10.6.1.30",
						},
						IPv6: []string{},
					},
				},
				Status: v1beta1.EgressGatewayStatus{
					NodeList: []egressv1.EgressIPStatus{
						{
							Name: "node1",
							Eips: []egressv1.Eips{{IPv4: "10.6.1.21"}},
						},
					},
				},
			},
			want4: 9,
			want6: 0,
		},
		{
			name: "case2-dual",
			args: &egressv1.EgressGateway{
				Spec: egressv1.EgressGatewaySpec{
					Ippools: egressv1.Ippools{
						IPv4: []string{
							"10.6.1.21",
							"10.6.1.22",
						},
						IPv6: []string{
							"fd00::1-fd00::2",
						},
					},
				},
				Status: v1beta1.EgressGatewayStatus{
					NodeList: []egressv1.EgressIPStatus{
						{
							Name: "node1",
							Eips: []egressv1.Eips{{IPv4: "10.6.1.21", IPv6: "fd00::1"}},
						},
					},
				},
			},
			want4: 1,
			want6: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got4, got6, err := countGatewayAvailableIP(tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if got4 != tt.want4 {
				t.Fatalf("got4 %d, want4 %d", got4, tt.want4)
			}
			if got6 != tt.want6 {
				t.Fatalf("got6 %d, want6 %d", got6, tt.want6)
			}
		})
	}
}
