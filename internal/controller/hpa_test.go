package controller

import (
	"context"
	"fmt"
	"testing"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUpdateVWA(t *testing.T) {
	tests := []struct {
		name                 string
		vwa                  *vwav1.VerticalWorkloadAutoscaler
		ignoreCPU            bool
		ignoreMemory         bool
		expectedUpdate       bool
		expectedIgnoreCPU    bool
		expectedIgnoreMemory bool
	}{
		{
			name: "No update needed",
			vwa: &vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					IgnoreCPURecommendations:    false,
					IgnoreMemoryRecommendations: false,
				},
			},
			ignoreCPU:            false,
			ignoreMemory:         false,
			expectedUpdate:       false,
			expectedIgnoreCPU:    false,
			expectedIgnoreMemory: false,
		},
		{
			name: "Update CPU recommendation",
			vwa: &vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					IgnoreCPURecommendations:    false,
					IgnoreMemoryRecommendations: false,
				},
			},
			ignoreCPU:            true,
			ignoreMemory:         false,
			expectedUpdate:       true,
			expectedIgnoreCPU:    true,
			expectedIgnoreMemory: false,
		},
		{
			name: "Update Memory recommendation",
			vwa: &vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					IgnoreCPURecommendations:    false,
					IgnoreMemoryRecommendations: false,
				},
			},
			ignoreCPU:            false,
			ignoreMemory:         true,
			expectedUpdate:       true,
			expectedIgnoreCPU:    false,
			expectedIgnoreMemory: true,
		},
		{
			name: "Update both recommendations",
			vwa: &vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					IgnoreCPURecommendations:    false,
					IgnoreMemoryRecommendations: false,
				},
			},
			ignoreCPU:            true,
			ignoreMemory:         true,
			expectedUpdate:       true,
			expectedIgnoreCPU:    true,
			expectedIgnoreMemory: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated := updateVWA(tt.vwa, tt.ignoreCPU, tt.ignoreMemory)
			assert.Equal(t, tt.expectedUpdate, updated)
			assert.Equal(t, tt.expectedIgnoreCPU, tt.vwa.Spec.IgnoreCPURecommendations)
			assert.Equal(t, tt.expectedIgnoreMemory, tt.vwa.Spec.IgnoreMemoryRecommendations)
		})
	}
}

func TestGetIgnoreFlags(t *testing.T) {
	tests := []struct {
		name                 string
		hpa                  *autoscalingv2.HorizontalPodAutoscaler
		expectedIgnoreCPU    bool
		expectedIgnoreMemory bool
	}{
		{
			name: "HPA with CPU metric",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.ResourceMetricSourceType,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: "cpu",
							},
						},
					},
				},
			},
			expectedIgnoreCPU:    true,
			expectedIgnoreMemory: false,
		},
		{
			name: "HPA with Memory metric",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.ResourceMetricSourceType,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: "memory",
							},
						},
					},
				},
			},
			expectedIgnoreCPU:    false,
			expectedIgnoreMemory: true,
		},
		{
			name: "HPA with both CPU and Memory metrics",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.ResourceMetricSourceType,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: "cpu",
							},
						},
						{
							Type: autoscalingv2.ResourceMetricSourceType,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: "memory",
							},
						},
					},
				},
			},
			expectedIgnoreCPU:    true,
			expectedIgnoreMemory: true,
		},
		{
			name: "HPA with no CPU or Memory metrics",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.PodsMetricSourceType,
						},
					},
				},
			},
			expectedIgnoreCPU:    false,
			expectedIgnoreMemory: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ignoreCPU, ignoreMemory := getIgnoreFlags(tt.hpa)
			assert.Equal(t, tt.expectedIgnoreCPU, ignoreCPU)
			assert.Equal(t, tt.expectedIgnoreMemory, ignoreMemory)
		})
	}
}

// fakeClientWithError is a mock client that returns an error on List
type fakeClientWithError struct {
	_client.Client
}

func (c *fakeClientWithError) List(ctx context.Context, list _client.ObjectList, opts ..._client.ListOption) error {
	return fmt.Errorf("simulated list error")
}

func (c *fakeClientWithError) Watch(ctx context.Context, obj _client.ObjectList, opts ..._client.ListOption) (watch.Interface, error) {
	return nil, fmt.Errorf("simulated watch error")
}

