package controller

import (
	"context"
	"testing"
	"time"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
)

func TestUpdateTargetObject(t *testing.T) {
	// set up the scheme for the fake client
	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &corev1.Pod{})

	// Create a fake client with a sample Deployment
	client := fake.NewClientBuilder().WithScheme(s).Build()

	// Create a reconciler with the fake client
	r := &VerticalWorkloadAutoscalerReconciler{
		Client: client,
		Scheme: s,
	}

	// Create a sample VWA object
	vwa := &vwav1.VerticalWorkloadAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vwa",
			Namespace: "default",
		},
		Spec: vwav1.VerticalWorkloadAutoscalerSpec{
			CustomAnnotations: map[string]string{
				"annotation-key": "annotation-value",
			},
		},
	}

	// Define test cases
	tests := []struct {
		name           string
		targetResource _client.Object
		newResources   map[string]corev1.ResourceRequirements
		updated        bool
		expectedError  bool
	}{
		{
			name: "Update Deployment",
			targetResource: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "test-container",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("100m"),
											corev1.ResourceMemory: resource.MustParse("200Mi"),
										},
									},
								},
							},
						},
					},
				},
			},
			newResources: map[string]corev1.ResourceRequirements{
				"test-container": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
			},
			updated:       true,
			expectedError: false,
		},
		{
			name: "Unsupported Resource Type",
			targetResource: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "test-container",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("200Mi"),
								},
							},
						},
					},
				},
			},
			newResources: map[string]corev1.ResourceRequirements{
				"test-container": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the target resource before updating
			err := r.Client.Create(context.TODO(), tt.targetResource)
			if err != nil {
				t.Fatalf("failed to create target resource: %v", err)
			}
			got, err := r.updateTargetObject(context.TODO(), tt.targetResource, vwa, tt.newResources)
			assert.Equal(t, tt.updated, got)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Check if annotations are set correctly
				annotations := tt.targetResource.GetAnnotations()
				assert.Equal(t, "annotation-value", annotations["annotation-key"])
				// Check if resources are updated correctly
				deployment := tt.targetResource.(*appsv1.Deployment)
				container := deployment.Spec.Template.Spec.Containers[0]
				assert.Equal(t, tt.newResources["test-container"].Requests, container.Resources.Requests)
				assert.Equal(t, tt.newResources["test-container"].Limits, container.Resources.Limits)
			}
		})
	}
}

func TestResourceRequirementsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        corev1.ResourceRequirements
		b        corev1.ResourceRequirements
		expected bool
	}{
		{
			name: "Equal resource requirements",
			a: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			b: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			expected: true,
		},
		{
			name: "Different CPU requests",
			a: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			b: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("150m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			expected: false,
		},
		{
			name: "Different memory requests",
			a: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			b: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("300Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			expected: false,
		},
		{
			name: "Different CPU limits",
			a: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			b: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			expected: false,
		},
		{
			name: "Different memory limits",
			a: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			b: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("500Mi"),
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resourceRequirementsEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFetchTargetObject(t *testing.T) {
	s := runtime.NewScheme()
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.StatefulSet{})
	s.AddKnownTypes(batchv1.SchemeGroupVersion, &batchv1.CronJob{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.ReplicaSet{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.DaemonSet{})
	s.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Pod{})

	client := fake.NewClientBuilder().WithScheme(s).Build()
	r := &VerticalWorkloadAutoscalerReconciler{
		Client: client,
		Scheme: s,
	}

	tests := []struct {
		name          string
		vpa           *vpav1.VerticalPodAutoscaler
		targetObject  _client.Object
		expectedKind  string
		expectedError bool
	}{
		{
			name: "Fetch Deployment",
			vpa: &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vpa", Namespace: "default"},
				Spec: vpav1.VerticalPodAutoscalerSpec{
					TargetRef: &autoscalingv1.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "test-deployment",
					},
				},
			},
			targetObject: &appsv1.Deployment{
				TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
			},
			expectedKind:  "Deployment",
			expectedError: false,
		},
		{
			name: "Fetch StatefulSet",
			vpa: &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vpa", Namespace: "default"},
				Spec: vpav1.VerticalPodAutoscalerSpec{
					TargetRef: &autoscalingv1.CrossVersionObjectReference{
						Kind: "StatefulSet",
						Name: "test-statefulset",
					},
				},
			},
			targetObject: &appsv1.StatefulSet{
				TypeMeta:   metav1.TypeMeta{Kind: "StatefulSet", APIVersion: "apps/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "test-statefulset", Namespace: "default"},
			},
			expectedKind:  "StatefulSet",
			expectedError: false,
		},
		{
			name: "Fetch CronJob",
			vpa: &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vpa", Namespace: "default"},
				Spec: vpav1.VerticalPodAutoscalerSpec{
					TargetRef: &autoscalingv1.CrossVersionObjectReference{
						Kind:       "CronJob",
						Name:       "test-cronjob",
						APIVersion: "batch/v1",
					},
				},
			},
			targetObject: &batchv1.CronJob{
				TypeMeta:   metav1.TypeMeta{Kind: "CronJob", APIVersion: "batch/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "test-cronjob", Namespace: "default"},
			},
			expectedKind:  "CronJob",
			expectedError: false,
		},
		{
			name: "Fetch ReplicaSet",
			vpa: &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vpa", Namespace: "default"},
				Spec: vpav1.VerticalPodAutoscalerSpec{
					TargetRef: &autoscalingv1.CrossVersionObjectReference{
						Kind: "ReplicaSet",
						Name: "test-replicaset",
					},
				},
			},
			targetObject: &appsv1.ReplicaSet{
				TypeMeta:   metav1.TypeMeta{Kind: "ReplicaSet", APIVersion: "apps/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "test-replicaset", Namespace: "default"},
			},
			expectedKind:  "ReplicaSet",
			expectedError: false,
		},
		{
			name: "Fetch DaemonSet",
			vpa: &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vpa", Namespace: "default"},
				Spec: vpav1.VerticalPodAutoscalerSpec{
					TargetRef: &autoscalingv1.CrossVersionObjectReference{
						Kind: "DaemonSet",
						Name: "test-daemonset",
					},
				},
			},
			targetObject: &appsv1.DaemonSet{
				TypeMeta:   metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "test-daemonset", Namespace: "default"},
			},
			expectedKind:  "DaemonSet",
			expectedError: false,
		},
		{
			name: "Unsupported Resource Kind",
			vpa: &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vpa", Namespace: "default"},
				Spec: vpav1.VerticalPodAutoscalerSpec{
					TargetRef: &autoscalingv1.CrossVersionObjectReference{
						Kind: "UnsupportedKind",
						Name: "test-unsupported",
					},
				},
			},
			targetObject: &corev1.Pod{
				TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "test-unsupported", Namespace: "default"},
			},
			expectedKind:  "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the target resource before fetching
			err := r.Client.Create(context.TODO(), tt.targetObject)
			if err != nil {
				t.Fatalf("failed to create target resource: %v", err)
			}
			targetResource, err := r.fetchTargetObject(context.TODO(), tt.vpa)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedKind, targetResource.GetObjectKind().GroupVersionKind().Kind)
			}
		})
	}
}

func TestGetTolerances(t *testing.T) {
	tests := []struct {
		name           string
		wa             vwav1.VerticalWorkloadAutoscaler
		expectedCPU    float64
		expectedMemory float64
	}{
		{
			name: "Default tolerances",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{},
			},
			expectedCPU:    0.10,
			expectedMemory: 0.10,
		},
		{
			name: "Custom CPU tolerance",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					UpdateTolerance: &vwav1.UpdateTolerance{
						CPU: 20,
					},
				},
			},
			expectedCPU:    0.20,
			expectedMemory: 0.10,
		},
		{
			name: "Custom Memory tolerance",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					UpdateTolerance: &vwav1.UpdateTolerance{
						Memory: 30,
					},
				},
			},
			expectedCPU:    0.10,
			expectedMemory: 0.30,
		},
		{
			name: "Custom CPU and Memory tolerances",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					UpdateTolerance: &vwav1.UpdateTolerance{
						CPU:    15,
						Memory: 25,
					},
				},
			},
			expectedCPU:    0.15,
			expectedMemory: 0.25,
		},
		{
			name: "Zero tolerances",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					UpdateTolerance: &vwav1.UpdateTolerance{
						CPU:    0,
						Memory: 0,
					},
				},
			},
			expectedCPU:    0.10,
			expectedMemory: 0.10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpuTolerance, memoryTolerance := getTolerances(tt.wa)
			assert.Equal(t, tt.expectedCPU, cpuTolerance)
			assert.Equal(t, tt.expectedMemory, memoryTolerance)
		})
	}
}

