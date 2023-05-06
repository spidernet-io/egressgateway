// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EgressGatewayList contains a list of EgressGateway
// +kubebuilder:object:root=true
type EgressGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressGateway `json:"items"`
}

// EgressGateway egress gateway
// +kubebuilder:object:root=true
// +kubebuilder:resource:categories={egressgateway},path="egressgateways",singular="egressgateway",scope="Cluster",shortName={egw}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +genclient:nonNamespaced
type EgressGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   EgressGatewaySpec   `json:"spec,omitempty"`
	Status EgressGatewayStatus `json:"status,omitempty"`
}

type EgressGatewaySpec struct {
	// +kubebuilder:validation:Optional
	Ranges Ranges `json:"ranges,omitempty"`
	// +kubebuilder:validation:Required
	NodeSelector NodeSelector `json:"nodeSelector,omitempty"`
}

type Ranges struct {
	// +kubebuilder:validation:Optional
	IPv4 []string `json:"ipv4,omitempty"`
	// +kubebuilder:validation:Optional
	IPv6 []string `json:"ipv6,omitempty"`
	// +kubebuilder:validation:Optional
	Policy []string `json:"policy,omitempty"`
}

type NodeSelector struct {
	// +kubebuilder:validation:Optional
	Policy string `json:"policy,omitempty"`
	// +kubebuilder:validation:Required
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

type EgressGatewayStatus struct {
	// +kubebuilder:validation:Optional
	NodeList []EgressIPStatus `json:"nodeList,omitempty"`
}

type EgressIPStatus struct {
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Optional
	Eips []Eips `json:"epis,omitempty"`
}

type Eips struct {
	// +kubebuilder:validation:Optional
	IPv4 string `json:"ipv4,omitempty"`
	// +kubebuilder:validation:Optional
	IPv6 string `json:"ipv6,omitempty"`
	// +kubebuilder:validation:Optional
	Policies []string `json:"policies,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EgressGateway{}, &EgressGatewayList{})
}
