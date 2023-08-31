// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/schema"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

func TestMutateHook(t *testing.T) {
	ctx := context.TODO()

	marshalledRequestObject, err := json.Marshal(&v1beta1.EgressGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "eg-test",
		},
		Spec: v1beta1.EgressGatewaySpec{
			Ippools: v1beta1.Ippools{
				IPv4: []string{"1.1.1.1x"},
			},
		},
	})
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

	Mutator := MutateHook(cli, conf)
	_ = Mutator.Handle(ctx, admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Name: "eg-test",
			Kind: metav1.GroupVersionKind{
				Kind: "EgressGateway",
			},
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: marshalledRequestObject,
			},
		},
	})

	_ = Mutator.Handle(ctx, admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Name: "eg-test",
			Kind: metav1.GroupVersionKind{
				Kind: "EgressPolicy",
			},
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: marshalledRequestObject,
			},
		},
	})

	_ = Mutator.Handle(ctx, admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Name: "eg-test",
			Kind: metav1.GroupVersionKind{
				Kind: "EgressClusterPolicy",
			},
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: marshalledRequestObject,
			},
		},
	})
}
