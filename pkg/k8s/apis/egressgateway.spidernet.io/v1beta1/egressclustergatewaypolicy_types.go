// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EgressClusterGatewayPolicyList contains a list of egress gateway policies
// +kubebuilder:object:root=true
type EgressClusterGatewayPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressClusterGatewayPolicy `json:"items"`
}

// EgressClusterGatewayPolicy represents a single egress gateway policy
// +kubebuilder:resource:categories={egressclustergatewaypolicy},path="egressclustergatewaypolicies",singular="egressclustergatewaypolicy",scope="Cluster",shortName={ecpo}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +genclient:nonNamespaced
type EgressClusterGatewayPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec EgressGatewayPolicySpec `json:"spec,omitempty"`
}

type EgressClusterGatewayPolicySpec struct {
	// +kubebuilder:validation:Optional
	EgressGatewayName string `json:"egressGatewayName,omitempty"`
	// +kubebuilder:validation:Optional
	EgressIP EgressIP `json:"egressIP,omitempty"`
	// +kubebuilder:validation:Required
	AppliedTo ClusterAppliedTo `json:"appliedTo"`
	// +kubebuilder:validation:Optional
	DestSubnet []string `json:"destSubnet"`
}

type ClusterAppliedTo struct {
	AppliedTo
	// +kubebuilder:validation:Optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}
