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
	"github.com/spidernet-io/egressgateway/pkg/constant"
	"github.com/spidernet-io/egressgateway/pkg/egressgateway"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
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
				return validateEgressClusterPolicy(ctx, client, req, cfg)
			case EgressPolicy:
				return validateEgressPolicy(ctx, client, req, cfg)
			}

			return webhook.Allowed("checked")
		}),
	}
}

func validateEgressPolicy(ctx context.Context, client client.Client, req webhook.AdmissionRequest, cfg *config.Config) webhook.AdmissionResponse {
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

	if len(egp.Spec.EgressIP.IPv4) != 0 && !isIPv4(egp.Spec.EgressIP.IPv4) {
		return webhook.Denied("invalid ipv4 format")
	}
	if len(egp.Spec.EgressIP.IPv6) != 0 && !isIPv6(egp.Spec.EgressIP.IPv6) {
		return webhook.Denied("invalid ipv6 format")
	}

	if egp.Spec.AppliedTo.PodSelector != nil && len(egp.Spec.AppliedTo.PodSelector.MatchLabels) != 0 && len(egp.Spec.AppliedTo.PodSubnet) != 0 {
		return webhook.Denied("podSelector and podSubnet cannot be used together")
	}

	// denied when both PodSelector and PodSubnet are empty
	if egp.Spec.AppliedTo.PodSubnet == nil || len(egp.Spec.AppliedTo.PodSubnet) == 0 {
		if egp.Spec.AppliedTo.PodSelector == nil || (len(egp.Spec.AppliedTo.PodSelector.MatchLabels) == 0 && len(egp.Spec.AppliedTo.PodSelector.MatchExpressions) == 0) {
			return webhook.Denied("invalid EgressPolicy, spec.appliedTo field requires at least one of spec.appliedTo.podSubnet, .spec.appliedTo.podSelector.matchLabels or .spec.appliedTo.podSelector.matchExpressions to be specified.")
		}
	}

	if req.Operation == v1.Update {
		oldEgp := new(egressv1.EgressPolicy)
		err := json.Unmarshal(req.OldObject.Raw, oldEgp)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("json unmarshal EgressPolicy with error: %v", err))
		}

		if egp.Spec.EgressGatewayName != oldEgp.Spec.EgressGatewayName {
			return webhook.Denied("'spec.EgressGatewayName' field is immutable")
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
			if ok, err := checkEIP(client, ctx, egp.Spec.EgressIP.IPv4, egp.Spec.EgressIP.IPv6, egp.Spec.EgressGatewayName); !ok {
				return webhook.Denied(err.Error())
			}

			if !egp.Spec.EgressIP.UseNodeIP {
				err := checkEGWIppools(client, cfg, ctx, egp.Spec.EgressGatewayName, egp.Spec.EgressIP.AllocatorPolicy)
				if err != nil {
					return webhook.Denied(fmt.Sprintf("when egp(%v) UseNodeIP is false, %v", egp.Name, err))
				}
			}
		}
	}

	return validateSubnet(egp.Spec.DestSubnet)
}

