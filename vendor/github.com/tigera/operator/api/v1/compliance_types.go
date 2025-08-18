// Copyright (c) 2020-2024 Tigera, Inc. All rights reserved.
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComplianceSpec defines the desired state of Tigera compliance reporting capabilities.
type ComplianceSpec struct {

	// ComplianceControllerDeployment configures the Compliance Controller Deployment.
	// +optional
	ComplianceControllerDeployment *ComplianceControllerDeployment `json:"complianceControllerDeployment,omitempty"`

	// ComplianceSnapshotterDeployment configures the Compliance Snapshotter Deployment.
	// +optional
	ComplianceSnapshotterDeployment *ComplianceSnapshotterDeployment `json:"complianceSnapshotterDeployment,omitempty"`

	// ComplianceBenchmarkerDaemonSet configures the Compliance Benchmarker DaemonSet.
	// +optional
	ComplianceBenchmarkerDaemonSet *ComplianceBenchmarkerDaemonSet `json:"complianceBenchmarkerDaemonSet,omitempty"`

	// ComplianceServerDeployment configures the Compliance Server Deployment.
	// +optional
	ComplianceServerDeployment *ComplianceServerDeployment `json:"complianceServerDeployment,omitempty"`

	// ComplianceReporterPodTemplate configures the Compliance Reporter PodTemplate.
	// +optional
	ComplianceReporterPodTemplate *ComplianceReporterPodTemplate `json:"complianceReporterPodTemplate,omitempty"`
}

// ComplianceStatus defines the observed state of Tigera compliance reporting capabilities.
type ComplianceStatus struct {

	// State provides user-readable status.
	State string `json:"state,omitempty"`

	// Conditions represents the latest observed set of conditions for the component. A component may be one or more of
	// Ready, Progressing, Degraded or other customer types.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// Compliance installs the components required for Tigera compliance reporting. At most one instance
// of this resource is supported. It must be named "tigera-secure".
type Compliance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired state for Tigera compliance reporting.
	Spec ComplianceSpec `json:"spec,omitempty"`
	// Most recently observed state for Tigera compliance reporting.
	Status ComplianceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ComplianceList contains a list of Compliance
type ComplianceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Compliance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Compliance{}, &ComplianceList{})
}
