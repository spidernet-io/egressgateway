// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/schema"
)

func TestMutateHook(t *testing.T) {
	ctx := context.TODO()

	cases := map[string]struct {
		kind      metav1.GroupVersionKind
		operation admissionv1.Operation
		reqOjb    any
		initObjs  []client.Object
	}{
		"mutate egressgateway": {
			kind: metav1.GroupVersionKind{
				Kind: "EgressGateway",
			},
			operation: admissionv1.Create,
			reqOjb: &v1beta1.EgressGateway{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eg-test",
				},
				Spec: v1beta1.EgressGatewaySpec{
					Ippools: v1beta1.Ippools{
						IPv4: []string{"1.1.1.1x"},
					},
				},
			},
		},
		"mutate egresspolicy without default gateway in ns label": {
			kind: metav1.GroupVersionKind{
				Kind: "EgressPolicy",
			},
			operation: admissionv1.Create,
			reqOjb: &v1beta1.EgressPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "eg-test",
					Namespace: "default",
				},
				Spec: v1beta1.EgressPolicySpec{
					EgressGatewayName: "",
					EgressIP:          v1beta1.EgressIP{},
				},
			},
			initObjs: []client.Object{
				&v1beta1.EgressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "egw-default",
					},
					Spec: v1beta1.EgressGatewaySpec{
						Ippools: v1beta1.Ippools{
							IPv4: []string{"10.10.0.2-10.10.0.12"},
						},
						ClusterDefault: true,
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				},
			},
		},
		"mutate egresspolicy without default gateway in ns label and have no default gateway obj": {
			kind: metav1.GroupVersionKind{
				Kind: "EgressPolicy",
			},
			operation: admissionv1.Create,
			reqOjb: &v1beta1.EgressPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "eg-test",
					Namespace: "default",
				},
				Spec: v1beta1.EgressPolicySpec{
					EgressGatewayName: "",
					EgressIP:          v1beta1.EgressIP{},
				},
			},
			initObjs: []client.Object{
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				},
			},
		},
		"mutate egresspolicy exist default gateway in ns label": {
			kind: metav1.GroupVersionKind{
				Kind: "EgressPolicy",
			},
			operation: admissionv1.Create,
			reqOjb: &v1beta1.EgressPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "eg-test",
					Namespace: "default",
				},
				Spec: v1beta1.EgressPolicySpec{
					EgressGatewayName: "",
					EgressIP:          v1beta1.EgressIP{},
				},
			},
			initObjs: []client.Object{
				&v1beta1.EgressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "egw-default",
					},
					Spec: v1beta1.EgressGatewaySpec{
						Ippools: v1beta1.Ippools{
							IPv4: []string{"10.10.0.2-10.10.0.12"},
						},
						ClusterDefault: true,
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "default",
						Labels: map[string]string{v1beta1.LabelNamespaceEgressGatewayDefault: "egw-default"},
					},
				},
			},
		},
		"mutate egressClusterpolicy": {
			kind: metav1.GroupVersionKind{
				Kind: "EgressClusterPolicy",
			},
			operation: admissionv1.Create,
			reqOjb: &v1beta1.EgressClusterPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "eg-test",
				},
				Spec: v1beta1.EgressClusterPolicySpec{
					EgressGatewayName: "",
					EgressIP:          v1beta1.EgressIP{},
				},
			},
			initObjs: []client.Object{
				&v1beta1.EgressGateway{
					ObjectMeta: metav1.ObjectMeta{
						Name: "egw-default",
					},
					Spec: v1beta1.EgressGatewaySpec{
						Ippools: v1beta1.Ippools{
							IPv4: []string{"10.10.0.2-10.10.0.12"},
						},
						ClusterDefault: true,
					},
				},
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			marshalledRequestObject, err := json.Marshal(v.reqOjb)
			assert.NoError(t, err)

			builder := fake.NewClientBuilder()
			builder.WithScheme(schema.GetScheme())
			if len(v.initObjs) != 0 {
				builder.WithObjects(v.initObjs...)
			}
			cli := builder.Build()
			conf := &config.Config{
				FileConfig: config.FileConfig{
					EnableIPv4: true,
					EnableIPv6: false,
				},
			}

			Mutator := MutateHook(cli, conf)
			_ = Mutator.Handle(ctx, admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Kind:      v.kind,
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: marshalledRequestObject,
					},
				},
			})
		})
	}
}