func validateEgressClusterPolicy(ctx context.Context, client client.Client, req webhook.AdmissionRequest, cfg *config.Config) webhook.AdmissionResponse {
	if req.Operation == v1.Delete {
		return webhook.Allowed("checked")
	}
	policy := new(egressv1.EgressClusterPolicy)
	err := json.Unmarshal(req.Object.Raw, policy)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("json unmarshal EgressClusterPolicy with error: %v", err))
	}

	if len(policy.Spec.EgressGatewayName) == 0 {
		return webhook.Denied("egressGatewayName cannot be empty")
	}

	if policy.Spec.EgressIP.UseNodeIP {
		if len(policy.Spec.EgressIP.IPv4) != 0 || len(policy.Spec.EgressIP.IPv6) != 0 {
			return webhook.Denied("useNodeIP cannot be used with egressIP.ipv4 or egressIP.ipv6 at the same time")
		}
	}

	if len(policy.Spec.EgressIP.IPv4) != 0 && !isIPv4(policy.Spec.EgressIP.IPv4) {
		return webhook.Denied("invalid ipv4 format")
	}
	if len(policy.Spec.EgressIP.IPv6) != 0 && !isIPv6(policy.Spec.EgressIP.IPv6) {
		return webhook.Denied("invalid ipv6 format")
	}

	if (policy.Spec.AppliedTo.PodSelector != nil && len(policy.Spec.AppliedTo.PodSelector.MatchLabels) != 0) &&
		(policy.Spec.AppliedTo.PodSubnet != nil && len(*policy.Spec.AppliedTo.PodSubnet) != 0) {
		return webhook.Denied("podSelector and podSubnet cannot be used together")
	}

	// denied when both PodSelector and PodSubnet are empty
	if policy.Spec.AppliedTo.PodSubnet == nil || len(*policy.Spec.AppliedTo.PodSubnet) == 0 {
		if policy.Spec.AppliedTo.PodSelector == nil || (len(policy.Spec.AppliedTo.PodSelector.MatchLabels) == 0 && len(policy.Spec.AppliedTo.PodSelector.MatchExpressions) == 0) {
			return webhook.Denied("invalid EgressClusterPolicy, spec.appliedTo field requires at least one of spec.appliedTo.podSubnet, .spec.appliedTo.podSelector.matchLabels or .spec.appliedTo.podSelector.matchExpressions to be specified.")
		}
	}

	if req.Operation == v1.Update {
		oldPolicy := new(egressv1.EgressClusterPolicy)
		err := json.Unmarshal(req.OldObject.Raw, oldPolicy)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("json unmarshal EgressClusterPolicy with error: %v", err))
		}

		if policy.Spec.EgressGatewayName != oldPolicy.Spec.EgressGatewayName {
			return webhook.Denied("'spec.EgressGatewayName' field is immutable")
		}

		if policy.Spec.EgressIP.UseNodeIP != oldPolicy.Spec.EgressIP.UseNodeIP {
			return webhook.Denied("the UseNodeIP field cannot be modified")
		}

		if policy.Spec.EgressIP.IPv4 != oldPolicy.Spec.EgressIP.IPv4 {
			return webhook.Denied("the EgressIP.IPv4 field cannot be modified")
		}

		if policy.Spec.EgressIP.IPv6 != oldPolicy.Spec.EgressIP.IPv6 {
			return webhook.Denied("the EgressIP.IPv6 field cannot be modified")
		}

		if policy.Spec.EgressIP.AllocatorPolicy != oldPolicy.Spec.EgressIP.AllocatorPolicy {
			return webhook.Denied("the EgressIP.AllocatorPolicy field cannot be modified")
		}
	}

	if req.Operation == v1.Create {
		if cfg.FileConfig.EnableIPv4 || cfg.FileConfig.EnableIPv6 {
			if ok, err := checkEIP(client, ctx, policy.Spec.EgressIP.IPv4, policy.Spec.EgressIP.IPv6, policy.Spec.EgressGatewayName); !ok {
				return webhook.Denied(err.Error())
			}

			if !policy.Spec.EgressIP.UseNodeIP {
				err := checkEGWIppools(client, cfg, ctx, policy.Spec.EgressGatewayName, policy.Spec.EgressIP.AllocatorPolicy)
				if err != nil {
					return webhook.Denied(fmt.Sprintf("when policy(%v) UseNodeIP is false, %v", policy.Name, err))
				}

			}
		}
	}

	return validateSubnet(policy.Spec.DestSubnet)
}

func checkEGWIppools(client client.Client, cfg *config.Config, ctx context.Context, name, allocatorPolicy string) error {

	egw := new(egressv1.EgressGateway)
	egw.Name = name
	err := client.Get(ctx, types.NamespacedName{Name: egw.Name}, egw)
	if err != nil {
		return fmt.Errorf("failed to obtain the EgressGateway: %v", err)
	}

	if cfg.FileConfig.EnableIPv4 && len(egw.Spec.Ippools.IPv4) == 0 {
		return fmt.Errorf("referenced egw(%v) spec.Ippools.IPv4 cannot be empty", egw.Name)
	}

	if cfg.FileConfig.EnableIPv6 && len(egw.Spec.Ippools.IPv6) == 0 {
		return fmt.Errorf("referenced egw(%v) spec.Ippools.IPv6 cannot be empty", egw.Name)
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
		return false, err
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

func countGatewayAvailableIP(egw *egressv1.EgressGateway) (int, int, error) {
	// check has free egress ip
	ipv4Ranges, err := ip.MergeIPRanges(constant.IPv4, egw.Spec.Ippools.IPv4)
	if err != nil {
		return 0, 0, err
	}
	ipv6Ranges, err := ip.MergeIPRanges(constant.IPv6, egw.Spec.Ippools.IPv6)
	if err != nil {
		return 0, 0, err
	}
	useIpv4s := make([]net.IP, 0)
	useIpv6s := make([]net.IP, 0)
	for _, node := range egw.Status.NodeList {
		for _, eip := range node.Eips {
			if len(eip.IPv4) != 0 {
				useIpv4s = append(useIpv4s, net.ParseIP(eip.IPv4))
			}
			if len(eip.IPv6) != 0 {
				useIpv6s = append(useIpv6s, net.ParseIP(eip.IPv6))
			}
		}
	}
	ipv4s, err := ip.ParseIPRanges(constant.IPv4, ipv4Ranges)
	if err != nil {
		return 0, 0, err
	}
	ipv6s, err := ip.ParseIPRanges(constant.IPv6, ipv6Ranges)
	if err != nil {
		return 0, 0, err
	}
	freeIpv4s := ip.IPsDiffSet(ipv4s, useIpv4s, false)
	freeIpv6s := ip.IPsDiffSet(ipv6s, useIpv6s, false)

	return len(freeIpv4s), len(freeIpv6s), nil
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

func isIPv4(ip string) bool {
	if netIP := net.ParseIP(ip); netIP != nil && netIP.To4() != nil {
		return true
	}
	return false
}

func isIPv6(ip string) bool {
	if netIP := net.ParseIP(ip); netIP != nil && netIP.To4() == nil && netIP.To16() != nil {
		return true
	}
	return false
}
