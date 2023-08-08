// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/spidernet-io/egressgateway/pkg/config"
	"github.com/spidernet-io/egressgateway/pkg/egressgateway"
	egressv1 "github.com/spidernet-io/egressgateway/pkg/k8s/apis/v1beta1"
)

// MutateHook MutateHook
func MutateHook(client client.Client, cfg *config.Config) *webhook.Admission {
	return &webhook.Admission{
		Handler: admission.HandlerFunc(func(ctx context.Context, req webhook.AdmissionRequest) webhook.AdmissionResponse {

			switch req.Kind.Kind {
			case EgressGateway:
				return (&egressgateway.EgressGatewayWebhook{Client: client, Config: cfg}).EgressGatewayMutate(ctx, req)
			case EgressPolicy:
				return mutateHookEgressPolicy(ctx, req, client)
			case EgressClusterPolicy:
				return mutateHookEgressClusterPolicy(ctx, req, client)
			}

			return webhook.Allowed("checked")
		}),
	}
}

func mutateHookEgressPolicy(ctx context.Context, req webhook.AdmissionRequest, cli client.Client) webhook.AdmissionResponse {
	policy := new(egressv1.EgressPolicy)
	err := json.Unmarshal(req.Object.Raw, policy)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("json unmarshal EgressPolicy with error: %v", err))
	}

	patchList := make([]jsonpatch.JsonPatchOperation, 0)

	if policy.Spec.EgressGatewayName == "" {
		ns := &corev1.Namespace{}
		err := cli.Get(ctx, types.NamespacedName{Name: policy.Namespace}, ns)
		if err != nil {
			return webhook.Denied(fmt.Sprintf("failed to get EgressPolicy namespaces: %v", err))
		}
		egw, ok := ns.Labels[egressv1.LabelNamespaceEgressGatewayDefault]
		if !ok || egw == "" {
			if policy.Spec.EgressGatewayName == "" {
				p, err := getGlobalDefaultEgwPatch(ctx, cli)
				if err != nil {
					return webhook.Denied(err.Error())
				}
				if p != nil {
					patchList = append(patchList, *p)
				}
			}
		} else {
			patchList = append(patchList, jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      "/spec/egressGatewayName",
				Value:     egw,
			})
		}
	}

	if policy.Spec.EgressIP.IsEmpty() {
		patchList = append(patchList, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/spec/egressIP",
			Value:     egressv1.EgressIP{UseNodeIP: false, AllocatorPolicy: "default"},
		})
	}

	if len(patchList) > 0 {
		return webhook.Patched("patched", patchList...)
	}

	return webhook.Allowed("skipped")
}

func getGlobalDefaultEgwPatch(ctx context.Context, cli client.Client) (*jsonpatch.JsonPatchOperation, error) {
	egwList := &egressv1.EgressGatewayList{}
	err := cli.List(ctx, egwList)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check global default EgressGateway: %v", err)
	}

	var name string
	for _, item := range egwList.Items {
		if item.Spec.ClusterDefault {
			name = item.Name
		}
	}
	if name == "" {
		return nil, nil
	}

	patch := &jsonpatch.JsonPatchOperation{
		Operation: "add",
		Path:      "/spec/egressGatewayName",
		Value:     name,
	}
	return patch, nil
}

func mutateHookEgressClusterPolicy(ctx context.Context, req webhook.AdmissionRequest, cli client.Client) webhook.AdmissionResponse {
	policy := new(egressv1.EgressClusterPolicy)
	err := json.Unmarshal(req.Object.Raw, policy)
	if err != nil {
		return webhook.Denied(fmt.Sprintf("json unmarshal EgressPolicy with error: %v", err))
	}

	patchList := make([]jsonpatch.JsonPatchOperation, 0)

	if policy.Spec.EgressGatewayName == "" {
		p, err := getGlobalDefaultEgwPatch(ctx, cli)
		if err != nil {
			return webhook.Denied(err.Error())
		}
		if p != nil {
			patchList = append(patchList, *p)
		}
	}

	if policy.Spec.EgressIP.IsEmpty() {
		patchList = append(patchList, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/spec/egressIP",
			Value:     egressv1.EgressIP{UseNodeIP: false, AllocatorPolicy: "default"},
		})
	}

	if len(patchList) > 0 {
		return webhook.Patched("patched", patchList...)
	}

	return webhook.Allowed("skipped")
}
