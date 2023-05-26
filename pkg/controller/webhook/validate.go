// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	v1 "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/egressgateway"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1"
)

const EgressClusterInfo = "EgressClusterInfo"

// ValidateHook ValidateHook
func ValidateHook(client client.Client, cfg *config.Config) *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
			if req.Operation == v1.Delete {
				switch req.Kind.Kind {
				case EgressClusterInfo:
					return webhook.Denied("EgressClusterInfo 'default' is not allowed to be deleted")
				}
				return webhook.Allowed("checked")
			}

			switch req.Kind.Kind {
			case "EgressGateway":
				return (&egressgateway.EgressGatewayWebhook{Client: client, Config: cfg}).EgressGatewayValidate(ctx, req)
			case "EgressPolicy":
				policy := new(egressv1.EgressGatewayPolicy)
				err := json.Unmarshal(req.Object.Raw, policy)
				if err != nil {
					return webhook.Denied(fmt.Sprintf("json unmarshal EgressGatewayPolicy with error: %v", err))
				}
				invalidList := make([]string, 0)
				for _, subnet := range policy.Spec.DestSubnet {
					ip, _, err := net.ParseCIDR(subnet)
					if err != nil {
						invalidList = append(invalidList, subnet)
						continue
					}
					if ip.To4() == nil && ip.To16() == nil {
						invalidList = append(invalidList, subnet)
					}
				}
				if len(invalidList) > 0 {
					return webhook.Denied(fmt.Sprintf("invalid destSubnet list: %v", invalidList))
				}
			}

			return webhook.Allowed("checked")
		}),
	}
}
