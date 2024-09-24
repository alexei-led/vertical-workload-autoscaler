/*
Copyright 2024 Alexei Ledenev.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WorkloadAutoscalerSpec defines the desired state of WorkloadAutoscaler
type WorkloadAutoscalerSpec struct {
	VPAReference         VPAReference     `json:"vpaReference"`
	UpdateFrequency      Duration         `json:"updateFrequency"`
	AllowedUpdateWindows []UpdateWindow   `json:"allowedUpdateWindows"`
	StepSize             ResourceRequests `json:"stepSize"`
	GracePeriod          Duration         `json:"gracePeriod"`
	// +kubebuilder:validation:Enum=Burstable;Guaranteed
	// +kubebuilder:default=Guaranteed
	QualityOfService string `json:"qualityOfService"`
	AvoidCPULimit    bool   `json:"avoidCPULimit,omitempty"`
}

// VPAReference defines the reference to the VerticalPodAutoscaler
type VPAReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// ResourceReference defines a reference to a Kubernetes resource
type ResourceReference struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
}

// UpdateWindow defines a time window for allowed updates
type UpdateWindow struct {
	DayOfWeek string `json:"dayOfWeek"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	TimeZone  string `json:"timeZone"`
}

// Duration defines a duration in Go Duration format
type Duration struct {
	Duration string `json:"duration"`
}

// WorkloadAutoscalerStatus defines the observed state of WorkloadAutoscaler
type WorkloadAutoscalerStatus struct {
	CurrentStatus       string                                 `json:"currentStatus,omitempty"`
	TargetResource      ResourceReference                      `json:"targetResource,omitempty"`
	LastUpdated         metav1.Time                            `json:"lastUpdated,omitempty"`
	CurrentRequests     ResourceRequests                       `json:"currentRequests,omitempty"`
	RecommendedRequests map[string]corev1.ResourceRequirements `json:"recommendedRequests,omitempty"`
	SkippedUpdates      bool                                   `json:"skippedUpdates,omitempty"`
	SkipReason          string                                 `json:"skipReason,omitempty"`
	StepSize            ResourceRequests                       `json:"stepSize,omitempty"`
	Errors              []string                               `json:"errors,omitempty"`
	UpdateCount         int32                                  `json:"updateCount,omitempty"`
	Conditions          []metav1.Condition                     `json:"conditions,omitempty"`
	QualityOfService    string                                 `json:"qualityOfService,omitempty"`
}

// ResourceRequests defines the resource requests for CPU and Memory
type ResourceRequests struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// WorkloadAutoscaler is the Schema for the workloadautoscalers API
type WorkloadAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadAutoscalerSpec   `json:"spec,omitempty"`
	Status WorkloadAutoscalerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkloadAutoscalerList contains a list of WorkloadAutoscaler
type WorkloadAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkloadAutoscaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkloadAutoscaler{}, &WorkloadAutoscalerList{})
}
