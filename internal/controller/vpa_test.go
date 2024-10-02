package controller

import (
	"context"
	"testing"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

func TestFindObjectsForVPA(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = vwav1.AddToScheme(scheme)
	_ = vpav1.AddToScheme(scheme)

	tests := []struct {
		name     string
		vpaName  string
		vwaList  []vwav1.VerticalWorkloadAutoscaler
		expected []reconcile.Request
	}{
		{
			name:    "VPA referenced by one VWA",
			vpaName: "vpa1",
			vwaList: []vwav1.VerticalWorkloadAutoscaler{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
					Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
				},
			},
			expected: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: "vwa1", Namespace: "default"}},
			},
		},
		{
			name:    "VPA not referenced by any VWA",
			vpaName: "vpa2",
			vwaList: []vwav1.VerticalWorkloadAutoscaler{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
					Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
				},
			},
			expected: []reconcile.Request{},
		},
		{
			name:    "Multiple VWAs referencing the same VPA",
			vpaName: "vpa1",
			vwaList: []vwav1.VerticalWorkloadAutoscaler{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vwa1", Namespace: "default"},
					Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "vwa2", Namespace: "default"},
					Spec:       vwav1.VerticalWorkloadAutoscalerSpec{VPAReference: vwav1.VPAReference{Name: "vpa1"}},
				},
			},
			expected: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: "vwa1", Namespace: "default"}},
				{NamespacedName: types.NamespacedName{Name: "vwa2", Namespace: "default"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vpa := &vpav1.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: tt.vpaName, Namespace: "default"},
			}
			objs := []_client.Object{vpa}
			for _, vwa := range tt.vwaList {
				objs = append(objs, &vwa)
			}
			client := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&vwav1.VerticalWorkloadAutoscaler{}).WithObjects(objs...).Build()
			r := &VerticalWorkloadAutoscalerReconciler{Client: client}

			requests := r.findVWAForVPA(context.Background(), vpa)
			assert.Equal(t, tt.expected, requests)
		})
	}
}
