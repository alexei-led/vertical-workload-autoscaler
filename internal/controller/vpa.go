package controller

import (
	"context"
	"reflect"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

func (r *VerticalWorkloadAutoscalerReconciler) fetchVPA(ctx context.Context, wa vwav1.VerticalWorkloadAutoscaler) (*vpav1.VerticalPodAutoscaler, error) {
	var vpa vpav1.VerticalPodAutoscaler
	if err := r.Get(ctx, client.ObjectKey{Name: wa.Spec.VPAReference.Name, Namespace: wa.Namespace}, &vpa); err != nil {
		return nil, err
	}
	return &vpa, nil
}

func (r *VerticalWorkloadAutoscalerReconciler) handleVPAUpdate(ctx context.Context, vpa *vpav1.VerticalPodAutoscaler) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the associated VWA object
	var waList vwav1.VerticalWorkloadAutoscalerList
	if err := r.List(ctx, &waList, client.InNamespace(vpa.Namespace)); err != nil {
		logger.Error(err, "failed to list VerticalWorkloadAutoscaler objects")
		return ctrl.Result{}, nil
	}

	for _, wa := range waList.Items {
		if wa.Spec.VPAReference.Name == vpa.Name {
			// Requeue the VWA for reconciliation
			return r.Reconcile(ctx, ctrl.Request{
				NamespacedName: client.ObjectKey{
					Name:      wa.Name,
					Namespace: wa.Namespace,
				},
			})
		}
	}

	logger.Info("no associated VerticalWorkloadAutoscaler found for VPA", "VPA", vpa.Name)
	return ctrl.Result{}, nil
}