func TestApplyUpdate(t *testing.T) {
	tests := []struct {
		name        string
		current     resource.Quantity
		recommended resource.Quantity
		tolerance   float64
		expected    bool
	}{
		{
			name:        "Update when current is zero",
			current:     resource.MustParse("0"),
			recommended: resource.MustParse("100m"),
			tolerance:   0.1,
			expected:    true,
		},
		{
			name:        "Update when change exceeds tolerance",
			current:     resource.MustParse("100m"),
			recommended: resource.MustParse("150m"),
			tolerance:   0.1,
			expected:    true,
		},
		{
			name:        "No update when change is within tolerance",
			current:     resource.MustParse("100m"),
			recommended: resource.MustParse("105m"),
			tolerance:   0.1,
			expected:    false,
		},
		{
			name:        "Update when change is negative and exceeds tolerance",
			current:     resource.MustParse("100m"),
			recommended: resource.MustParse("50m"),
			tolerance:   0.1,
			expected:    true,
		},
		{
			name:        "No update when change is negative and within tolerance",
			current:     resource.MustParse("100m"),
			recommended: resource.MustParse("95m"),
			tolerance:   0.1,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyUpdate(tt.current, tt.recommended, tt.tolerance)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateGuaranteedResources(t *testing.T) {
	tests := []struct {
		name            string
		currentReq      corev1.ResourceRequirements
		containerRec    vpav1.RecommendedContainerResources
		cpuTolerance    float64
		memoryTolerance float64
		avoidCPULimit   bool
		expectedReq     corev1.ResourceRequirements
	}{
		{
			name: "Update CPU and Memory when both exceed tolerance",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				Target: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
			},
		},
		{
			name: "Avoid CPU limit when avoidCPULimit is true",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				Target: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   true,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
			},
		},
		{
			name: "No update when changes are within tolerance",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				Target: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("105m"),
					corev1.ResourceMemory: resource.MustParse("210Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
		},
		{
			name: "Update only Memory when CPU change is within tolerance",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				Target: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("105m"),
					corev1.ResourceMemory: resource.MustParse("300Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("300Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("300Mi"),
				},
			},
		},
		{
			name: "Update only CPU when Memory change is within tolerance",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				Target: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("210Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
		},
		{
			name: "Update when currentReq does not have Limits set",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				Target: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
			},
		},
		{
			name: "Update when currentReq does not have Requests set",
			currentReq: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				Target: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updateGuaranteedResources(tt.currentReq, tt.containerRec, tt.cpuTolerance, tt.memoryTolerance, tt.avoidCPULimit)
			assert.Equal(t, tt.expectedReq, *result)
		})
	}
}

