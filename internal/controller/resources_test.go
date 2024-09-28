package controller

import (
	"context"
	"fmt"
	"testing"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/kubernetes/scheme"
	_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
)

func TestUpdateTargetResource(t *testing.T) {
	// set up the scheme for the fake client
	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.StatefulSet{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &batchv1.CronJob{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.ReplicaSet{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.DaemonSet{})

	// Create a fake client with a sample Deployment
	client := fake.NewClientBuilder().WithScheme(s).Build()

	// Create a reconciler with the fake client
	r := &VerticalWorkloadAutoscalerReconciler{
		Client: client,
		Scheme: s,
	}

	// Define test cases
	tests := []struct {
		name           string
		targetResource _client.Object
		newResources   map[string]corev1.ResourceRequirements
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
			updated, err := r.updateTargetResource(context.TODO(), tt.targetResource, tt.newResources)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.True(t, updated)
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

type errorClient struct {
	_client.Client
}

func (e *errorClient) Update(_ context.Context, _ _client.Object, _ ..._client.UpdateOption) error {
	return fmt.Errorf("client error")
}

func (e *errorClient) Watch(_ context.Context, _ _client.ObjectList, _ ..._client.ListOption) (watch.Interface, error) {
	return nil, fmt.Errorf("client error")
}

func TestUpdateAnnotations(t *testing.T) {
	tests := []struct {
		name           string
		targetResource _client.Object
		expectedError  bool
	}{
		{
			name: "Add annotations to resource without existing annotations",
			targetResource: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
			},
			expectedError: false,
		},
		{
			name: "Update annotations for resource with existing annotations",
			targetResource: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
					Annotations: map[string]string{
						"existing-annotation": "value",
					},
				},
			},
			expectedError: false,
		},
		{
			name: "Fail to update annotations due to client error",
			targetResource: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			r := &VerticalWorkloadAutoscalerReconciler{
				Client: client,
				Scheme: scheme.Scheme,
			}

			// Create the target resource before updating
			err := r.Client.Create(context.TODO(), tt.targetResource)
			if err != nil {
				t.Fatalf("failed to create target resource: %v", err)
			}

			if tt.expectedError {
				client = &errorClient{Client: client}
				r.Client = client
			}

			err = r.updateAnnotations(context.TODO(), tt.targetResource)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				annotations := tt.targetResource.GetAnnotations()
				assert.NotNil(t, annotations)
				assert.Contains(t, annotations, "verticalworkloadautoscaler.kubernetes.io/restartedAt")
				assert.Contains(t, annotations, "argocd.argoproj.io/compare-options")
				assert.Contains(t, annotations, "fluxcd.io/ignore")
			}
		})
	}
}

func TestFetchTargetObject(t *testing.T) {
	s := scheme.Scheme
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.StatefulSet{})
	s.AddKnownTypes(batchv1.SchemeGroupVersion, &batchv1.CronJob{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.ReplicaSet{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.DaemonSet{})

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
						Kind: "CronJob",
						Name: "test-cronjob",
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
					corev1.ResourceCPU:    resource.MustParse("250m"),
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
					corev1.ResourceCPU:    resource.MustParse("250m"),
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
