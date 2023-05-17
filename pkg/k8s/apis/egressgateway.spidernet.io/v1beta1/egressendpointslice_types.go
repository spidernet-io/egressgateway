// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EgressEndpointSliceList contains a list of EgressEndpointSlice
// +kubebuilder:object:root=true
type EgressEndpointSliceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressEndpointSlice `json:"items"`
}

// EgressEndpointSlice is a list of endpoint
// +kubebuilder:object:root=true
// +kubebuilder:resource:categories={egressendpointslice},path="egressendpointslices",singular="egressendpointslice",scope="Namespaced",shortName={ees}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
type EgressEndpointSlice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// +kubebuilder:validation:Optional
	Endpoints []EgressEndpoint `json:"endpoints,omitempty"`
}

type EgressEndpoint struct {
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
	// +kubebuilder:validation:Optional
	Pod string `json:"pod,omitempty"`
	// +kubebuilder:validation:Optional
	IPv4 []string `json:"ipv4,omitempty"`
	// +kubebuilder:validation:Optional
	IPv6 []string `json:"ipv6,omitempty"`
	// +kubebuilder:validation:Optional
	Node string `json:"node,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EgressEndpointSlice{}, &EgressEndpointSliceList{})
}
