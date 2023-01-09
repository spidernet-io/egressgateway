// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"

	v1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// MutatingHook MutatingHook
func MutatingHook(client client.Client) *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
			return webhook.Allowed("allowed")
		}),
	}
}

// nolint
func egressGatewayWebhook(ctx context.Context, req *webhook.AdmissionRequest, client client.Client) webhook.AdmissionResponse {
	return webhook.AdmissionResponse{
		Patches:           nil,
		AdmissionResponse: v1.AdmissionResponse{},
	}
}
