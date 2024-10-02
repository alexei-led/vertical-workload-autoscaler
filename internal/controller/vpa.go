package controller

import (
	"context"
	"reflect"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

func (r *VerticalWorkloadAutoscalerReconciler) fetchVPA(ctx context.Context, wa vwav1.VerticalWorkloadAutoscaler) (*vpav1.VerticalPodAutoscaler, error) {
	var vpa vpav1.VerticalPodAutoscaler
	if err := r.Get(ctx, client.ObjectKey{Name: wa.Spec.VPAReference.Name, Namespace: wa.Namespace}, &vpa); err != nil {
		return nil, err
	}
	return &vpa, nil
}

func (r *VerticalWorkloadAutoscalerReconciler) findVWAForVPA(_ context.Context, vpa client.Object) []reconcile.Request {
	requests := make([]reconcile.Request, 0)
	vpaObj, ok := vpa.(*vpav1.VerticalPodAutoscaler)
	if !ok {
		return requests
	}

	var vwaList vwav1.VerticalWorkloadAutoscalerList
	if err := r.List(context.Background(), &vwaList); err != nil {
		log.Log.Error(err, "failed to list VerticalWorkloadAutoscaler objects")
		return requests
	}

	for _, vwa := range vwaList.Items {
		if vwa.Spec.VPAReference.Name == vpaObj.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: vwa.Namespace,
					Name:      vwa.Name,
				},
			})
		}
	}
	return requests
}
