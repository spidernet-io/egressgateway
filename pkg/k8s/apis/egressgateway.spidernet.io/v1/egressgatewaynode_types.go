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

type EgressGatewayNodeSpec struct {
	// +kubebuilder:validation:Required
	NodeSelector *metav1.LabelSelector `json:"nodeSelector,omitempty"`
}

type EgressGatewayNodeStatus struct {
	// +kubebuilder:validation:Optional
	NodeList SelectedEgressNodes `json:"nodeList,omitempty"`
}

type SelectedEgressNodes []SelectedEgressNode

type SelectedEgressNode struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Optional
	Ready bool `json:"ready"`
	// +kubebuilder:validation:Optional
	Active bool `json:"active"`
	// +kubebuilder:validation:Optional
	InterfaceStatus []InterfaceStatus `json:"interfaceStatus"`
}

func (s SelectedEgressNodes) Len() int {
	return len(s)
}

func (s SelectedEgressNodes) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

func (s SelectedEgressNodes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type InterfaceStatus struct {
	// +kubebuilder:validation:Required
	InterfaceName string `json:"interfaceName"`

	// +kubebuilder:validation:Required
	IPv4List []string `json:"ipv4List"`

	// +kubebuilder:validation:Required
	IPv6List []string `json:"ipv6List"`
}

// scope(Namespaced or Cluster)
// +kubebuilder:resource:categories={egressgatewaynode},path="egressgatewaynodes",singular="egressgatewaynode",scope="Cluster",shortName={egn}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +genclient
// +genclient:nonNamespaced

type EgressGatewayNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   EgressGatewayNodeSpec   `json:"spec,omitempty"`
	Status EgressGatewayNodeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EgressGatewayNodeList contains a list of SpiderIPPool
type EgressGatewayNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressGatewayNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EgressGatewayNode{}, &EgressGatewayNodeList{})
}
