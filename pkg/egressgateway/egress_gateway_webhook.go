// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/spidernet-io/egressgateway/pkg/utils/ip"

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/constant"
	egress "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
)

type EgressGatewayWebhook struct {
	Client client.Client
	Config *config.Config
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

var egressGatewayFinalizers = "egressgateway.spidernet.io/egressgateway"

func (egw *EgressGatewayWebhook) EgressGatewayValidate(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	if req.Operation == v1.Delete {
		return webhook.Allowed("checked")
	}

	newEg := new(egress.EgressGateway)
	err := json.Unmarshal(req.Object.Raw, newEg)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("json unmarshal EgressGateway with error: %v", err))
	}

	if newEg.Spec.NodeSelector.Selector == nil ||
		(len(newEg.Spec.NodeSelector.Selector.MatchLabels) == 0 && len(newEg.Spec.NodeSelector.Selector.MatchExpressions) == 0) {
		return webhook.Denied("The field spec.nodeSelector.selector is not set")
	}

	if egw.Config.FileConfig.EnableIPv4 && !egw.Config.FileConfig.EnableIPv6 {
		if len(newEg.Spec.Ippools.IPv6) != 0 {
			return webhook.Denied("Please do not configure spec.ippools.ipv6, as the current installation settings have not enabled IPv6")
		}
	}
	if !egw.Config.FileConfig.EnableIPv4 && egw.Config.FileConfig.EnableIPv6 {
		if len(newEg.Spec.Ippools.IPv4) != 0 {
			return webhook.Denied("Please do not configure spec.ippools.ipv4, as the current installation settings have not enabled IPv4")
		}
	}

	if newEg.Spec.ClusterDefault {
		egwList := new(egress.EgressGatewayList)
		for _, item := range egwList.Items {
			if item.Spec.ClusterDefault && item.Name != newEg.Name {
				return webhook.Denied(fmt.Sprintf("A cluster can only have one default gateway, default gateway: %s.", item.Name))
			}
		}
	}

	// Checking the number of IPV4 and IPV6 addresses
	var ipv4s, ipv6s []net.IP
	ipv4Ranges, err := ip.MergeIPRanges(constant.IPv4, newEg.Spec.Ippools.IPv4)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("Failed to check IP: %v", err))
	}

	ipv6Ranges, _ := ip.MergeIPRanges(constant.IPv6, newEg.Spec.Ippools.IPv6)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("Failed to check IP: %v", err))
	}

	if egw.Config.FileConfig.EnableIPv4 {
		ipv4s, err = ip.ParseIPRanges(constant.IPv4, ipv4Ranges)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("Failed to check IP: %v", err))
		}

	}

	if egw.Config.FileConfig.EnableIPv6 {
		ipv6s, err = ip.ParseIPRanges(constant.IPv6, ipv6Ranges)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("Failed to check IP: %v", err))
		}

	}

	if egw.Config.FileConfig.EnableIPv4 && egw.Config.FileConfig.EnableIPv6 {
		if len(ipv4s) != len(ipv6s) {
			return webhook.Denied("The number of ipv4 and ipv6 is not equal")
		}
	}

	eg := new(egress.EgressGateway)
	err = egw.Client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, eg)
	if err != nil {
		if !errors.IsNotFound(err) {
			return webhook.Denied(fmt.Sprintf("failed to obtain the EgressGateway: %v", err))
		}
	}

	// Check whether the IP address to be deleted has been allocated
	for _, item := range eg.Status.NodeList {
		for _, eip := range item.Eips {
			// skip the cases of using useNodeIP
			if eip.IPv4 == "" && eip.IPv6 == "" {
				continue
			}

			result, err := ip.IsIPIncludedRange(constant.IPv4, eip.IPv4, ipv4Ranges)
			if err != nil {
				return webhook.Denied(fmt.Sprintf("Failed to check IP: %v", err))
			}

			if !result {
				return webhook.Denied(fmt.Sprintf("%v has been allocated and cannot be deleted", eip.IPv4))
			}
		}
	}

	// Check the defaultEIP
	if len(newEg.Spec.Ippools.Ipv4DefaultEIP) != 0 {
		result, err := ip.IsIPIncludedRange(constant.IPv4, newEg.Spec.Ippools.Ipv4DefaultEIP, ipv4Ranges)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("Failed to check Ipv4DefaultEIP: %v", err))
		}
		if !result {
			return webhook.Denied(fmt.Sprintf("%v is not covered by Ippools", newEg.Spec.Ippools.Ipv4DefaultEIP))
		}
	}

	if len(newEg.Spec.Ippools.Ipv6DefaultEIP) != 0 {
		result, err := ip.IsIPIncludedRange(constant.IPv6, newEg.Spec.Ippools.Ipv6DefaultEIP, ipv6Ranges)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("Failed to check Ipv6DefaultEIP: %v", err))
		}
		if !result {
			return webhook.Denied(fmt.Sprintf("%v is not covered by Ippools", newEg.Spec.Ippools.Ipv6DefaultEIP))
		}
	}

	return webhook.Allowed("checked")
}

