package controller

import (
	"context"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *VerticalWorkloadAutoscalerReconciler) handleHPAUpdate(ctx context.Context, hpa *autoscalingv2.HorizontalPodAutoscaler) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !shouldHandleHPA(hpa) {
		logger.Info("skipping HPA as it does not use CPU or memory metrics", "HPA", hpa.Name)
		return ctrl.Result{}, nil
	}

	vwa, err := r.findMatchingVWA(ctx, hpa)
	if err != nil {
		logger.Error(err, "failed to list VWAs", "HPA", hpa.Name)
		return ctrl.Result{}, nil
	}
	if vwa == nil {
		logger.Info("no matching VWA found for HPA", "HPA", hpa.Name)
		return ctrl.Result{}, nil
	}

	ignoreCPU, ignoreMemory := getIgnoreFlags(hpa)
	if !hpa.DeletionTimestamp.IsZero() {
		logger.Info("HPA is being deleted, resetting ignore flags", "HPA", hpa.Name)
		ignoreCPU = false
		ignoreMemory = false
	}

	if updateVWA(vwa, ignoreCPU, ignoreMemory) {
		logger.Info("updating VWA with HPA conflicts", "VWA", vwa.Name, "HPA", hpa.Name)
		if err = r.Update(ctx, vwa); err != nil {
			logger.Error(err, "Failed to update VWA", "VWA", vwa.Name, "HPA", hpa.Name)
			r.recordEvent(vwa, "Warning", "UpdateFailed", "Failed to update VWA with HPA conflicts")
			return ctrl.Result{}, nil
		}
		r.recordEvent(vwa, "Normal", "Updated", "Updated VWA with HPA CPU/Memory conflicts")
		r.updateStatusCondition(ctx, vwa, ConditionTypeReconciled, metav1.ConditionTrue, ReasonUpdatedResources, "updated VWA with HPA CPU/Memory conflicts") //nolint:errcheck
	}

	logger.Info("Successfully handled HPA update", "HPA", hpa.Name, "VWA", vwa.Name)
	return ctrl.Result{}, nil
}

func shouldHandleHPA(hpa *autoscalingv2.HorizontalPodAutoscaler) bool {
	for _, metric := range hpa.Spec.Metrics {
		if metric.Type == autoscalingv2.ResourceMetricSourceType {
			if metric.Resource.Name == "cpu" || metric.Resource.Name == "memory" {
				return true
			}
		}
	}
	return false
}

func (r *VerticalWorkloadAutoscalerReconciler) findMatchingVWA(ctx context.Context, hpa *autoscalingv2.HorizontalPodAutoscaler) (*vwav1.VerticalWorkloadAutoscaler, error) {
	var vwaList vwav1.VerticalWorkloadAutoscalerList
	if err := r.List(ctx, &vwaList, client.InNamespace(hpa.Namespace)); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	hpaTarget := hpa.Spec.ScaleTargetRef
	for _, wa := range vwaList.Items {
		if wa.Status.ScaleTargetRef.Kind == hpaTarget.Kind && wa.Status.ScaleTargetRef.Name == hpaTarget.Name {
			return &wa, nil
		}
	}
	return nil, nil
}

func getIgnoreFlags(hpa *autoscalingv2.HorizontalPodAutoscaler) (bool, bool) {
	ignoreCPU := false
	ignoreMemory := false
	for _, metric := range hpa.Spec.Metrics {
		if metric.Type == autoscalingv2.ResourceMetricSourceType {
			if metric.Resource.Name == "cpu" {
				ignoreCPU = true
			} else if metric.Resource.Name == "memory" {
				ignoreMemory = true
			}
		}
	}
	return ignoreCPU, ignoreMemory
}

func updateVWA(vwa *vwav1.VerticalWorkloadAutoscaler, ignoreCPU, ignoreMemory bool) bool {
	updateNeeded := false
	if vwa.Spec.IgnoreCPURecommendations != ignoreCPU {
		vwa.Spec.IgnoreCPURecommendations = ignoreCPU
		updateNeeded = true
	}
	if vwa.Spec.IgnoreMemoryRecommendations != ignoreMemory {
		vwa.Spec.IgnoreMemoryRecommendations = ignoreMemory
		updateNeeded = true
	}
	return updateNeeded
}
