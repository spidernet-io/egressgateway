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

type EgressGatewayPolicySpec struct {

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
	PodSubnet *[]string `json:"podSubnet,omitempty"`
}

type GatewayNodeConfig struct {
	// +kubebuilder:validation:Optional
	EgressIPv4VIP string `json:"egressIPv4VIP,omitempty"`

	// +kubebuilder:validation:Optional
	EgressIPv6VIP string `json:"egressIPv6VIP,omitempty"`

	// +kubebuilder:validation:Required
	Interface string `json:"interface"`
}

type GatewayNodeStatus struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum="Ready";"NotReady";"Unknown"
	Status string `json:"status"`
	// +kubebuilder:validation:Optional
	Active bool `json:"active"`
}

// scope(Namespaced or Cluster)
// +kubebuilder:resource:categories={egressgatewaypolicy},path="egressgatewaypolicies",singular="egressgatewaypolicy",scope="Cluster",shortName={epo}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +genclient:nonNamespaced

type EgressGatewayPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec EgressGatewayPolicySpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// EgressGatewayPolicyList contains a list of SpiderIPPool
type EgressGatewayPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressGatewayPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EgressGatewayPolicy{}, &EgressGatewayPolicyList{})
}
