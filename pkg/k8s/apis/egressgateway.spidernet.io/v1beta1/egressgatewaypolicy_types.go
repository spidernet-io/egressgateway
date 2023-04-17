// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EgressGatewayPolicyList contains a list of egress gateway policies
// +kubebuilder:object:root=true
type EgressGatewayPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressGatewayPolicy `json:"items"`
}

// EgressGatewayPolicy represents a single egress gateway policy
// +kubebuilder:resource:categories={egressgatewaypolicy},path="egressgatewaypolicies",singular="egressgatewaypolicy",scope="Cluster",shortName={epo}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +genclient:nonNamespaced
type EgressGatewayPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec EgressGatewayPolicySpec `json:"spec,omitempty"`
}

type EgressGatewayPolicySpec struct {
	// +kubebuilder:validation:Optional
	EgressGatewayName string `json:"egressGatewayName,omitempty"`
	// +kubebuilder:validation:Optional
	EgressIP EgressIP `json:"egressIP,omitempty"`
	// +kubebuilder:validation:Required
	AppliedTo AppliedTo `json:"appliedTo"`
	// +kubebuilder:validation:Optional
	DestSubnet []string `json:"destSubnet"`
}

type EgressIP struct {
	IPv4      string `json:"ipv4,omitempty"`
	IPv6      string `json:"ipv6,omitempty"`
	UseNodeIP bool   `json:"useNodeIP,omitempty"`
}

type AppliedTo struct {
	// +kubebuilder:validation:Optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
	// +kubebuilder:validation:Optional
	PodSubnet *[]string `json:"podSubnet,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EgressGatewayPolicy{}, &EgressGatewayPolicyList{})
}
