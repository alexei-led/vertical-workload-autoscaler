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

// QualityOfServiceClass defines the quality of service class
// Only Burstable and Guaranteed are supported
// if not set, the default is Guaranteed
// +kubebuilder:validation:Enum=Burstable;Guaranteed
// +kubebuilder:default=Guaranteed
type QualityOfServiceClass string

const (
	// BurstableQualityOfService is the burstable quality of service class
	BurstableQualityOfService QualityOfServiceClass = "Burstable"
	// GuaranteedQualityOfService is the guaranteed quality of service class
	GuaranteedQualityOfService QualityOfServiceClass = "Guaranteed"
)

// VerticalWorkloadAutoscalerSpec defines the desired state of VerticalWorkloadAutoscaler
type VerticalWorkloadAutoscalerSpec struct {
	// VPAReference defines the reference to the VerticalPodAutoscaler that this VWA is managing.
	// This allows the VWA to coordinate with the VPA to ensure optimal resource allocation.
	// +kubebuilder:validation:required
	VPAReference VPAReference `json:"vpaReference"`

	// UpdateFrequency specifies how often the VWA should check and apply updates to resource requests.
	// It is defined as a duration (e.g., "30s", "1m"). The default value is set to 5 minutes if not specified.
	// +kubebuilder:default="5m"
	// +optional
	UpdateFrequency *metav1.Duration `json:"updateFrequency"`

	// AllowedUpdateWindows defines specific time windows during which updates to resource requests
	// are permitted. This can help minimize disruptions during peak usage times.
	// Each update window should specify the day of the week, start time, and end time.
	// +optional
	AllowedUpdateWindows []UpdateWindow `json:"allowedUpdateWindows"`

	// StepSize represents the incremental size by which the resource requests (CPU and Memory)
	// can be adjusted during each update. This allows for controlled and gradual scaling.
	// For example, a step size of "200m" for CPU means that requests can be increased or decreased
	// by 200 milliCPU units.
	// +optional
	StepSize ResourceRequests `json:"stepSize"`

	// GracePeriod indicates the duration the VWA should wait after applying updates before
	// making further adjustments. This allows for stabilization of the resource usage after changes.
	// The default value is set to 30 seconds if not specified.
	// +kubebuilder:default="30s"
	// +optional
	GracePeriod *metav1.Duration `json:"gracePeriod"`

	// QualityOfService defines the quality of service class to be applied to the managed resource.
	// This can help Kubernetes make scheduling decisions based on the resource guarantees.
	// Possible values are:
	// - "Guaranteed": CPU and Memory requests are equal to limits for all containers.
	// - "Burstable": Requests are lower than limits, allowing bursts of usage.
	// If not set, the default is "Guaranteed".
	// +kubebuilder:validation:Enum=Guaranteed;Burstable
	// +kubebuilder:default=Guaranteed
	// +optional
	QualityOfService QualityOfServiceClass `json:"qualityOfService"`

	// AvoidCPULimit indicates whether the VWA should avoid setting CPU limits on the managed resource.
	// If set to true, only resource requests will be set, which may be beneficial in scenarios
	// where burstable workloads are expected. The default value is true.
	// +kubebuilder:default=true
	// +optional
	AvoidCPULimit bool `json:"avoidCPULimit,omitempty"`
}

// VPAReference defines the reference to the VerticalPodAutoscaler
type VPAReference struct {
	Name string `json:"name"`
}

// HPAReference defines the reference to the HorizontalPodAutoscaler
type HPAReference struct {
	Name string `json:"name"`
}

// ResourceReference defines a reference to a Kubernetes resource
type ResourceReference struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

// UpdateWindow defines a time window for allowed updates
type UpdateWindow struct {
	// DayOfWeek represents the day of the week for the update window.
	// +kubebuilder:validation:Enum=Monday;Tuesday;Wednesday;Thursday;Friday;Saturday;Sunday
	DayOfWeek string `json:"dayOfWeek"`

	// StartTime represents the start of the update window
	// +kubebuilder:validation:required
	StartTime metav1.Time `json:"startTime"`

	// EndTime represents the end of the update window
	// +kubebuilder:validation:required
	EndTime metav1.Time `json:"endTime"`

	// TimeZone represents the time zone in IANA format, like "UTC" or "America/New_York"
	// +kubebuilder:validation:Pattern="^[A-Za-z]+/[A-Za-z_]+$"
	TimeZone string `json:"timeZone"`
}

