// Copyright (c) 2024 Tigera, Inc. All rights reserved.
/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in with the License.
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

// ComplianceReporterPodTemplate is the configuration for the ComplianceReporter PodTemplate.
type ComplianceReporterPodTemplate struct {

	// Spec is the specification of the ComplianceReporter PodTemplateSpec.
	// +optional
	Template *ComplianceReporterPodTemplateSpec `json:"template,omitempty"`
}

// ComplianceReporterPodTemplateSpec is the ComplianceReporter PodTemplateSpec.
type ComplianceReporterPodTemplateSpec struct {

	// Spec is the ComplianceReporter PodTemplate's PodSpec.
	// +optional
	Spec *ComplianceReporterPodSpec `json:"spec,omitempty"`
}

// ComplianceReporterPodSpec is the ComplianceReporter PodSpec.
type ComplianceReporterPodSpec struct {
	// InitContainers is a list of ComplianceReporter PodSpec init containers.
	// If specified, this overrides the specified ComplianceReporter PodSpec init containers.
	// If omitted, the ComplianceServer Deployment will use its default values for its init containers.
	// +optional
	InitContainers []ComplianceReporterPodTemplateInitContainer `json:"initContainers,omitempty"`

	// Containers is a list of ComplianceServer containers.
	// If specified, this overrides the specified ComplianceReporter PodSpec containers.
	// If omitted, the ComplianceServer Deployment will use its default values for its containers.
	// +optional
	Containers []ComplianceReporterPodTemplateContainer `json:"containers,omitempty"`
}

// ComplianceReporterPodTemplateContainer is a ComplianceServer Deployment container.
type ComplianceReporterPodTemplateContainer struct {
	// Name is an enum which identifies the ComplianceServer Deployment container by name.
	// Supported values are: reporter
	// +kubebuilder:validation:Enum=reporter
	Name string `json:"name"`

	// Resources allows customization of limits and requests for compute resources such as cpu and memory.
	// If specified, this overrides the named ComplianceServer Deployment container's resources.
	// If omitted, the ComplianceServer Deployment will use its default value for this container's resources.
	// +optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
}

// ComplianceReporterPodTemplateInitContainer is a ComplianceServer Deployment init container.
type ComplianceReporterPodTemplateInitContainer struct {
	// Name is an enum which identifies the ComplianceReporter PodSpec init container by name.
	// Supported values are: tigera-compliance-reporter-tls-key-cert-provisioner
	// +kubebuilder:validation:Enum=tigera-compliance-reporter-tls-key-cert-provisioner
	Name string `json:"name"`

	// Resources allows customization of limits and requests for compute resources such as cpu and memory.
	// If specified, this overrides the named ComplianceReporter PodSpec init container's resources.
	// If omitted, the ComplianceServer Deployment will use its default value for this init container's resources.
	// +optional
	Resources *v1.ResourceRequirements `json:"resources,omitempty"`
}
