// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// EgressClusterInfoList contains a list of EgressClusterInfo
// +kubebuilder:object:root=true
type EgressClusterInfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressClusterInfo `json:"items"`
}

// EgressClusterInfo describes the status of cluster
// +kubebuilder:object:root=true
// +kubebuilder:resource:categories={egressclusterinfo},path="egressclusterinfos",singular="egressclusterinfo",scope="Cluster",shortName={egci}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type EgressClusterInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// +kubebuilder:validation:Optional
	Spec EgressClusterInfoSpec `json:"spec,omitempty"`
	// +kubebuilder:validation:Optional
	Status EgressClusterInfoStatus `json:"status,omitempty"`
}

type EgressClusterInfoSpec struct {
	// +kubebuilder:validation:Optional
	AutoDetect AutoDetect `json:"autoDetect,omitempty"`
	// +kubebuilder:validation:Optional
	ExtraCidr []string `json:"extraCidr,omitempty"`
}

type EgressClusterInfoStatus struct {
	// +kubebuilder:validation:Optional
	PodCidrMode PodCidrMode `json:"podCidrMode,omitempty"`
	// +kubebuilder:validation:Optional
	NodeIP map[string]IPListPair `json:"nodeIP,omitempty"`
	// +kubebuilder:validation:Optional
	ClusterIP *IPListPair `json:"clusterIP,omitempty"`
	// +kubebuilder:validation:Optional
	PodCIDR map[string]IPListPair `json:"podCIDR,omitempty"`
	// +kubebuilder:validation:Optional
	ExtraCidr []string `json:"extraCidr,omitempty"`
}

type AutoDetect struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:="auto"
	PodCidrMode PodCidrMode `json:"podCidrMode,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	ClusterIP bool `json:"clusterIP,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	NodeIP bool `json:"nodeIP,omitempty"`
}

type IPListPair struct {
	// +kubebuilder:validation:Optional
	IPv4 []string `json:"ipv4,omitempty"`
	// +kubebuilder:validation:Optional
	IPv6 []string `json:"ipv6,omitempty"`
}

type PodCidrMode string

const (
	CniTypeK8s    PodCidrMode = "k8s"
	CniTypeCalico PodCidrMode = "calico"
	CniTypeAuto   PodCidrMode = "auto"
	CniTypeEmpty  PodCidrMode = ""
)

func init() {
	SchemeBuilder.Register(&EgressClusterInfo{}, &EgressClusterInfoList{})
}
