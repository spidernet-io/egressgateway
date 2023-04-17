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
	NodeSelector *metav1.LabelSelector `json:"nodeSelector,omitempty"`
	// +kubebuilder:validation:Optional
	Scope Scope `json:"scope,omitempty"`
}

type Ranges struct {
	// +kubebuilder:validation:Optional
	IPv4 []string `json:"ipv4,omitempty"`
	// +kubebuilder:validation:Optional
	IPv6 []string `json:"ipv6,omitempty"`
}

type Scope struct {
	// Namespaces represent the list of allowed namespaces to use, if empty, all namespaces are allowed to use.
	// +kubebuilder:validation:Optional
	Namespaces []string `json:"namespaces,omitempty"`
}

type EgressGatewayStatus struct {
	// +kubebuilder:validation:Optional
	NodeList []EgressIPStatus `json:"nodeList,omitempty"`
}

type EgressIPStatus struct {
	// +kubebuilder:validation:Optional
	IPv4 string `json:"ipv4,omitempty"`
	// +kubebuilder:validation:Optional
	IPv6 string `json:"ipv6,omitempty"`
	// +kubebuilder:validation:Optional
	Policies []string `json:"policies,omitempty"`
	// +kubebuilder:validation:Optional
	UseNodeIP bool `json:"useNodeIP,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EgressGateway{}, &EgressGatewayList{})
}
