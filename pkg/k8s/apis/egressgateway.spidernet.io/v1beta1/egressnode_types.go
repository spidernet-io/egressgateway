// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EgressNodeList egress node list
// +kubebuilder:object:root=true
type EgressNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressNode `json:"items"`
}

// EgressNode represents an egress node
// +kubebuilder:resource:categories={egressnode},path="egressnodes",singular="egressnode",scope="Cluster"
// +kubebuilder:printcolumn:JSONPath=".status.tunnel.mac",description="tunnelMac",name="tunnelMac",type=string
// +kubebuilder:printcolumn:JSONPath=".status.tunnel.ipv4",description="tunnelIPv4",name="tunnelIPv4",type=string
// +kubebuilder:printcolumn:JSONPath=".status.tunnel.ipv6",description="tunnelIPv6",name="tunnelIPv6",type=string
// +kubebuilder:printcolumn:JSONPath=".status.mark",description="mark",name="mark",type=string
// +kubebuilder:printcolumn:JSONPath=".status.phase",description="phase",name="phase",type=string
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type EgressNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   EgressNodeSpec   `json:"spec,omitempty"`
	Status EgressNodeStatus `json:"status,omitempty"`
}

type EgressNodeSpec struct{}

type EgressNodeStatus struct {
	// +kubebuilder:validation:Optional
	Tunnel Tunnel `json:"tunnel,omitempty"`
	// +kubebuilder:validation:Enum=Pending;Init;Failed;Succeeded;""
	Phase EgressNodePhase `json:"phase,omitempty"`
	// +kubebuilder:validation:Optional
	Mark string `json:"mark,omitempty"`
}

type Tunnel struct {
	// +kubebuilder:validation:Optional
	IPv4 string `json:"ipv4,omitempty"`
	// +kubebuilder:validation:Optional
	IPv6 string `json:"ipv6,omitempty"`
	// +kubebuilder:validation:Optional
	MAC string `json:"mac,omitempty"`
	// +kubebuilder:validation:Optional
	Parent Parent `json:"parent,omitempty"`
}

type Parent struct {
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Optional
	IPv4 string `json:"ipv4,omitempty"`
	// +kubebuilder:validation:Optional
	IPv6 string `json:"ipv6,omitempty"`
}

type EgressNodePhase string

const (
	// EgressNodePending wait for tunnel address available
	EgressNodePending EgressNodePhase = "Pending"
	// EgressNodeInit Init tunnel address
	EgressNodeInit EgressNodePhase = "Init"
	// EgressNodeFailed allocate tunnel address failed
	EgressNodeFailed EgressNodePhase = "Failed"
	// EgressNodeSucceeded tunnel is available
	EgressNodeSucceeded EgressNodePhase = "Succeeded"
)

func init() {
	SchemeBuilder.Register(&EgressNode{}, &EgressNodeList{})
}
