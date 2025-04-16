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

// ScheduleSpec defines a scaling schedule using time ranges and days of the week.
type ScheduleSpec struct {
	// Days of the week to apply the schedule. Use "*" for every day.
	Days []string `json:"days"`

	// Start time of the scaling window in HH:MM format (24-hour).
	StartTime string `json:"startTime"`

	// End time of the scaling window in HH:MM format (24-hour).
	EndTime string `json:"endTime"`
}

// DeploymentScale defines scaling rules for a specific deployment
type DeploymentScale struct {
	Name        string        `json:"name"`
	Schedule    *ScheduleSpec `json:"schedule,omitempty"` // Changed to *ScheduleSpec
	MinReplicas int32         `json:"minReplicas,omitempty"`
	OptOut      bool          `json:"optOut,omitempty"`
}

// FinOpsScalePolicySpec defines the desired state of FinOpsScalePolicy.
type FinOpsScalePolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	OptOut          bool              `json:"optOut,omitempty"`
	DefaultSchedule *ScheduleSpec     `json:"defaultSchedule,omitempty"` // Changed to *ScheduleSpec
	Timezone        string            `json:"timezone,omitempty"`
	Deployments     []DeploymentScale `json:"deployments,omitempty"`
}

// FinOpsScalePolicyStatus defines the observed state of FinOpsScalePolicy.
type FinOpsScalePolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Active bool `json:"active"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// FinOpsScalePolicy is the Schema for the finopsscalepolicies API.
type FinOpsScalePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FinOpsScalePolicySpec   `json:"spec,omitempty"`
	Status FinOpsScalePolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FinOpsScalePolicyList contains a list of FinOpsScalePolicy.
type FinOpsScalePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FinOpsScalePolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FinOpsScalePolicy{}, &FinOpsScalePolicyList{})
}
