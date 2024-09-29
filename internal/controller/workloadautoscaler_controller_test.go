package controller

import (
	"context"
	"fmt"
	"testing"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

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

			requests := r.findObjectsForVPA(context.Background(), vpa)
			assert.Equal(t, tt.expected, requests)
		})
	}
}

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