// VerticalWorkloadAutoscalerStatus defines the observed state of VerticalWorkloadAutoscaler
type VerticalWorkloadAutoscalerStatus struct {
	// CurrentStatus represents the current status of the VWA.
	// Possible values:
	// - "Pending": VWA is being initialized.
	// - "Active": VWA is actively managing workloads.
	// - "Updating": VWA is applying updates to resources.
	// - "Failed": VWA encountered an error while processing.
	// +kubebuilder:validation:Enum=Pending;Active;Updating;Failed
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// TargetResource defines the reference to the resource being managed by the VWA.
	// This could reference different kinds of resources (e.g., Deployment, StatefulSet).
	// +kubebuilder:validation:required
	TargetResource ResourceReference `json:"targetResource"`

	// HPAReference defines the reference to the HorizontalPodAutoscaler that is associated with the VWA.
	// +optional
	HPAReference *HPAReference `json:"hpaReference,omitempty"`

	// LastUpdated indicates the last time the VWA status was updated.
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// CurrentRequests represents the current resource requests (CPU and Memory) for the managed resource.
	// +optional
	CurrentRequests ResourceRequests `json:"currentRequests,omitempty"`

	// RecommendedRequests maps the recommended resource requests for the managed resource.
	// The key is the container name, and the value is the resource requirements.
	// +optional
	RecommendedRequests map[string]corev1.ResourceRequirements `json:"recommendedRequests,omitempty"`

	// SkippedUpdates indicates whether updates were skipped during the last reconciliation.
	// +optional
	SkippedUpdates bool `json:"skippedUpdates,omitempty"`

	// SkipReason provides the reason for skipped updates, if applicable.
	// +optional
	SkipReason string `json:"skipReason,omitempty"`

	// StepSize defines the resource change size for each update.
	// +optional
	StepSize ResourceRequests `json:"stepSize,omitempty"`

	// Errors contains a list of error messages encountered during the VWA's operation.
	// +optional
	Errors []string `json:"errors,omitempty"`

	// UpdateCount represents the number of updates applied by the VWA.
	// +optional
	UpdateCount int32 `json:"updateCount,omitempty"`

	// Conditions contains the current conditions of the VWA, which can provide insights
	// about its operational state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// QualityOfService defines the quality of service class being applied to the resource.
	// Possible values:
	// - "Guaranteed": CPU/Memory requests are equal to limits for all containers.
	// - "Burstable": Requests are lower than limits, allowing bursts.
	// +kubebuilder:validation:Enum=Guaranteed;Burstable
	// +optional
	QualityOfService string `json:"qualityOfService,omitempty"`

	// Conflicts contains a list of resources that conflict with the VWA's recommendations.
	// +optional
	Conflicts []Conflict `json:"conflicts,omitempty"`
}

// ResourceRequests defines the resource requests for CPU and Memory
type ResourceRequests struct {
	// Step size for CPU requests (default: 100m)
	// +kubebuilder:default="100m"
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Step size for memory requests (default: 128Mi)
	// +kubebuilder:default="128Mi"
	// +optional
	Memory string `json:"memory,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=vwa

// VerticalWorkloadAutoscaler is the Schema for the VerticalWorkloadAutoscalers API
type VerticalWorkloadAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerticalWorkloadAutoscalerSpec   `json:"spec,omitempty"`
	Status VerticalWorkloadAutoscalerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerticalWorkloadAutoscalerList contains a list of VerticalWorkloadAutoscaler
type VerticalWorkloadAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VerticalWorkloadAutoscaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VerticalWorkloadAutoscaler{}, &VerticalWorkloadAutoscalerList{})
}

type Conflict struct {
	Resource     string `json:"resource"`
	ConflictWith string `json:"conflictWith"`
	Reason       string `json:"reason,omitempty"`
}
