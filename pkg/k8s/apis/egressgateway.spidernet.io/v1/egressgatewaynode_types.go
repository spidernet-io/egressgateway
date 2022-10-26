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
	InterfaceList []InterfaceStatus `json:"interfaceList,omitempty"`

	// +kubebuilder:validation:Optional
	NodeList []string `json:"nodeList,omitempty"`
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
// +kubebuilder:resource:categories={egressgatewaynode},path="egressgatewaynodess",singular="egressgatewaynode",scope="Cluster",shortName={en}
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

// SpiderIPPoolList contains a list of SpiderIPPool
type EgressGatewayNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressGatewayNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EgressGatewayNode{}, &EgressGatewayNodeList{})
}
