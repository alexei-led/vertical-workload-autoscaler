package controller

import (
	"context"
	"fmt"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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
				ObjectMeta: v1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
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
				ObjectMeta: v1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
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
				ObjectMeta: v1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
			},
			expectedError: false,
		},
		{
			name: "Update annotations for resource with existing annotations",
			targetResource: &appsv1.Deployment{
				ObjectMeta: v1.ObjectMeta{
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
				ObjectMeta: v1.ObjectMeta{Name: "test-deployment", Namespace: "default"},
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
