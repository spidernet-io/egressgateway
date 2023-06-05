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
type EgressGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   EgressGatewaySpec   `json:"spec,omitempty"`
	Status EgressGatewayStatus `json:"status,omitempty"`
}

type EgressGatewaySpec struct {
	// +kubebuilder:validation:Optional
	Ippools Ippools `json:"ippools,omitempty"`
	// +kubebuilder:validation:Required
	NodeSelector NodeSelector `json:"nodeSelector,omitempty"`
}

type Ippools struct {
	// +kubebuilder:validation:Optional
	IPv4 []string `json:"ipv4,omitempty"`
	// +kubebuilder:validation:Optional
	IPv6 []string `json:"ipv6,omitempty"`
	// +kubebuilder:validation:Optional
	Ipv4DefaultEIP string `json:"ipv4DefaultEIP,omitempty"`
	// +kubebuilder:validation:Optional
	Ipv6DefaultEIP string `json:"ipv6DefaultEIP,omitempty"`
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

func (status *EgressGatewayStatus) GetNodeIPs(nodeName string) []Eips {
	for _, items := range status.NodeList {
		if items.Name == nodeName {
			return items.Eips
		}
	}
	return make([]Eips, 0)
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
	Policies []Policy `json:"policies,omitempty"`
}

type Policy struct {
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

func init() {
	SchemeBuilder.Register(&EgressGateway{}, &EgressGatewayList{})
}
