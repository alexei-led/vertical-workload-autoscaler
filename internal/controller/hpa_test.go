package controller

import (
	"context"
	"testing"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestGetIgnoreFlags(t *testing.T) {
	tests := []struct {
		name           string
		hpa            *autoscalingv2.HorizontalPodAutoscaler
		expectedCPU    bool
		expectedMemory bool
	}{
		{
			name: "No metrics",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "hpa1", Namespace: "default"},
				Spec:       autoscalingv2.HorizontalPodAutoscalerSpec{},
			},
			expectedCPU:    false,
			expectedMemory: false,
		},
		{
			name: "CPU metric present",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "hpa2", Namespace: "default"},
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
			expectedCPU:    true,
			expectedMemory: false,
		},
		{
			name: "Memory metric present",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "hpa3", Namespace: "default"},
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
			expectedCPU:    false,
			expectedMemory: true,
		},
		{
			name: "Both CPU and Memory metrics present",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "hpa4", Namespace: "default"},
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
			expectedCPU:    true,
			expectedMemory: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &VerticalWorkloadAutoscalerReconciler{
				Client: fake.NewClientBuilder().WithStatusSubresource(&autoscalingv2.HorizontalPodAutoscaler{}).Build(),
			}
			ignoreCPU, ignoreMemory := r.getIgnoreFlags(tt.hpa)
			assert.Equal(t, tt.expectedCPU, ignoreCPU)
			assert.Equal(t, tt.expectedMemory, ignoreMemory)
		})
	}
}

func TestFindHPAForVWA(t *testing.T) {
	s := runtime.NewScheme()
	_ = vwav1.AddToScheme(s)
	_ = autoscalingv2.AddToScheme(s)

	tests := []struct {
		name          string
		vwa           *vwav1.VerticalWorkloadAutoscaler
		hpaList       *autoscalingv2.HorizontalPodAutoscalerList
		expectedHPA   *autoscalingv2.HorizontalPodAutoscaler
		expectedError bool
	}{
		{
			name: "Matching HPA found",
			vwa: &vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vwa",
					Namespace: "default",
				},
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Name: "test-deployment",
						Kind: "Deployment",
					},
				},
			},
			hpaList: &autoscalingv2.HorizontalPodAutoscalerList{
				Items: []autoscalingv2.HorizontalPodAutoscaler{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-hpa",
							Namespace: "default",
						},
						Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
							ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
								Name: "test-deployment",
								Kind: "Deployment",
							},
						},
					},
				},
			},
			expectedHPA: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-hpa",
					Namespace:       "default",
					ResourceVersion: "999",
				},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Name: "test-deployment",
						Kind: "Deployment",
					},
				},
			},
			expectedError: false,
		},
		{
			name: "No matching HPA found",
			vwa: &vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vwa",
					Namespace: "default",
				},
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Name: "test-deployment",
						Kind: "Deployment",
					},
				},
			},
			hpaList: &autoscalingv2.HorizontalPodAutoscalerList{
				Items: []autoscalingv2.HorizontalPodAutoscaler{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-hpa",
							Namespace: "default",
						},
						Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
							ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
								Name: "another-deployment",
								Kind: "Deployment",
							},
						},
					},
				},
			},
			expectedHPA:   nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(s).
				WithRuntimeObjects(tt.hpaList).
				WithIndex(&autoscalingv2.HorizontalPodAutoscaler{}, hpaSpecScaleTargetRefName, func(obj _client.Object) []string {
					hpa := obj.(*autoscalingv2.HorizontalPodAutoscaler)
					return []string{hpa.Spec.ScaleTargetRef.Name}
				}).
				WithIndex(&autoscalingv2.HorizontalPodAutoscaler{}, hpaSpecScaleTargetRefKind, func(obj _client.Object) []string {
					hpa := obj.(*autoscalingv2.HorizontalPodAutoscaler)
					return []string{hpa.Spec.ScaleTargetRef.Kind}
				}).Build()
			r := &VerticalWorkloadAutoscalerReconciler{
				Client: client,
			}

			hpa, err := r.findHPAForVWA(context.TODO(), tt.vwa)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedHPA, hpa)
			}
		})
	}
}

func TestFindVWAForHPA(t *testing.T) {
	s := runtime.NewScheme()
	_ = vwav1.AddToScheme(s)
	_ = autoscalingv2.AddToScheme(s)

	tests := []struct {
		name         string
		hpa          *autoscalingv2.HorizontalPodAutoscaler
		vwaList      *vwav1.VerticalWorkloadAutoscalerList
		expectedReqs []reconcile.Request
	}{
		{
			name: "Matching VWA found",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "test-hpa", Namespace: "default"},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Name: "test-deployment",
						Kind: "Deployment",
					},
				},
			},
			vwaList: &vwav1.VerticalWorkloadAutoscalerList{
				Items: []vwav1.VerticalWorkloadAutoscaler{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "test-vwa", Namespace: "default"},
						Status: vwav1.VerticalWorkloadAutoscalerStatus{
							ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
								Name: "test-deployment",
								Kind: "Deployment",
							},
						},
					},
				},
			},
			expectedReqs: []reconcile.Request{
				{
					NamespacedName: _client.ObjectKey{Name: "test-vwa", Namespace: "default"},
				},
			},
		},
		{
			name: "No matching VWA found",
			hpa: &autoscalingv2.HorizontalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "test-hpa", Namespace: "default"},
				Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
						Name: "test-deployment",
						Kind: "Deployment",
					},
				},
			},
			vwaList: &vwav1.VerticalWorkloadAutoscalerList{
				Items: []vwav1.VerticalWorkloadAutoscaler{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "another-vwa", Namespace: "default"},
						Status: vwav1.VerticalWorkloadAutoscalerStatus{
							ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
								Name: "another-deployment",
								Kind: "Deployment",
							},
						},
					},
				},
			},
			expectedReqs: []reconcile.Request{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(s).
				WithRuntimeObjects(tt.vwaList).
				WithIndex(&vwav1.VerticalWorkloadAutoscaler{}, statusScaleTargetRefName, func(obj _client.Object) []string {
					vwa := obj.(*vwav1.VerticalWorkloadAutoscaler)
					return []string{vwa.Status.ScaleTargetRef.Name}
				}).
				WithIndex(&vwav1.VerticalWorkloadAutoscaler{}, statusScaleTargetRefKind, func(obj _client.Object) []string {
					vwa := obj.(*vwav1.VerticalWorkloadAutoscaler)
					return []string{vwa.Status.ScaleTargetRef.Kind}
				}).
				Build()

			r := &VerticalWorkloadAutoscalerReconciler{
				Client: client,
			}

			reqs := r.findVWAForHPA(context.TODO(), tt.hpa)
			assert.Equal(t, tt.expectedReqs, reqs)
		})
	}
}
