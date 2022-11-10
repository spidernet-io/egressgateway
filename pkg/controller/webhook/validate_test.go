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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
)

func TestValidateEgressGatewayNode(t *testing.T) {
	ctx := context.Background()

	cases := map[string]struct {
		existingResources []runtime.Object
		newResource       *egressv1.EgressGatewayNode
		expAllow          bool
		expErrMessage     string
	}{
		"no duplicates, valid": {
			existingResources: nil,
			newResource: &egressv1.EgressGatewayNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: "EgressGatewayNode",
				},
				Spec: egressv1.EgressGatewayNodeSpec{},
			},
			expAllow: true,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			marshalledRequestObject, err := json.Marshal(c.newResource)
			assert.NoError(t, err)

			validator := ValidateHook()
			resp := validator.Handle(ctx, admission.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Name:      c.newResource.Name,
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
