// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/egressgateway"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
)

const (
	EgressClusterInfo   = "EgressClusterInfo"
	EgressGateway       = "EgressGateway"
	EgressPolicy        = "EgressPolicy"
	EgressClusterPolicy = "EgressClusterPolicy"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// ValidateHook ValidateHook
func ValidateHook(client client.Client, cfg *config.Config) *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {

			switch req.Kind.Kind {
			case EgressClusterInfo:
				if req.Operation == v1.Delete {
					return webhook.Denied("EgressClusterInfo 'default' is not allowed to be deleted")
				}
				return webhook.Allowed("checked")
			case EgressGateway:
				return (&egressgateway.EgressGatewayWebhook{Client: client, Config: cfg}).EgressGatewayValidate(ctx, req)
			case EgressClusterPolicy:
				if req.Operation == v1.Delete {
					return webhook.Allowed("checked")
				}
				policy := new(egressv1.EgressClusterPolicy)
				err := json.Unmarshal(req.Object.Raw, policy)
				if err != nil {
					return webhook.Denied(fmt.Sprintf("json unmarshal EgressClusterPolicy with error: %v", err))
				}
				return validateSubnet(policy.Spec.DestSubnet)
			case EgressPolicy:
				if req.Operation == v1.Delete {
					return webhook.Allowed("checked")
				}

				egp := new(egressv1.EgressPolicy)
				err := json.Unmarshal(req.Object.Raw, egp)
				if err != nil {
					return webhook.Denied(fmt.Sprintf("json unmarshal EgressPolicy with error: %v", err))
				}

				if len(egp.Spec.EgressGatewayName) == 0 {
					return webhook.Denied("egressGatewayName cannot be empty")
				}

				if egp.Spec.EgressIP.UseNodeIP {
					if len(egp.Spec.EgressIP.IPv4) != 0 || len(egp.Spec.EgressIP.IPv6) != 0 {
						return webhook.Denied("useNodeIP cannot be used with egressIP.ipv4 or egressIP.ipv6 at the same time")
					}
				}

				if len(egp.Spec.AppliedTo.PodSelector.MatchLabels) != 0 && len(egp.Spec.AppliedTo.PodSubnet) != 0 {
					return webhook.Denied("podSelector and podSubnet cannot be used together")
				}

				if req.Operation == v1.Update {
					oldEgp := new(egressv1.EgressPolicy)
					err := json.Unmarshal(req.OldObject.Raw, oldEgp)
					if err != nil {
						return webhook.Denied(fmt.Sprintf("json unmarshal EgressPolicy with error: %v", err))
					}

					if egp.Spec.EgressGatewayName != oldEgp.Spec.EgressGatewayName {
						return webhook.Denied("the bound EgressGateway cannot be modified")
					}

					if egp.Spec.EgressIP.UseNodeIP != oldEgp.Spec.EgressIP.UseNodeIP {
						return webhook.Denied("the UseNodeIP field cannot be modified")
					}

					if egp.Spec.EgressIP.IPv4 != oldEgp.Spec.EgressIP.IPv4 {
						return webhook.Denied("the EgressIP.IPv4 field cannot be modified")
					}

					if egp.Spec.EgressIP.IPv6 != oldEgp.Spec.EgressIP.IPv6 {
						return webhook.Denied("the EgressIP.IPv6 field cannot be modified")
					}

					if egp.Spec.EgressIP.AllocatorPolicy != oldEgp.Spec.EgressIP.AllocatorPolicy {
						return webhook.Denied("the EgressIP.AllocatorPolicy field cannot be modified")
					}
				}

				if req.Operation == v1.Create {
					if cfg.FileConfig.EnableIPv4 || cfg.FileConfig.EnableIPv6 {
						if ok, err := checkEIP(client, ctx, *egp); !ok {
							return webhook.Denied(err.Error())
						}
					}
				}

				return validateSubnet(egp.Spec.DestSubnet)
			}

			return webhook.Allowed("checked")
		}),
	}
}