func TestFindMatchingVWA(t *testing.T) {
	tests := []struct {
		name        string
		hpa         *autoscalingv2.HorizontalPodAutoscaler
		vwaList     vwav1.VerticalWorkloadAutoscalerList
		expectedVWA *vwav1.VerticalWorkloadAutoscaler
		expectError bool
	}{
		{
			name: "Matching VWA found",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-hpa",
					Namespace: "default",
				},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "test-deployment",
					},
				},
			},
			vwaList: vwav1.VerticalWorkloadAutoscalerList{
				Items: []vwav1.VerticalWorkloadAutoscaler{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-vwa",
							Namespace: "default",
						},
						Status: vwav1.VerticalWorkloadAutoscalerStatus{
							ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
								Kind: "Deployment",
								Name: "test-deployment",
							},
						},
					},
				},
			},
			expectedVWA: &vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-vwa",
					Namespace:       "default",
					ResourceVersion: "1",
				},
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "test-deployment",
					},
				},
			},
			expectError: false,
		},
		{
			name: "No matching VWA found",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-hpa",
					Namespace: "default",
				},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "test-deployment",
					},
				},
			},
			vwaList: vwav1.VerticalWorkloadAutoscalerList{
				Items: []vwav1.VerticalWorkloadAutoscaler{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-vwa",
							Namespace: "default",
						},
						Status: vwav1.VerticalWorkloadAutoscalerStatus{
							ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
								Kind: "Deployment",
								Name: "another-deployment",
							},
						},
					},
				},
			},
			expectedVWA: nil,
			expectError: false,
		},
		{
			name: "Error listing VWAs",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-hpa",
					Namespace: "default",
				},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "test-deployment",
					},
				},
			},
			vwaList:     vwav1.VerticalWorkloadAutoscalerList{},
			expectedVWA: nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			vwav1.AddToScheme(scheme)
			autoscalingv2.AddToScheme(scheme)

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&vwav1.VerticalWorkloadAutoscaler{}).
				WithObjects(tt.hpa).
				Build()

			if tt.expectError {
				client = &fakeClientWithError{client}
			}

			for _, vwa := range tt.vwaList.Items {
				client.Create(context.Background(), &vwa)
			}

			r := &VerticalWorkloadAutoscalerReconciler{
				Client: client,
				Scheme: scheme,
			}

			vwa, err := r.findMatchingVWA(context.Background(), tt.hpa)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedVWA, vwa)
		})
	}
}

