// Copyright (c) 2024 Tigera, Inc. All rights reserved.
/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	v1 "k8s.io/api/core/v1"
)

// ComplianceBenchmarkerDaemonSet is the configuration for the Compliance Benchmarker DaemonSet.
type ComplianceBenchmarkerDaemonSet struct {

	// Spec is the specification of the Compliance Benchmarker DaemonSet.
	// +optional
	Spec *ComplianceBenchmarkerDaemonSetSpec `json:"spec,omitempty"`
}

// ComplianceBenchmarkerDaemonSetSpec defines configuration for the Compliance Benchmarker DaemonSet.
type ComplianceBenchmarkerDaemonSetSpec struct {

	// Template describes the Compliance Benchmarker DaemonSet pod that will be created.
	// +optional
	Template *ComplianceBenchmarkerDaemonSetPodTemplateSpec `json:"template,omitempty"`
}

// ComplianceBenchmarkerDaemonSetPodTemplateSpec is the Compliance Benchmarker DaemonSet's PodTemplateSpec
type ComplianceBenchmarkerDaemonSetPodTemplateSpec struct {

	// Spec is the Compliance Benchmarker DaemonSet's PodSpec.
	// +optional
	Spec *ComplianceBenchmarkerDaemonSetPodSpec `json:"spec,omitempty"`
}

// ComplianceBenchmarkerDaemonSetPodSpec is the Compliance Benchmarker DaemonSet's PodSpec.
type ComplianceBenchmarkerDaemonSetPodSpec struct {
	// InitContainers is a list of Compliance benchmark init containers.
	// If specified, this overrides the specified Compliance Benchmarker DaemonSet init containers.
	// If omitted, the Compliance Benchmarker DaemonSet will use its default values for its init containers.
	// +optional
	InitContainers []ComplianceBenchmarkerDaemonSetInitContainer `json:"initContainers,omitempty"`

	// Containers is a list of Compliance benchmark containers.
	// If specified, this overrides the specified Compliance Benchmarker DaemonSet containers.
	// If omitted, the Compliance Benchmarker DaemonSet will use its default values for its containers.
	// +optional
	Containers []ComplianceBenchmarkerDaemonSetContainer `json:"containers,omitempty"`
}

// ComplianceBenchmarkerDaemonSetContainer is a Compliance Benchmarker DaemonSet container.
type ComplianceBenchmarkerDaemonSetContainer struct {
	// Name is an enum which identifies the Compliance Benchmarker DaemonSet container by name.
	// Supported values are: compliance-benchmarker
	// +kubebuilder:validation:Enum=compliance-benchmarker
	Name string `json:"name"`

	// Resources allows customization of limits and requests for compute resources such as cpu and memory.
	// If specified, this overrides the named Compliance Benchmarker DaemonSet container's resources.
	// If omitted, the Compliance Benchmarker DaemonSet will use its default value for this container's resources.
	// +optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
}

// ComplianceBenchmarkerDaemonSetInitContainer is a Compliance Benchmarker DaemonSet init container.
type ComplianceBenchmarkerDaemonSetInitContainer struct {
	// Name is an enum which identifies the Compliance Benchmarker DaemonSet init container by name.
	// Supported values are: tigera-compliance-benchmarker-tls-key-cert-provisioner
	// +kubebuilder:validation:Enum=tigera-compliance-benchmarker-tls-key-cert-provisioner
	Name string `json:"name"`

	// Resources allows customization of limits and requests for compute resources such as cpu and memory.
	// If specified, this overrides the named Compliance Benchmarker DaemonSet init container's resources.
	// If omitted, the Compliance Benchmarker DaemonSet will use its default value for this init container's resources.
	// +optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
}
