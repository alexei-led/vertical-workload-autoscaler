package controller

import (
	"context"
	"testing"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestVPARecommendationChangedPredicate_Update(t *testing.T) {
	oldVPA := &vpav1.VerticalPodAutoscaler{
		Status: vpav1.VerticalPodAutoscalerStatus{
			Recommendation: &vpav1.RecommendedPodResources{
				ContainerRecommendations: []vpav1.RecommendedContainerResources{
					{
						ContainerName: "test-container",
						Target: corev1.ResourceList{
							"cpu":    resource.MustParse("100m"),
							"memory": resource.MustParse("200Mi"),
						},
					},
				},
			},
		},
	}

	newVPA := &vpav1.VerticalPodAutoscaler{
		Status: vpav1.VerticalPodAutoscalerStatus{
			Recommendation: &vpav1.RecommendedPodResources{
				ContainerRecommendations: []vpav1.RecommendedContainerResources{
					{
						ContainerName: "test-container",
						Target: corev1.ResourceList{
							"cpu":    resource.MustParse("200m"),
							"memory": resource.MustParse("400Mi"),
						},
					},
				},
			},
		},
	}

	pred := VPARecommendationChangedPredicate{}

	updateEvent := event.UpdateEvent{
		ObjectOld: oldVPA,
		ObjectNew: newVPA,
	}

	assert.True(t, pred.Update(updateEvent), "Expected predicate to return true for changed VPA recommendation")
}

func TestFetchVPA(t *testing.T) {
	ctx := context.TODO()
	s := runtime.NewScheme()
	s.AddKnownTypes(vpav1.SchemeGroupVersion, &vpav1.VerticalPodAutoscaler{})

	vpa := &vpav1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vpa",
			Namespace: "default",
		},
	}

	client := fake.NewClientBuilder().WithScheme(s).WithObjects(vpa).Build()
	r := &VerticalWorkloadAutoscalerReconciler{
		Client: client,
		Scheme: s,
	}

	wa := vwav1.VerticalWorkloadAutoscaler{
		Spec: vwav1.VerticalWorkloadAutoscalerSpec{
			VPAReference: vwav1.VPAReference{
				Name: "test-vpa",
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
		},
	}

	fetchedVPA, err := r.fetchVPA(ctx, wa)
	assert.NoError(t, err, "Expected no error fetching VPA")
	assert.Equal(t, vpa, fetchedVPA, "Expected fetched VPA to match the created VPA")
}

func TestHandleVPAUpdate(t *testing.T) {
	ctx := context.TODO()
	s := runtime.NewScheme()
	s.AddKnownTypes(vpav1.SchemeGroupVersion, &vpav1.VerticalPodAutoscaler{})
	s.AddKnownTypes(vwav1.SchemeGroupVersion, &vwav1.VerticalWorkloadAutoscaler{})
	s.AddKnownTypes(appsv1.SchemeGroupVersion, &appsv1.Deployment{})

	vpa := &vpav1.VerticalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vpa",
			Namespace: "default",
		},
		Spec: vpav1.VerticalPodAutoscalerSpec{
			TargetRef: &autoscalingv1.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "test-deployment",
			},
		},
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("200Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("200m"),
									"memory": resource.MustParse("400Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	wa := &vwav1.VerticalWorkloadAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-wa",
			Namespace: "default",
		},
		Spec: vwav1.VerticalWorkloadAutoscalerSpec{
			VPAReference: vwav1.VPAReference{
				Name: "test-vpa",
			},
		},
		Status: vwav1.VerticalWorkloadAutoscalerStatus{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "test-deployment",
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(s).WithObjects(vpa, wa, deployment).Build()
	r := &VerticalWorkloadAutoscalerReconciler{
		Client: client,
		Scheme: s,
	}

	tests := []struct {
		name          string
		vpa           *vpav1.VerticalPodAutoscaler
		expectedError bool
	}{
		{
			name:          "VPA with associated VWA",
			vpa:           vpa,
			expectedError: false,
		},
		{
			name: "VPA with no associated VWA",
			vpa: &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-existent-vpa",
					Namespace: "default",
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.handleVPAUpdate(ctx, tt.vpa)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, ctrl.Result{}, result)
		})
	}
}