func TestShouldHandleHPA(t *testing.T) {
	tests := []struct {
		name     string
		hpa      *autoscalingv2.HorizontalPodAutoscaler
		expected bool
	}{
		{
			name: "HPA with CPU metric",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.ResourceMetricSourceType,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: "cpu",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "HPA with Memory metric",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.ResourceMetricSourceType,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: "memory",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "HPA with no CPU or Memory metric",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.PodsMetricSourceType,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "HPA with empty metrics",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					Metrics: []autoscalingv2.MetricSpec{},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldHandleHPA(tt.hpa)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleHPAUpdate(t *testing.T) {
	tests := []struct {
		name                 string
		hpa                  *autoscalingv2.HorizontalPodAutoscaler
		vwaList              vwav1.VerticalWorkloadAutoscalerList
		expectedIgnoreCPU    bool
		expectedIgnoreMemory bool
		expectUpdate         bool
	}{
		{
			name: "HPA with CPU metric",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-hpa",
					Namespace: "default",
				},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "test-deployment",
					},
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.ResourceMetricSourceType,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: "cpu",
							},
						},
					},
				},
			},
			vwaList: vwav1.VerticalWorkloadAutoscalerList{
				Items: []vwav1.VerticalWorkloadAutoscaler{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-vwa",
							Namespace: "default",
						},
						Status: vwav1.VerticalWorkloadAutoscalerStatus{
							ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
								Kind: "Deployment",
								Name: "test-deployment",
							},
						},
						Spec: vwav1.VerticalWorkloadAutoscalerSpec{
							IgnoreCPURecommendations:    false,
							IgnoreMemoryRecommendations: false,
						},
					},
				},
			},
			expectedIgnoreCPU:    true,
			expectedIgnoreMemory: false,
			expectUpdate:         true,
		},
		{
			name: "HPA with Memory metric",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-hpa",
					Namespace: "default",
				},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "test-deployment",
					},
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.ResourceMetricSourceType,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: "memory",
							},
						},
					},
				},
			},
			vwaList: vwav1.VerticalWorkloadAutoscalerList{
				Items: []vwav1.VerticalWorkloadAutoscaler{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-vwa",
							Namespace: "default",
						},
						Status: vwav1.VerticalWorkloadAutoscalerStatus{
							ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
								Kind: "Deployment",
								Name: "test-deployment",
							},
						},
						Spec: vwav1.VerticalWorkloadAutoscalerSpec{
							IgnoreCPURecommendations:    false,
							IgnoreMemoryRecommendations: false,
						},
					},
				},
			},
			expectedIgnoreCPU:    false,
			expectedIgnoreMemory: true,
			expectUpdate:         true,
		},
		{
			name: "HPA with no CPU or Memory metric",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-hpa",
					Namespace: "default",
				},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "test-deployment",
					},
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.PodsMetricSourceType,
						},
					},
				},
			},
			vwaList: vwav1.VerticalWorkloadAutoscalerList{
				Items: []vwav1.VerticalWorkloadAutoscaler{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-vwa",
							Namespace: "default",
						},
						Status: vwav1.VerticalWorkloadAutoscalerStatus{
							ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
								Kind: "Deployment",
								Name: "test-deployment",
							},
						},
						Spec: vwav1.VerticalWorkloadAutoscalerSpec{
							IgnoreCPURecommendations:    false,
							IgnoreMemoryRecommendations: false,
						},
					},
				},
			},
			expectedIgnoreCPU:    false,
			expectedIgnoreMemory: false,
			expectUpdate:         false,
		},
		{
			name: "HPA with DeletionTimestamp",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: timeNow()},
					Name:              "test-hpa",
					Namespace:         "default",
					Finalizers:        []string{"test-finalizer"},
				},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "test-deployment",
					},
					Metrics: []autoscalingv2.MetricSpec{
						{
							Type: autoscalingv2.ResourceMetricSourceType,
							Resource: &autoscalingv2.ResourceMetricSource{
								Name: "cpu",
							},
						},
					},
				},
			},
			vwaList: vwav1.VerticalWorkloadAutoscalerList{
				Items: []vwav1.VerticalWorkloadAutoscaler{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-vwa",
							Namespace: "default",
						},
						Status: vwav1.VerticalWorkloadAutoscalerStatus{
							ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
								Kind: "Deployment",
								Name: "test-deployment",
							},
						},
						Spec: vwav1.VerticalWorkloadAutoscalerSpec{
							IgnoreCPURecommendations:    true,
							IgnoreMemoryRecommendations: false,
						},
					},
				},
			},
			expectedIgnoreCPU:    false,
			expectedIgnoreMemory: false,
			expectUpdate:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			vwav1.AddToScheme(scheme)
			autoscalingv2.AddToScheme(scheme)

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&vwav1.VerticalWorkloadAutoscaler{}).
				WithObjects(tt.hpa).
				Build()

			for _, vwa := range tt.vwaList.Items {
				client.Create(context.Background(), &vwa)
			}

			r := &VerticalWorkloadAutoscalerReconciler{
				Client: client,
				Scheme: scheme,
			}

			// Create the HPA object without DeletionTimestamp if set; fakeClient does not support DeletionTimestamp
			client.Create(context.Background(), &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:       tt.hpa.Name,
					Namespace:  tt.hpa.Namespace,
					Finalizers: tt.hpa.Finalizers,
				},
				Spec: tt.hpa.Spec,
			})

			_, err := r.handleHPAUpdate(context.TODO(), tt.hpa)
			assert.NoError(t, err)

			var updatedVWA vwav1.VerticalWorkloadAutoscaler
			err = client.Get(context.TODO(), _client.ObjectKey{
				Namespace: tt.hpa.Namespace,
				Name:      tt.vwaList.Items[0].Name,
			}, &updatedVWA)
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedIgnoreCPU, updatedVWA.Spec.IgnoreCPURecommendations)
			assert.Equal(t, tt.expectedIgnoreMemory, updatedVWA.Spec.IgnoreMemoryRecommendations)
			if tt.expectUpdate {
				if tt.expectedIgnoreCPU != tt.vwaList.Items[0].Spec.IgnoreCPURecommendations {
					assert.NotEqual(t, tt.vwaList.Items[0].Spec.IgnoreCPURecommendations, updatedVWA.Spec.IgnoreCPURecommendations)
				}
				if tt.expectedIgnoreMemory != tt.vwaList.Items[0].Spec.IgnoreMemoryRecommendations {
					assert.NotEqual(t, tt.vwaList.Items[0].Spec.IgnoreMemoryRecommendations, updatedVWA.Spec.IgnoreMemoryRecommendations)
				}
			} else {
				assert.Equal(t, tt.vwaList.Items[0].Spec.IgnoreCPURecommendations, updatedVWA.Spec.IgnoreCPURecommendations)
				assert.Equal(t, tt.vwaList.Items[0].Spec.IgnoreMemoryRecommendations, updatedVWA.Spec.IgnoreMemoryRecommendations)
			}
		})
	}
}
