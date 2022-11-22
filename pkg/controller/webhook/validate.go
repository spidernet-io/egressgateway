// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// ValidateHook ValidateHook
func ValidateHook() *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
			fmt.Println("webhook debug")
			if req.Kind.Kind != "" {
				return webhook.Allowed("skip check")
			}

			return webhook.Allowed("checked")
		}),
	}
}
