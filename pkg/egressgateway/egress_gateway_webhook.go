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

	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/constant"
	egress "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils/ip"
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
		err := egw.Client.List(ctx, egwList)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("Check for duplicate EgressGateway, get EgressGatewayList: %v", err))
		}
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

	ipv6Ranges, err := ip.MergeIPRanges(constant.IPv6, newEg.Spec.Ippools.IPv6)
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
		// allowed single ipv4 or ipv6 when both ipv4 and ipv6 are enabled
		if len(newEg.Spec.Ippools.IPv4) > 0 && len(newEg.Spec.Ippools.IPv6) > 0 && len(ipv4s) != len(ipv6s) {
			return webhook.Denied("The number of ipv4 and ipv6 is not equal")
		}
	}

	// Check the defaultEIP
	if len(newEg.Spec.Ippools.Ipv4DefaultEIP) != 0 {
		result, err := ip.IsIPIncludedRange(constant.IPv4, newEg.Spec.Ippools.Ipv4DefaultEIP, ipv4Ranges)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("Failed to check ipv4DefaultEIP: %v", err))
		}
		if !result {
			return webhook.Denied(fmt.Sprintf("%v is not covered by IPPools", newEg.Spec.Ippools.Ipv4DefaultEIP))
		}
	}

	if len(newEg.Spec.Ippools.Ipv6DefaultEIP) != 0 {
		result, err := ip.IsIPIncludedRange(constant.IPv6, newEg.Spec.Ippools.Ipv6DefaultEIP, ipv6Ranges)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("Failed to check ipv6DefaultEIP: %v", err))
		}
		if !result {
			return webhook.Denied(fmt.Sprintf("%v is not covered by Ippools", newEg.Spec.Ippools.Ipv6DefaultEIP))
		}
	}

	// check if the current egw ip pool is duplicated by other egw ip pools
	egwList := &egress.EgressGatewayList{}
	err = egw.Client.List(ctx, egwList)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("Failed to get EgressGatewayList: %v", err))
	}
	clusterMap, err := buildClusterIPMap(egwList, newEg.Name)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("Failed to build cluster EgressGateway IP map: %v", err))
	}
	err = checkDupIP(ipv4s, ipv6s, clusterMap)
	if err != nil {
		return webhook.Denied(err.Error())
	}

	// only for update
	if req.Operation == v1.Update {
		oldEgressGateway := new(egress.EgressGateway)
		err = egw.Client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, oldEgressGateway)
		if err != nil {
			if !errors.IsNotFound(err) {
				return webhook.Denied(fmt.Sprintf("failed to obtain the EgressGateway: %v", err))
			}
		}

		// it should be denied when the single IPv4 or IPv6 is updated to the other type
		if len(oldEgressGateway.Spec.Ippools.IPv4) == 0 && len(newEg.Spec.Ippools.IPv4) > 0 {
			return webhook.Denied("the 'spec.Ippools.IPv4' field cannot to be modified when it is empty")
		}
		if len(oldEgressGateway.Spec.Ippools.IPv6) == 0 && len(newEg.Spec.Ippools.IPv6) > 0 {
			return webhook.Denied("the 'spec.Ippools.IPv6' field cannot to be modified when it is empty")
		}

		// check if the IP to be deleted is already assigned
		for _, item := range oldEgressGateway.Status.NodeList {
			for _, eip := range item.Eips {
				// skip the cases of using useNodeIP
				if eip.IPv4 == "" && eip.IPv6 == "" {
					continue
				}

				if eip.IPv4 != "" {
					result, err := ip.IsIPIncludedRange(constant.IPv4, eip.IPv4, ipv4Ranges)
					if err != nil {
						return webhook.Denied(fmt.Sprintf("Failed to check IPv4: %v", err))
					}
					if !result {
						return webhook.Denied(fmt.Sprintf("%v has been allocated and cannot be deleted", eip.IPv4))
					}
				}
				if eip.IPv6 != "" {
					result, err := ip.IsIPIncludedRange(constant.IPv6, eip.IPv6, ipv6Ranges)
					if err != nil {
						return webhook.Denied(fmt.Sprintf("Failed to check IPv6: %v", err))
					}
					if !result {
						return webhook.Denied(fmt.Sprintf("%v has been allocated and cannot be deleted", eip.IPv6))
					}
				}
			}
		}
	}

	return webhook.Allowed("checked")
}

func buildClusterIPMap(egwList *egress.EgressGatewayList, skipName string) (map[string]map[string]struct{}, error) {
	res := make(map[string]map[string]struct{})
	for _, item := range egwList.Items {
		if item.Name == skipName {
			continue
		}

		var ipv4s, ipv6s []net.IP
		ipv4Ranges, err := ip.MergeIPRanges(constant.IPv4, item.Spec.Ippools.IPv4)
		if err != nil {
			return nil, err
		}
		ipv6Ranges, err := ip.MergeIPRanges(constant.IPv6, item.Spec.Ippools.IPv6)
		if err != nil {
			return nil, err
		}
		ipv4s, err = ip.ParseIPRanges(constant.IPv4, ipv4Ranges)
		if err != nil {
			return nil, err
		}
		ipv6s, err = ip.ParseIPRanges(constant.IPv6, ipv6Ranges)
		if err != nil {
			return nil, err
		}
		m := make(map[string]struct{})
		for _, v := range ipv4s {
			m[v.String()] = struct{}{}
		}
		for _, v := range ipv6s {
			m[v.String()] = struct{}{}
		}
		res[item.Name] = m
	}
	return res, nil
}

func checkDupIP(currentIPv4List, currentIPv6List []net.IP, clusterIPMap map[string]map[string]struct{}) error {
	for _, v := range currentIPv4List {
		addr := v.String()
		for gwName, gw := range clusterIPMap {
			if _, ok := gw[addr]; ok {
				return fmt.Errorf("find duplicate IPv4 %s in EgressGateway %s", v.String(), gwName)
			}
		}
	}
	for _, v := range currentIPv6List {
		addr := v.String()
		for gwName, gw := range clusterIPMap {
			if _, ok := gw[addr]; ok {
				return fmt.Errorf("find duplicate IPv6 %s in EgressGateway %s", v.String(), gwName)
			}
		}
	}
	return nil
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
