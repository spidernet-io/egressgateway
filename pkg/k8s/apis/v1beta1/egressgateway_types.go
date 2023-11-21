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
// +kubebuilder:printcolumn:JSONPath=".spec.ippools.ipv4DefaultEIP",description="ipv4DefaultEIP",name="ipv4DefaultEIP",type=string
// +kubebuilder:printcolumn:JSONPath=".spec.ippools.ipv6DefaultEIP",description="ipv6DefaultEIP",name="ipv6DefaultEIP",type=string
// +kubebuilder:printcolumn:JSONPath=".spec.clusterDefault",description="clusterDefault",name="clusterDefault",type=boolean
// +kubebuilder:printcolumn:JSONPath=".status.ipUsage.ipv4Total",description="ipv4Total",name="ipv4Total",type=integer
// +kubebuilder:printcolumn:JSONPath=".status.ipUsage.ipv4Free",description="ipv4Free",name="ipv4Free",type=integer
// +kubebuilder:printcolumn:JSONPath=".status.ipUsage.ipv6Total",description="ipv6Total",name="ipv6Total",type=integer
// +kubebuilder:printcolumn:JSONPath=".status.ipUsage.ipv6Free",description="ipv6Free",name="ipv6Free",type=integer
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
	ClusterDefault bool `json:"clusterDefault,omitempty"`
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
	// +kubebuilder:validation:Optional
	IPUsage IPUsage `json:"ipUsage,omitempty"`
}

type IPUsage struct {
	// +kubebuilder:validation:Optional
	IPv4Total int `json:"ipv4Total"`
	// +kubebuilder:validation:Optional
	IPv4Free int `json:"ipv4Free"`
	// +kubebuilder:validation:Optional
	IPv6Total int `json:"ipv6Total"`
	// +kubebuilder:validation:Optional
	IPv6Free int `json:"ipv6Free"`
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
	Eips []Eips `json:"eips,omitempty"`
	// +kubebuilder:validation:Optional
	Status string `json:"status,omitempty"`
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
