// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EgressNodeSpec struct{}

type EgressNodeStatus struct {
	// +kubebuilder:validation:Optional
	VxlanIPv4 string `json:"vxlanIPv4,omitempty"`

	// +kubebuilder:validation:Optional
	VxlanIPv6 string `json:"vxlanIPv6,omitempty"`

	// +kubebuilder:validation:Optional
	TunnelMac string `json:"tunnelMac,omitempty"`

	// +kubebuilder:validation:Enum=Pending;Init;Failed;Succeeded;""
	// +optional
	Phase EgressNodePhase `json:"phase,omitempty"`

	// +kubebuilder:validation:Optional
	PhysicalInterface string `json:"physicalInterface,omitempty"`

	// +kubebuilder:validation:Optional
	PhysicalInterfaceIPv4 string `json:"physicalInterfaceIPv4,omitempty"`

	// +kubebuilder:validation:Optional
	PhysicalInterfaceIPv6 string `json:"physicalInterfaceIPv6,omitempty"`
}

type EgressNodePhase string

const (
	// EgressNodePending wait for tunnel address available
	EgressNodePending EgressNodePhase = "Pending"
	// EgressNodeInit Init tunnel address
	EgressNodeInit EgressNodePhase = "Init"
	// EgressNodeFailed allocate tunnel address failed
	EgressNodeFailed EgressNodePhase = "Failed"
	// EgressNodeSucceeded vxlan tunnel is available
	EgressNodeSucceeded EgressNodePhase = "Succeeded"
)

// scope(Namespaced or Cluster)
// +kubebuilder:resource:categories={egressnode},path="egressnodes",singular="egressnode",scope="Cluster"
// +kubebuilder:printcolumn:JSONPath=".status.tunnelMac",description="tunnelMac",name="tunnelMac",type=string
// +kubebuilder:printcolumn:JSONPath=".status.vxlanIPv4",description="tunnelIPv4",name="tunnelIPv4",type=string
// +kubebuilder:printcolumn:JSONPath=".status.vxlanIPv6",description="tunnelIPv6",name="tunnelIPv6",type=string
// +kubebuilder:printcolumn:JSONPath=".status.phase",description="phase",name="phase",type=string
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +genclient:nonNamespaced

type EgressNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   EgressNodeSpec   `json:"spec,omitempty"`
	Status EgressNodeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type EgressNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EgressNode{}, &EgressNodeList{})
}
