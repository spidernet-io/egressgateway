// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EgressTunnelList egress tunnel list
// +kubebuilder:object:root=true
type EgressTunnelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []EgressTunnel `json:"items"`
}

// EgressTunnel represents an egress tunnel
// +kubebuilder:resource:categories={egresstunnel},path="egresstunnels",singular="egresstunnel",scope="Cluster",shortName={egt}
// +kubebuilder:printcolumn:JSONPath=".status.tunnel.mac",description="tunnelMac",name="tunnelMac",type=string
// +kubebuilder:printcolumn:JSONPath=".status.tunnel.ipv4",description="tunnelIPv4",name="tunnelIPv4",type=string
// +kubebuilder:printcolumn:JSONPath=".status.tunnel.ipv6",description="tunnelIPv6",name="tunnelIPv6",type=string
// +kubebuilder:printcolumn:JSONPath=".status.mark",description="mark",name="mark",type=string
// +kubebuilder:printcolumn:JSONPath=".status.phase",description="phase",name="phase",type=string
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type EgressTunnel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   EgressTunnelSpec   `json:"spec,omitempty"`
	Status EgressTunnelStatus `json:"status,omitempty"`
}

type EgressTunnelSpec struct{}

type EgressTunnelStatus struct {
	// +kubebuilder:validation:Optional
	Tunnel Tunnel `json:"tunnel,omitempty"`
	// +kubebuilder:validation:Enum=Pending;Init;Failed;Ready;HeartbeatTimeout;NodeNotReady
	Phase EgressTunnelPhase `json:"phase,omitempty"`
	// +kubebuilder:validation:Optional
	Mark string `json:"mark,omitempty"`
	// +kubebuilder:validation:Optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
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

type EgressTunnelPhase string

func (e EgressTunnelPhase) String() string {
	return string(e)
}

func (e EgressTunnelPhase) IsEqual(s string) bool {
	return string(e) == s
}

func (e EgressTunnelPhase) IsNotEqual(s string) bool {
	return string(e) != s
}

const (
	// EgressTunnelPending wait for tunnel address available
	EgressTunnelPending EgressTunnelPhase = "Pending"
	// EgressTunnelInit Init tunnel address
	EgressTunnelInit EgressTunnelPhase = "Init"
	// EgressTunnelFailed allocate tunnel address failed
	EgressTunnelFailed EgressTunnelPhase = "Failed"
	// EgressTunnelHeartbeatTimeout tunnel heartbeat timeout
	EgressTunnelHeartbeatTimeout EgressTunnelPhase = "HeartbeatTimeout"
	// EgressTunnelNodeNotReady node not ready
	EgressTunnelNodeNotReady EgressTunnelPhase = "NodeNotReady"
	// EgressTunnelReady tunnel is available
	EgressTunnelReady EgressTunnelPhase = "Ready"
)

var ReasonStatusChanged = "StatusChanged"

func init() {
	SchemeBuilder.Register(&EgressTunnel{}, &EgressTunnelList{})
}