func (egw *EgressGatewayWebhook) EgressGatewayMutate(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	rander := rand.New(rand.NewSource(time.Now().UnixNano()))
	eg := new(egress.EgressGateway)
	err := json.Unmarshal(req.Object.Raw, eg)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("json unmarshal EgressGateway with error: %v", err))
	}

	reviewResponse := webhook.AdmissionResponse{}
	var patchList []patchOperation

	// patch egress gateway default eip
	if egw.Config.FileConfig.EnableIPv4 {
		if len(eg.Spec.Ippools.Ipv4DefaultEIP) == 0 && len(eg.Spec.Ippools.IPv4) != 0 {
			ipv4Ranges, err := ip.MergeIPRanges(constant.IPv4, eg.Spec.Ippools.IPv4)
			if err != nil {
				return webhook.Denied(fmt.Sprintf("ippools.ipv4 format error: %v", err))
			}

			ipv4s, _ := ip.ParseIPRanges(constant.IPv4, ipv4Ranges)
			if len(ipv4s) != 0 {
				patchList = append(patchList, patchOperation{
					Op:    "add",
					Path:  "/spec/ippools/ipv4DefaultEIP",
					Value: ipv4s[rander.Intn(len(ipv4s))].String(),
				})
			}

		}

	}

	if egw.Config.FileConfig.EnableIPv6 {
		if len(eg.Spec.Ippools.Ipv6DefaultEIP) == 0 && len(eg.Spec.Ippools.IPv6) != 0 {
			ipv6Ranges, err := ip.MergeIPRanges(constant.IPv6, eg.Spec.Ippools.IPv6)
			if err != nil {
				return webhook.Denied(fmt.Sprintf("ippools.ipv6 format error: %v", err))
			}

			ipv6s, _ := ip.ParseIPRanges(constant.IPv6, ipv6Ranges)
			if len(ipv6s) != 0 {
				patchList = append(patchList, patchOperation{
					Op:    "add",
					Path:  "/spec/ippools/ipv6DefaultEIP",
					Value: ipv6s[rander.Intn(len(ipv6s))].String(),
				})
			}
		}
	}

	// patch egress gateway finalizer
	patch := getEgressGatewayFinalizerPatch(req, []string{egressGatewayFinalizers})
	if patch != nil {
		patchList = append(patchList, *patch)
	}

	if len(patchList) > 0 {
		patchBytes, err := json.Marshal(patchList)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("failed to Marshal patchList.: %v", err))
		}

		reviewResponse.Allowed = true
		reviewResponse.Patch = patchBytes
		pt := admissionv1.PatchTypeJSONPatch
		reviewResponse.PatchType = &pt

		return reviewResponse
	}

	return webhook.Allowed("checked")
}

func getEgressGatewayFinalizerPatch(req webhook.AdmissionRequest, finalizer []string) *patchOperation {
	if req.Operation == v1.Create {
		return &patchOperation{
			Op:    "add",
			Path:  "/metadata/finalizers",
			Value: finalizer,
		}
	}
	return nil
}
