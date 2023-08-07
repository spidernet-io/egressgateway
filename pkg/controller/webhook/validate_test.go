// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
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

func TestValidateEgressNode(t *testing.T) {
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
				Spec: v1beta1.EgressNodeSpec{},
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
