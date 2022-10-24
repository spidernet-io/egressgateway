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

type EgressGatewaySpec struct {

	// +kubebuilder:validation:Required
	AppliedTo AppliedTo `json:"appliedTo"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	DestSubnet []string `json:"destSubnet"`

	// +kubebuilder:validation:Optional
	GatewayNodeSelector metav1.LabelSelector `json:"gatewayNodeSelector,omitempty"`

	// +kubebuilder:validation:Maximum=4095
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Optional
	Priority *int64 `json:"priority,omitempty"`
}

type AppliedTo struct {
	// +kubebuilder:validation:Optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty"`

	// +kubebuilder:validation:Optional
	PodSubnet *[]string `json:"PodSubnet"`
}

type EgressGatewayStatus struct {
	// +kubebuilder:validation:Optional
	GatewayNodeList map[string]GatewayNodeStatus `json:"gatewayNodeList,omitempty"`

	// +kubebuilder:validation:Optional
	EgressIPv4Vip string `json:"egressIPv4Vip,omitempty"`

	// +kubebuilder:validation:Optional
	EgressIPv6Vip string `json:"egressIPv6Vip,omitempty"`
}

type GatewayNodeStatus struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum="active";"standby"
	Status string `json:"status"`
}

// scope(Namespaced or Cluster)
// +kubebuilder:resource:categories={egressgateway},path="egressgateways",singular="egressgateway",scope="Cluster",shortName={et}
// +kubebuilder:printcolumn:JSONPath=".spec.priority",description="priority",name="PRIORITY",type=string
// +kubebuilder:printcolumn:JSONPath=".spec.appliedTo.srcSubnet",description="srcSubnet",name="SRCSUBNET",type=string
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

// +kubebuilder:object:root=true

// SpiderIPPoolList contains a list of SpiderIPPool
type EgressGatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressGateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EgressGateway{}, &EgressGatewayList{})
}
