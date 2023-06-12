// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

func TestValidateEgressGateway(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		existingResources []runtime.Object
		newResource       *v1.EgressGateway
		expAllow          bool
		expErrMessage     string
	}{
		"EgressGateway the EIP format is incorrect": {
			existingResources: nil,
			newResource: &v1.EgressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eg-test",
				},
				Spec: v1.EgressGatewaySpec{
					Ippools: v1.Ippools{
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

func TestValidateEgressGatewayPolicy(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		existingResources []runtime.Object
		destSubnet        []string
		expAllow          bool
		expErrMessage     string
	}{
		"case, valid": {
			existingResources: nil,
			destSubnet: []string{
				"192.168.1.1/24",
				"1.1.1.1/32",
				"10.0.6.1/16",
				"fd00::21/112",
			},
			expAllow: true,
		},
		"case1, not valid": {
			existingResources: nil,
			destSubnet: []string{
				"1.1.1.1",
			},
			expAllow: false,
		},
		"case2, not valid": {
			existingResources: nil,
			destSubnet: []string{
				"1.1.1.1999/24",
			},
			expAllow: false,
		},
		"case3, not valid": {
			existingResources: nil,
			destSubnet: []string{
				"---",
			},
			expAllow: false,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {

			policy := &v1.EgressPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "policy",
				},
				Spec: v1.EgressPolicySpec{
					EgressGatewayName: "test",
					EgressIP: v1.EgressIP{
						UseNodeIP: false,
						IPv4:      "",
						IPv6:      "",
					},
					AppliedTo: v1.AppliedTo{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "test"},
						},
					},
					DestSubnet: c.destSubnet,
				},
			}

			marshalledRequestObject, err := json.Marshal(policy)
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

func TestValidateEgressNode(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		newResource   *v1.EgressNode
		expAllow      bool
		expErrMessage string
	}{
		"all valid": {
			newResource: &v1.EgressNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node1",
				},
				Spec: v1.EgressNodeSpec{},
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
						Kind: "EgressNode",
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
