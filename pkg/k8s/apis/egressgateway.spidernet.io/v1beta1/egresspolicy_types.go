// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EgressPolicyList contains a list of egress gateway policies
// +kubebuilder:object:root=true
type EgressPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressPolicy `json:"items"`
}

// EgressPolicy represents a single egress gateway policy
// +kubebuilder:resource:categories={egresspolicy},path="egresspolicies",singular="egresspolicy",scope="Namespaced",shortName={egp}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".spec.egressGatewayName",description="egressGatewayName",name="gateway",type=string
// +kubebuilder:printcolumn:JSONPath=".status.eip.ipv4",description="ipv4",name="ipv4",type=string
// +kubebuilder:printcolumn:JSONPath=".status.eip.ipv6",description="ipv6",name="ipv6",type=string
// +kubebuilder:printcolumn:JSONPath=".status.node",description="egressNode",name="egressNode",type=string
type EgressPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   EgressPolicySpec   `json:"spec,omitempty"`
	Status EgressPolicyStatus `json:"status,omitempty"`
}

type EgressPolicySpec struct {
	// +kubebuilder:validation:Optional
	EgressGatewayName string `json:"egressGatewayName,omitempty"`
	// +kubebuilder:validation:Optional
	EgressIP EgressIP `json:"egressIP,omitempty"`
	// +kubebuilder:validation:Required
	AppliedTo AppliedTo `json:"appliedTo"`
	// +kubebuilder:validation:Optional
	DestSubnet []string `json:"destSubnet"`
}

type EgressPolicyStatus struct {
	// +kubebuilder:validation:Optional
	Eip Eip `json:"eip,omitempty"`
	// +kubebuilder:validation:Optional
	Node string `json:"node,omitempty"`
}

type Eip struct {
	// +kubebuilder:validation:Optional
	Ipv4 string `json:"ipv4,omitempty"`
	// +kubebuilder:validation:Optional
	Ipv6 string `json:"ipv6,omitempty"`
}

type EgressIP struct {
	// +kubebuilder:validation:Optional
	IPv4 string `json:"ipv4,omitempty"`
	// +kubebuilder:validation:Optional
	IPv6 string `json:"ipv6,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	UseNodeIP bool `json:"useNodeIP,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="default"
	AllocatorPolicy string `json:"allocatorPolicy,omitempty"`
}

type AppliedTo struct {
	// +kubebuilder:validation:Optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
	// +kubebuilder:validation:Optional
	PodSubnet []string `json:"podSubnet,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EgressPolicy{}, &EgressPolicyList{})
}

const (
	// In the default mode, Ipv4DefaultEIP and Ipv6DefaultEIP are used if EIP is not specified
	EipAllocatorDefault = "default"
	// The unassigned EIP is preferred. If no EIP is available, select one at random
	EipAllocatorRR = "rr"
)
