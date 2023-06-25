// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EgressClusterEndpointSliceList contains a list of EgressClusterEndpointSlice
// +kubebuilder:object:root=true
type EgressClusterEndpointSliceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressClusterEndpointSlice `json:"items"`
}

// EgressClusterEndpointSlice is a list of endpoint
// +kubebuilder:object:root=true
// +kubebuilder:resource:categories={egressclusterendpointslice},path="egressclusterendpointslices",singular="egressclusterendpointslice",scope="Cluster",shortName={egcep}
// +kubebuilder:subresource:status
type EgressClusterEndpointSlice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// +kubebuilder:validation:Optional
	Endpoints []EgressEndpoint `json:"endpoints,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EgressClusterEndpointSlice{}, &EgressClusterEndpointSliceList{})
}
