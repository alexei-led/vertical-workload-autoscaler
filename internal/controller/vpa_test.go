package controller

import (
	"context"
	"testing"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/kubernetes/scheme"
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
	s := scheme.Scheme
	_ = vpav1.AddToScheme(s)

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
				Name:      "test-vpa",
				Namespace: "default",
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
