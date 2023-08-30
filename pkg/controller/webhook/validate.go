// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

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
				egcp := new(egressv1.EgressClusterPolicy)
				err := json.Unmarshal(req.Object.Raw, egcp)
				if err != nil {
					return webhook.Denied(fmt.Sprintf("json unmarshal EgressClusterPolicy with error: %v", err))
				}

				if len(egcp.Spec.EgressGatewayName) == 0 {
					return webhook.Denied("egressGatewayName cannot be empty")
				}

				if egcp.Spec.EgressIP.UseNodeIP {
					if len(egcp.Spec.EgressIP.IPv4) != 0 || len(egcp.Spec.EgressIP.IPv6) != 0 {
						return webhook.Denied("useNodeIP cannot be used with egressIP.ipv4 or egressIP.ipv6 at the same time")
					}
				}

				if len(egcp.Spec.AppliedTo.PodSelector.MatchLabels) != 0 && len(*egcp.Spec.AppliedTo.PodSubnet) != 0 {
					return webhook.Denied("podSelector and podSubnet cannot be used together")
				}

				if req.Operation == v1.Update {
					oldEgcp := new(egressv1.EgressClusterPolicy)
					err := json.Unmarshal(req.OldObject.Raw, oldEgcp)
					if err != nil {
						return webhook.Denied(fmt.Sprintf("json unmarshal EgressClusterPolicy with error: %v", err))
					}

					if egcp.Spec.EgressGatewayName != oldEgcp.Spec.EgressGatewayName {
						return webhook.Denied("the bound EgressClusterPolicy cannot be modified")
					}

					if egcp.Spec.EgressIP.UseNodeIP != oldEgcp.Spec.EgressIP.UseNodeIP {
						return webhook.Denied("the UseNodeIP field cannot be modified")
					}

					if egcp.Spec.EgressIP.IPv4 != oldEgcp.Spec.EgressIP.IPv4 {
						return webhook.Denied("the EgressIP.IPv4 field cannot be modified")
					}

					if egcp.Spec.EgressIP.IPv6 != oldEgcp.Spec.EgressIP.IPv6 {
						return webhook.Denied("the EgressIP.IPv6 field cannot be modified")
					}

					if egcp.Spec.EgressIP.AllocatorPolicy != oldEgcp.Spec.EgressIP.AllocatorPolicy {
						return webhook.Denied("the EgressIP.AllocatorPolicy field cannot be modified")
					}
				}

				if req.Operation == v1.Create {
					if cfg.FileConfig.EnableIPv4 || cfg.FileConfig.EnableIPv6 {
						if ok, err := checkEIP(client, ctx, egcp.Spec.EgressIP.IPv4, egcp.Spec.EgressIP.IPv6, egcp.Name); !ok {
							return webhook.Denied(err.Error())
						}

						if !egcp.Spec.EgressIP.UseNodeIP {
							err := checkEGWIppools(client, cfg, ctx, egcp.Spec.EgressGatewayName)
							if err != nil {
								return webhook.Denied(fmt.Sprintf("when egcp(%v) UseNodeIP is false, %v", egcp.Name, err))
							}
						}
					}
				}

				return validateSubnet(egcp.Spec.DestSubnet)
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
						if ok, err := checkEIP(client, ctx, egp.Spec.EgressIP.IPv4, egp.Spec.EgressIP.IPv6, egp.Name); !ok {
							return webhook.Denied(err.Error())
						}

						if !egp.Spec.EgressIP.UseNodeIP {
							err := checkEGWIppools(client, cfg, ctx, egp.Spec.EgressGatewayName)
							if err != nil {
								return webhook.Denied(fmt.Sprintf("when egp(%v) UseNodeIP is false, %v", egp.Name, err))
							}
						}
					}
				}

				return validateSubnet(egp.Spec.DestSubnet)
			}

			return webhook.Allowed("checked")
		}),
	}
}

func checkEGWIppools(client client.Client, cfg *config.Config, ctx context.Context, name string) error {

	egw := new(egressv1.EgressGateway)
	egw.Name = name
	err := client.Get(ctx, types.NamespacedName{Name: egw.Name}, egw)
	if err != nil {
		return fmt.Errorf("failed to obtain the EgressGateway: %v", err)
	}

	if cfg.FileConfig.EnableIPv4 && len(egw.Spec.Ippools.IPv4) == 0 {
		return fmt.Errorf("referenced egw(%v) pec.Ippools.IPv4 cannot be empty", egw.Name)
	}

	if cfg.FileConfig.EnableIPv6 && len(egw.Spec.Ippools.IPv6) == 0 {
		return fmt.Errorf("referenced egw(%v) pec.Ippools.IPv6 cannot be empty", egw.Name)
	}

	return nil
}

func checkEIP(client client.Client, ctx context.Context, ipv4, ipv6, egwName string) (bool, error) {

	eipIPV4 := ipv4
	eipIPV6 := ipv6

	if len(eipIPV4) == 0 && len(eipIPV6) == 0 {
		return true, nil
	}

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
