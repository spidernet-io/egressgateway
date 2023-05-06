// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package egressgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/constant"
	egress "github.com/spidernet-io/egressgateway/pkg/k8s/apis/egressgateway.spidernet.io/v1beta1"
	"github.com/spidernet-io/egressgateway/pkg/utils"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type EgressGatewayWebhook struct {
	Client client.Client
	Config *config.Config
}

func (egw *EgressGatewayWebhook) EgressGatewayValidate(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {
	newEg := new(egress.EgressGateway)
	err := json.Unmarshal(req.Object.Raw, newEg)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("json unmarshal EgressGateway with error: %v", err))
	}

	// Checking the number of IPV4 and IPV6 addresses
	var ipv4s, ipv6s []net.IP
	if egw.Config.FileConfig.EnableIPv4 {
		ipv4s, err = utils.ParseIPRanges(constant.IPv4, newEg.Spec.Ranges.IPv4)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("Failed to check IP: %v", err))
		}
	}
	if egw.Config.FileConfig.EnableIPv6 {
		ipv6s, err = utils.ParseIPRanges(constant.IPv6, newEg.Spec.Ranges.IPv4)
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

	// Check whether the deleted EgressGateway is referenced
	if req.Operation == v1.Delete {
		for _, item := range eg.Status.NodeList {
			for _, eip := range item.Eips {
				if len(eip.Policies) != 0 {
					return webhook.Denied(fmt.Sprintf("Do not delete %v:%v because it is already referenced by EgressGatewayPolicy", req.Namespace, req.Name))
				}
			}
		}
		return webhook.Allowed("checked")
	}

	// Check whether the IP address to be deleted has been allocated
	eips, _ := utils.MergeIPRanges(constant.IPv4, newEg.Spec.Ranges.IPv4)
	for _, item := range eg.Status.NodeList {
		for _, eip := range item.Eips {
			result, err := utils.IsIPIncludedRange(constant.IPv4, eip.IPv4, eips)
			if err != nil {
				return webhook.Denied(fmt.Sprintf("Failed to check IP: %v", err))
			}

			if !result {
				return webhook.Denied(fmt.Sprintf("%v has been allocated and cannot be deleted", eip.IPv4))
			}
		}
	}

	return webhook.Allowed("checked")
}
