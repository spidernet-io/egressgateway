// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeSpec struct {
}

type NodeStatus struct {
	// +kubebuilder:validation:Optional
	VxlanIPv4IP string `json:"VxlanIPv4IP,omitempty"`

	// +kubebuilder:validation:Optional
	VxlanIPv6IP string `json:"VxlanIPv6IP,omitempty"`
}

// scope(Namespaced or Cluster)
// +kubebuilder:resource:categories={node},path="nodes",singular="node",scope="Cluster",shortName={nd}
// +kubebuilder:printcolumn:JSONPath=".status.VxlanIPv4IP",description="tunnelIPv4",name="tunnelIPv4",type=string
// +kubebuilder:printcolumn:JSONPath=".status.VxlanIPv6IP",description="tunnelIPv6",name="tunnelIPv6",type=string
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +genclient:nonNamespaced

type Node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   NodeSpec   `json:"spec,omitempty"`
	Status NodeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type NodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Node `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Node{}, &NodeList{})
}
