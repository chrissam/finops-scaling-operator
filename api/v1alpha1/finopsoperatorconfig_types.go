/*
Copyright 2025.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ExcludedDeploy defines deployments that should never be scaled down
type ExcludedDeploy struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// ScheduleSpec defines the global schedule for scaling when ForceScaleDown is true.
type GlobalScheduleSpec struct {
    Days      []string `json:"days"`
    StartTime string   `json:"startTime"`
    EndTime   string     `json:"endTime"`
}

// FinOpsOperatorConfigSpec defines the desired state of FinOpsOperatorConfig.
type FinOpsOperatorConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ExcludedNamespaces    []string         `json:"excludedNamespaces,omitempty"`
	ExcludedDeployments   []ExcludedDeploy `json:"excludedDeployments,omitempty"`
	MaxParallelOperations int              `json:"maxParallelOperations,omitempty"`
	CheckInterval         string           `json:"checkInterval,omitempty"`
	ForceScaleDown        bool             `json:"forceScaleDown,omitempty"`
	ForceScaleDownSchedule *GlobalScheduleSpec      `json:"forceScaleDownSchedule,omitempty"`
	ForceScaleDownTimezone        string            `json:"forceScaleDownTimezone,omitempty"`
}

// FinOpsOperatorConfigStatus defines the observed state of FinOpsOperatorConfig.
type FinOpsOperatorConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Active bool `json:"active"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// FinOpsOperatorConfig is the Schema for the finopsoperatorconfigs API.
type FinOpsOperatorConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FinOpsOperatorConfigSpec   `json:"spec,omitempty"`
	Status FinOpsOperatorConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FinOpsOperatorConfigList contains a list of FinOpsOperatorConfig.
type FinOpsOperatorConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FinOpsOperatorConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FinOpsOperatorConfig{}, &FinOpsOperatorConfigList{})
}