func TestUpdateBurstableResources(t *testing.T) {
	tests := []struct {
		name            string
		currentReq      corev1.ResourceRequirements
		containerRec    vpav1.RecommendedContainerResources
		cpuTolerance    float64
		memoryTolerance float64
		avoidCPULimit   bool
		expectedReq     corev1.ResourceRequirements
	}{
		{
			name: "Update CPU and Memory when both exceed tolerance",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				LowerBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				UpperBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("800Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("800Mi"),
				},
			},
		},
		{
			name: "Avoid CPU limit when avoidCPULimit is true",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				LowerBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				UpperBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("800Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   true,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("800Mi"),
				},
			},
		},
		{
			name: "No update when changes are within tolerance",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				LowerBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("105m"),
					corev1.ResourceMemory: resource.MustParse("210Mi"),
				},
				UpperBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("210m"),
					corev1.ResourceMemory: resource.MustParse("410Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
		},
		{
			name: "Update only Memory when CPU change is within tolerance",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				LowerBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("105m"),
					corev1.ResourceMemory: resource.MustParse("300Mi"),
				},
				UpperBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("210m"),
					corev1.ResourceMemory: resource.MustParse("500Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("300Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("500Mi"),
				},
			},
		},
		{
			name: "Update only CPU when Memory change is within tolerance",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				LowerBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("210Mi"),
				},
				UpperBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("410Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
		},
		{
			name: "Update only Memory limits when CPU change is within tolerance",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				LowerBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("105m"),
					corev1.ResourceMemory: resource.MustParse("210Mi"),
				},
				UpperBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("210m"),
					corev1.ResourceMemory: resource.MustParse("500Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("500Mi"),
				},
			},
		},
		{
			name: "Update when currentReq does not have Limits set",
			currentReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("200Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				LowerBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				UpperBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("800Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("800Mi"),
				},
			},
		},
		{
			name: "Update when currentReq does not have Requests set",
			currentReq: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("400Mi"),
				},
			},
			containerRec: vpav1.RecommendedContainerResources{
				LowerBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				UpperBound: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("800Mi"),
				},
			},
			cpuTolerance:    0.1,
			memoryTolerance: 0.1,
			avoidCPULimit:   false,
			expectedReq: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("300m"),
					corev1.ResourceMemory: resource.MustParse("600Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("800Mi"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updateBurstableResources(tt.currentReq, tt.containerRec, tt.cpuTolerance, tt.memoryTolerance, tt.avoidCPULimit)
			assert.Equal(t, tt.expectedReq, *result)
		})
	}
}

func TestFetchCurrentResources(t *testing.T) {
	tests := []struct {
		name              string
		targetObject      _client.Object
		expectedResources map[string]corev1.ResourceRequirements
		expectedError     bool
	}{
		{
			name: "Fetch resources from Deployment",
			targetObject: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "test-container",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("100m"),
											corev1.ResourceMemory: resource.MustParse("200Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("200m"),
											corev1.ResourceMemory: resource.MustParse("400Mi"),
										},
									},
								},
							},
						},
					},
				},
			},
			expectedResources: map[string]corev1.ResourceRequirements{
				"test-container": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Unsupported resource type",
			targetObject: &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "test-container",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("200Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("400Mi"),
								},
							},
						},
					},
				},
			},
			expectedResources: nil,
			expectedError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &VerticalWorkloadAutoscalerReconciler{}
			resources, err := r.fetchCurrentResources(tt.targetObject)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResources, resources)
			}
		})
	}
}

