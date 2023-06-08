// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EgressClusterPolicyList contains a list of egress gateway policies
// +kubebuilder:object:root=true
type EgressClusterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressClusterPolicy `json:"items"`
}

// EgressClusterPolicy represents a cluster egress policy
// +kubebuilder:resource:categories={egressclusterpolicy},path="egressclusterpolicies",singular="egressclusterpolicy",scope="Cluster",shortName={egcp}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +genclient:nonNamespaced
type EgressClusterPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec EgressClusterPolicySpec `json:"spec,omitempty"`
}

type EgressClusterPolicySpec struct {
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
	// +kubebuilder:validation:Optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`
	// +kubebuilder:validation:Optional
	PodSubnet *[]string `json:"podSubnet,omitempty"`
	// +kubebuilder:validation:Optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EgressClusterPolicy{}, &EgressClusterPolicyList{})
}
