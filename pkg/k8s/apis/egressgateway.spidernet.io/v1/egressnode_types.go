// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EgressNodeSpec struct {
}

type EgressNodeStatus struct {
	// +kubebuilder:validation:Optional
	VxlanIPv4IP string `json:"vxlanIPv4IP,omitempty"`

	// +kubebuilder:validation:Optional
	VxlanIPv6IP string `json:"vxlanIPv6IP,omitempty"`

	// +kubebuilder:validation:Optional
	VxlanIPv4Mac string `json:"vxlanIPv4Mac,omitempty"`

	// +kubebuilder:validation:Optional
	VxlanIPv6Mac string `json:"vxlanIPv6Mac,omitempty"`

	// +kubebuilder:validation:Optional
	PhysicalInterface string `json:"physicalInterface,omitempty"`

	// +kubebuilder:validation:Optional
	PhysicalInterfaceIPv4 string `json:"physicalInterfaceIPv4,omitempty"`

	// +kubebuilder:validation:Optional
	PhysicalInterfaceIPv6 string `json:"physicalInterfaceIPv6,omitempty"`
}

// scope(Namespaced or Cluster)
// +kubebuilder:resource:categories={egressnode},path="egressnodes",singular="egressnode",scope="Cluster"
// +kubebuilder:printcolumn:JSONPath=".status.VxlanIPv4IP",description="tunnelIPv4",name="tunnelIPv4",type=string
// +kubebuilder:printcolumn:JSONPath=".status.VxlanIPv6IP",description="tunnelIPv6",name="tunnelIPv6",type=string
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
