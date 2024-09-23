package controller

import (
	"context"
	"reflect"

	autoscalingk8siov1alpha1 "github.com/alexei-led/workload-autoscaler/api/v1alpha1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// VPARecommendationChangedPredicate is a custom predicate to watch for changes in VPA recommendations
type VPARecommendationChangedPredicate struct {
	predicate.Funcs
}

func (VPARecommendationChangedPredicate) Update(e event.UpdateEvent) bool {
	oldVPA, okOld := e.ObjectOld.(*vpav1.VerticalPodAutoscaler)
	newVPA, okNew := e.ObjectNew.(*vpav1.VerticalPodAutoscaler)
	if !okOld || !okNew {
		return false
	}
	return !reflect.DeepEqual(oldVPA.Status.Recommendation, newVPA.Status.Recommendation)
}

func (r *WorkloadAutoscalerReconciler) fetchVPA(ctx context.Context, wa autoscalingk8siov1alpha1.WorkloadAutoscaler) (*vpav1.VerticalPodAutoscaler, error) {
	var vpa vpav1.VerticalPodAutoscaler
	if err := r.Get(ctx, client.ObjectKey{Name: wa.Spec.VPAReference.Name, Namespace: wa.Namespace}, &vpa); err != nil {
		return nil, err
	}
	return &vpa, nil
}
