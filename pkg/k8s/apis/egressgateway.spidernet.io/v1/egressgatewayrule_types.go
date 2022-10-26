// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

// !!!!!! crd marker:
// https://github.com/kubernetes-sigs/controller-tools/blob/master/pkg/crd/markers/crd.go
// https://book.kubebuilder.io/reference/markers/crd.html
// https://github.com/kubernetes-sigs/controller-tools/blob/master/pkg/crd/markers/validation.go
// https://book.kubebuilder.io/reference/markers/crd-validation.html

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EgressGatewayRuleSpec struct {

	// +kubebuilder:validation:Required
	AppliedTo AppliedTo `json:"appliedTo"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	DestSubnet []string `json:"destSubnet"`

	// +kubebuilder:validation:Optional
	GatewayNodeConfig GatewayNodeConfig `json:"gatewayNodeConfig"`
}

type AppliedTo struct {
	// +kubebuilder:validation:Optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`

	// +kubebuilder:validation:Optional
	PodSubnet *[]string `json:"PodSubnet,omitempty"`
}

type GatewayNodeConfig struct {
	// +kubebuilder:validation:Optional
	EgressIPv4VIP string `json:"egressIPv4VIP,omitempty"`

	// +kubebuilder:validation:Optional
	EgressIPv6VIP string `json:"egressIPv6VIP,omitempty"`

	// +kubebuilder:validation:Required
	Interface string `json:"interface"`
}

type EgressGatewayRuleStatus struct {
	// +kubebuilder:validation:Optional
	GatewayNodeList map[string]GatewayNodeStatus `json:"gatewayNodeList,omitempty"`

	// +kubebuilder:validation:Optional
	EgressIPv4IP string `json:"egressIPv4Ip,omitempty"`

	// +kubebuilder:validation:Optional
	EgressIPv6IP string `json:"egressIPv6Ip,omitempty"`

	// +kubebuilder:validation:Maximum=4095
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Optional
	Priority *int64 `json:"priority,omitempty"`
}

type GatewayNodeStatus struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum="active";"standby"
	Status string `json:"status"`
}

// scope(Namespaced or Cluster)
// +kubebuilder:resource:categories={egressgatewayrule},path="egressgatewayrules",singular="egressgatewayrule",scope="Cluster",shortName={er}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +genclient:nonNamespaced

type EgressGatewayRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   EgressGatewayRuleSpec   `json:"spec,omitempty"`
	Status EgressGatewayRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpiderIPPoolList contains a list of SpiderIPPool
type EgressGatewayRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressGatewayRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EgressGatewayRule{}, &EgressGatewayRuleList{})
}