func TestCalculateNewResources(t *testing.T) {
	tests := []struct {
		name             string
		wa               vwav1.VerticalWorkloadAutoscaler
		currentResources map[string]corev1.ResourceRequirements
		recommendations  *vpav1.RecommendedPodResources
		expected         map[string]corev1.ResourceRequirements
	}{
		{
			name: "Guaranteed QoS with updates",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					QualityOfService: vwav1.GuaranteedQualityOfService,
					AvoidCPULimit:    false,
				},
			},
			currentResources: map[string]corev1.ResourceRequirements{
				"test-container": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
			},
			recommendations: &vpav1.RecommendedPodResources{
				ContainerRecommendations: []vpav1.RecommendedContainerResources{
					{
						ContainerName: "test-container",
						Target: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("300m"),
							corev1.ResourceMemory: resource.MustParse("600Mi"),
						},
					},
				},
			},
			expected: map[string]corev1.ResourceRequirements{
				"test-container": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("600Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("600Mi"),
					},
				},
			},
		},
		{
			name: "Burstable QoS with updates",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					QualityOfService: vwav1.BurstableQualityOfService,
					AvoidCPULimit:    false,
				},
			},
			currentResources: map[string]corev1.ResourceRequirements{
				"test-container": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
			},
			recommendations: &vpav1.RecommendedPodResources{
				ContainerRecommendations: []vpav1.RecommendedContainerResources{
					{
						ContainerName: "test-container",
						LowerBound: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("300m"),
							corev1.ResourceMemory: resource.MustParse("600Mi"),
						},
						UpperBound: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("800Mi"),
						},
					},
				},
			},
			expected: map[string]corev1.ResourceRequirements{
				"test-container": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("600Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("800Mi"),
					},
				},
			},
		},
		{
			name: "Default QoS to Guaranteed",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					AvoidCPULimit: false,
				},
			},
			currentResources: map[string]corev1.ResourceRequirements{
				"test-container": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
			},
			recommendations: &vpav1.RecommendedPodResources{
				ContainerRecommendations: []vpav1.RecommendedContainerResources{
					{
						ContainerName: "test-container",
						Target: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("300m"),
							corev1.ResourceMemory: resource.MustParse("600Mi"),
						},
					},
				},
			},
			expected: map[string]corev1.ResourceRequirements{
				"test-container": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("600Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("300m"),
						corev1.ResourceMemory: resource.MustParse("600Mi"),
					},
				},
			},
		},
		{
			name: "No updates when within tolerance",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					QualityOfService: vwav1.GuaranteedQualityOfService,
					AvoidCPULimit:    false,
				},
			},
			currentResources: map[string]corev1.ResourceRequirements{
				"test-container": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
			},
			recommendations: &vpav1.RecommendedPodResources{
				ContainerRecommendations: []vpav1.RecommendedContainerResources{
					{
						ContainerName: "test-container",
						Target: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("105m"),
							corev1.ResourceMemory: resource.MustParse("210Mi"),
						},
					},
				},
			},
			expected: map[string]corev1.ResourceRequirements{
				"test-container": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &VerticalWorkloadAutoscalerReconciler{}
			result := r.calculateNewResources(tt.wa, tt.currentResources, tt.recommendations)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetAnnotations(t *testing.T) {
	tests := []struct {
		name                string
		targetObject        _client.Object
		vwa                 *vwav1.VerticalWorkloadAutoscaler
		expectedAnnotations map[string]string
	}{
		{
			name: "Add custom annotations",
			targetObject: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
			},
			vwa: &vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vwa",
					Namespace: "default",
				},
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					CustomAnnotations: map[string]string{
						"annotation-key": "annotation-value",
					},
				},
			},
			expectedAnnotations: map[string]string{
				"annotation-key": "annotation-value",
				"verticalworkloadautoscaler.kubernetes.io/lastUpdated": timeNow().Format(time.RFC3339),
				"verticalworkloadautoscaler.kubernetes.io/updatedBy":   "test-vwa",
			},
		},
		{
			name: "Overwrite existing annotations",
			targetObject: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
					Annotations: map[string]string{
						"annotation-key": "old-value",
					},
				},
			},
			vwa: &vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vwa",
					Namespace: "default",
				},
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					CustomAnnotations: map[string]string{
						"annotation-key": "new-value",
					},
				},
			},
			expectedAnnotations: map[string]string{
				"annotation-key": "new-value",
				"verticalworkloadautoscaler.kubernetes.io/lastUpdated": timeNow().Format(time.RFC3339),
				"verticalworkloadautoscaler.kubernetes.io/updatedBy":   "test-vwa",
			},
		},
		{
			name: "Add VWA specific annotations",
			targetObject: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
			},
			vwa: &vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vwa",
					Namespace: "default",
				},
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{},
			},
			expectedAnnotations: map[string]string{
				"verticalworkloadautoscaler.kubernetes.io/lastUpdated": timeNow().Format(time.RFC3339),
				"verticalworkloadautoscaler.kubernetes.io/updatedBy":   "test-vwa",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &VerticalWorkloadAutoscalerReconciler{}
			r.setAnnotations(tt.targetObject, tt.vwa)
			assert.Equal(t, tt.expectedAnnotations, tt.targetObject.GetAnnotations())
		})
	}
}
