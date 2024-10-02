package controller

import (
	"context"
	"fmt"
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
	"k8s.io/apimachinery/pkg/types"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureNoDuplicateVWA(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = vwav1.AddToScheme(scheme)

	tests := []struct {
		name     string
		vwa      vwav1.VerticalWorkloadAutoscaler
		vwaList  []vwav1.VerticalWorkloadAutoscaler
		expected error
	}{
		{
			name: "No duplicate VWA",
			vwa: vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
				Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
			},
			vwaList: []vwav1.VerticalWorkloadAutoscaler{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vwa2", Namespace: "default"},
					Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa2"}},
				},
			},
			expected: nil,
		},
		{
			name: "Duplicate VWA exists",
			vwa: vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
				Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
			},
			vwaList: []vwav1.VerticalWorkloadAutoscaler{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vwa2", Namespace: "default"},
					Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
				},
			},
			expected: fmt.Errorf("VPA 'vpa1' is already referenced by another VWA object 'vwa2'"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []_client.Object{}
			for _, vwa := range tt.vwaList {
				objs = append(objs, &vwa)
			}
			client := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&vwav1.VerticalWorkloadAutoscaler{}).WithObjects(objs...).Build()
			r := &VerticalWorkloadAutoscalerReconciler{Client: client}

			err := r.ensureNoDuplicateVWA(context.Background(), &tt.vwa)
			assert.Equal(t, tt.expected, err)
		})
	}
}

func TestHandleVWAChange(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = vwav1.AddToScheme(scheme)
	_ = vpav1.AddToScheme(scheme)
	_ = autoscalingv2.AddToScheme(scheme)
	_ = autoscalingv1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	tests := []struct {
		name                string
		vwa                 vwav1.VerticalWorkloadAutoscaler
		vwa2                *vwav1.VerticalWorkloadAutoscaler
		vpa                 *vpav1.VerticalPodAutoscaler
		hpa                 *autoscalingv2.HorizontalPodAutoscaler
		deployment          *appsv1.Deployment
		updatedRequirements map[string]corev1.ResourceRequirements
		expected            error
	}{
		{
			name: "No duplicate VWA",
			vwa: vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
				Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
			},
			vwa2: &vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vwa2", Namespace: "default"},
				Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa2"}},
			},
			expected: nil,
		},
		{
			name: "Duplicate VWA exists",
			vwa: vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
				Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
			},
			vwa2: &vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vwa2", Namespace: "default"},
				Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
			},
			expected: fmt.Errorf("VPA 'vpa1' is already referenced by another VWA object 'vwa2'"),
		},
		{
			name: "VPA not found",
			vwa: vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
				Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
			},
			expected: nil,
		},
		{
			name: "VPA has no recommendations",
			vwa: vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
				Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
			},
			vpa: &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vpa1", Namespace: "default"},
				Spec: vpav1.VerticalPodAutoscalerSpec{
					TargetRef: &autoscalingv1.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "deployment1",
					},
				},
			},
			expected: nil,
		},
		{
			name: "Target object not found",
			vwa: vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
				Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
			},
			vpa: &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vpa1", Namespace: "default"},
				Spec: vpav1.VerticalPodAutoscalerSpec{
					TargetRef: &autoscalingv1.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "deployment1",
					},
				},
				Status: vpav1.VerticalPodAutoscalerStatus{
					Recommendation: &vpav1.RecommendedPodResources{},
				},
			},
			expected: nil,
		},
		{
			name: "Update Deployment (Guaranteed) according to VPA recommendations",
			vwa: vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
				Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
			},
			vpa: &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vpa1", Namespace: "default"},
				Spec: vpav1.VerticalPodAutoscalerSpec{
					TargetRef: &autoscalingv1.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "deployment1",
					},
				},
				Status: vpav1.VerticalPodAutoscalerStatus{
					Recommendation: &vpav1.RecommendedPodResources{
						ContainerRecommendations: []vpav1.RecommendedContainerResources{
							{
								ContainerName: "container1",
								Target: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
							{
								ContainerName: "container2",
								Target: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
							},
						},
					},
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "deployment1", Namespace: "default"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "container1",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("250m"),
											corev1.ResourceMemory: resource.MustParse("128Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("250m"),
											corev1.ResourceMemory: resource.MustParse("128Mi"),
										},
									},
								},
								{
									Name: "container2",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("500m"),
											corev1.ResourceMemory: resource.MustParse("640Mi"),
										},
									},
								},
							},
						},
					},
				},
			},
			updatedRequirements: map[string]corev1.ResourceRequirements{
				"container1": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
				"container2": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("64Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("64Mi"),
					},
				},
			},
		},
		{
			name: "Update Deployment (Burstable) according to VPA recommendations",
			vwa: vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					VPAReference:     vwav1.VPAReference{Name: "vpa1"},
					QualityOfService: vwav1.BurstableQualityOfService,
				},
			},
			vpa: &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "vpa1", Namespace: "default"},
				Spec: vpav1.VerticalPodAutoscalerSpec{
					TargetRef: &autoscalingv1.CrossVersionObjectReference{
						Kind: "Deployment",
						Name: "deployment1",
					},
				},
				Status: vpav1.VerticalPodAutoscalerStatus{
					Recommendation: &vpav1.RecommendedPodResources{
						ContainerRecommendations: []vpav1.RecommendedContainerResources{
							{
								ContainerName: "container1",
								Target: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("50m"),
									corev1.ResourceMemory: resource.MustParse("100Mi"),
								},
								LowerBound: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("10m"),
									corev1.ResourceMemory: resource.MustParse("50Mi"),
								},
								UpperBound: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("200Mi"),
								},
							},
						},
					},
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "deployment1", Namespace: "default"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "container1",
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("250m"),
											corev1.ResourceMemory: resource.MustParse("128Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("500m"),
											corev1.ResourceMemory: resource.MustParse("1Gi"),
										},
									},
								},
							},
						},
					},
				},
			},
			updatedRequirements: map[string]corev1.ResourceRequirements{
				"container1": {
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("10m"),
						corev1.ResourceMemory: resource.MustParse("50Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("200Mi"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := []_client.Object{&tt.vwa}
			if tt.vwa2 != nil {
				objs = append(objs, tt.vwa2)
			}
			if tt.vpa != nil {
				objs = append(objs, tt.vpa)
			}
			if tt.hpa != nil {
				objs = append(objs, tt.hpa)
			}
			if tt.deployment != nil {
				objs = append(objs, tt.deployment)
			}
			client := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&vwav1.VerticalWorkloadAutoscaler{}).WithObjects(objs...).Build()
			r := &VerticalWorkloadAutoscalerReconciler{Client: client}

			result, err := r.handleVWAChange(context.Background(), &tt.vwa)
			assert.Equal(t, tt.expected, err)
			assert.Equal(t, ctrl.Result{}, result)

			if tt.deployment != nil {
				updatedDeployment := &appsv1.Deployment{}
				err = client.Get(context.Background(), types.NamespacedName{Name: "deployment1", Namespace: "default"}, updatedDeployment)
				assert.NoError(t, err)

				for containerName, expectedResources := range tt.updatedRequirements {
					for _, container := range updatedDeployment.Spec.Template.Spec.Containers {
						if container.Name == containerName {
							assert.Equal(t, expectedResources, container.Resources)
						}
					}
				}
			}
		})
	}
}
