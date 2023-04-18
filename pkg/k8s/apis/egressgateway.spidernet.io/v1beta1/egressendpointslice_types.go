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
	Spec EgressEndpointSliceSpec `json:"spec,omitempty"`
	// +kubebuilder:validation:Optional
	Status EgressEndpointSliceSpecStatus `json:"status,omitempty"`
}

type EgressEndpointSliceSpec struct{}

type EgressEndpointSliceSpecStatus struct {
	Endpoints []EgressEndpoint `json:"endpoints,omitempty"`
}

type EgressEndpoint struct {
	// +kubebuilder:validation:Optional
	PodName string `json:"podName,omitempty"`
	// +kubebuilder:validation:Optional
	IPv4List []string `json:"IPv4List,omitempty"`
	// +kubebuilder:validation:Optional
	IPv6List []string `json:"IPv6List,omitempty"`
	// +kubebuilder:validation:Optional
	NodeName string `json:"nodeName,omitempty"`
	// +kubebuilder:validation:Optional
	UUID string `json:"uuid,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EgressEndpointSlice{}, &EgressEndpointSliceList{})
}