func checkEIP(client client.Client, ctx context.Context, egp egressv1.EgressPolicy) (bool, error) {

	eipIPV4 := egp.Spec.EgressIP.IPv4
	eipIPV6 := egp.Spec.EgressIP.IPv6

	if len(eipIPV4) == 0 && len(eipIPV6) == 0 {
		return true, nil
	}

	egwName := egp.Spec.EgressGatewayName
	egw := new(egressv1.EgressGateway)
	err := client.Get(ctx, types.NamespacedName{Name: egwName}, egw)
	if err != nil {
		if !errors.IsNotFound(err) {
			return false, fmt.Errorf("failed to get the EgressGateway: %v", err)
		}
	}

	if eipIPV4 == egw.Spec.Ippools.Ipv4DefaultEIP || eipIPV6 == egw.Spec.Ippools.Ipv6DefaultEIP {
		if eipIPV4 != egw.Spec.Ippools.Ipv4DefaultEIP || eipIPV6 != egw.Spec.Ippools.Ipv6DefaultEIP {
			return false, fmt.Errorf("%v egw Ipv4DefaultEIP=%v Ipv6DefaultEIP=%v, they can only be used together", egwName, egw.Spec.Ippools.Ipv4DefaultEIP, egw.Spec.Ippools.Ipv6DefaultEIP)
		}
	}

	eips := egressgateway.GetEipByIPV4(eipIPV4, *egw)
	if len(eips.IPv6) != 0 {
		if eipIPV6 != eips.IPv6 {
			return false, fmt.Errorf("%v cannot be used, when %v is used, %v must be used", eipIPV6, eipIPV4, eips.IPv6)
		}
	}

	eips = egressgateway.GetEipByIPV6(eipIPV6, *egw)
	if len(eips.IPv4) != 0 {
		if eipIPV4 != eips.IPv4 {
			return false, fmt.Errorf("%v cannot be used, when %v is used, %v must be used", eipIPV4, eipIPV6, eips.IPv4)
		}
	}

	return true, nil
}

// MutateHook MutateHook
func MutateHook(client client.Client, cfg *config.Config) *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {

			switch req.Kind.Kind {
			case EgressGateway:
				return (&egressgateway.EgressGatewayWebhook{Client: client, Config: cfg}).EgressGatewayMutate(ctx, req)
			case EgressPolicy:
				return egressPolicyMutateHook(client, cfg, ctx, req)
			case EgressClusterPolicy:
				return egressClusterPolicyMutateHook(client, cfg, ctx, req)
			}

			return webhook.Allowed("checked")
		}),
	}
}

// MutateHook egresspolicy
func egressPolicyMutateHook(client client.Client, cfg *config.Config, ctx context.Context, req webhook.AdmissionRequest) admission.Response {
	isPatch := false
	egp := new(egressv1.EgressPolicy)
	err := json.Unmarshal(req.Object.Raw, egp)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("json unmarshal EgressPolicy with error: %v", err))
	}

	reviewResponse := webhook.AdmissionResponse{}
	var patch []patchOperation

	if egp.Spec.EgressIP.IsEmpty() {
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  "/spec/egressIP",
			Value: egressv1.EgressIP{UseNodeIP: false, AllocatorPolicy: "default"},
		})
		isPatch = true
	}

	if isPatch {
		patchBytes, err := json.Marshal(patch)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("failed to set the default value of the EgressIP field: %v", err))
		}

		reviewResponse.Allowed = true
		reviewResponse.Patch = patchBytes
		pt := admissionv1.PatchTypeJSONPatch
		reviewResponse.PatchType = &pt

		return reviewResponse
	}

	return webhook.Allowed("checked")
}

// MutateHook egressclusterpolicy
func egressClusterPolicyMutateHook(client client.Client, cfg *config.Config, ctx context.Context, req webhook.AdmissionRequest) admission.Response {
	isPatch := false
	egcp := new(egressv1.EgressClusterPolicy)
	err := json.Unmarshal(req.Object.Raw, egcp)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("json unmarshal EgressPolicy with error: %v", err))
	}

	reviewResponse := webhook.AdmissionResponse{}
	var patch []patchOperation

	if egcp.Spec.EgressIP.IsEmpty() {
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  "/spec/egressIP",
			Value: egressv1.EgressIP{UseNodeIP: false, AllocatorPolicy: "default"},
		})
		isPatch = true
	}

	if isPatch {
		patchBytes, err := json.Marshal(patch)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("failed to set the default value of the EgressIP field: %v", err))
		}

		reviewResponse.Allowed = true
		reviewResponse.Patch = patchBytes
		pt := admissionv1.PatchTypeJSONPatch
		reviewResponse.PatchType = &pt

		return reviewResponse
	}

	return webhook.Allowed("checked")
}

func validateSubnet(subnet []string) webhook.AdmissionResponse {
	invalidList := make([]string, 0)
	for _, subnet := range subnet {
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
	return webhook.Allowed("checked")
}
