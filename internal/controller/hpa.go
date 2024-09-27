package controller

import (
	"context"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *VerticalWorkloadAutoscalerReconciler) handleHPAUpdate(ctx context.Context, hpa *autoscalingv2.HorizontalPodAutoscaler) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// fetch target resource of the HPA
	hpaTarget := hpa.Spec.ScaleTargetRef

	// check HPA metrics (CPU/Memory)
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

	// if metrics are not based on CPU or Memory, nothing to do
	if !ignoreCPU && !ignoreMemory {
		return ctrl.Result{}, nil
	}

	// fetch the corresponding VWA object
	var vwaList vwav1.VerticalWorkloadAutoscalerList
	if err := r.List(ctx, &vwaList, client.InNamespace(hpa.Namespace)); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info("no VWA found for the HPA target", "hpa", hpa.Name)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to list VWAs")
		return ctrl.Result{}, nil
	}

	// find VWA that references the same target as the HPA
	var vwa *vwav1.VerticalWorkloadAutoscaler
	for _, wa := range vwaList.Items {
		currentWA := wa
		if currentWA.Status.ScaleTargetRef.Kind == hpaTarget.Kind && currentWA.Status.ScaleTargetRef.Name == hpaTarget.Name {
			vwa = &currentWA
			break
		}
	}

	// no matching VWA, nothing to do
	if vwa == nil {
		return ctrl.Result{}, nil
	}

	// if the HPA is going to be deleted, reset the ignore properties
	// the update event will be triggered before the delete event because Kubernetes will add DeletionTimestamp before deleting an object
	if !hpa.DeletionTimestamp.IsZero() {
		ignoreCPU = false
		ignoreMemory = false
	}

	// update the VWA Object’s ignore properties
	updateNeeded := false
	if vwa.Spec.IgnoreCPURecommendations != ignoreCPU {
		vwa.Spec.IgnoreCPURecommendations = ignoreCPU
		updateNeeded = true
	}
	if vwa.Spec.IgnoreMemoryRecommendations != ignoreMemory {
		vwa.Spec.IgnoreMemoryRecommendations = ignoreMemory
		updateNeeded = true
	}

	if updateNeeded {
		// update the VWA object
		if err := r.Update(ctx, vwa); err != nil {
			logger.Error(err, "failed to update VWA with HPA conflicts")
			return ctrl.Result{}, nil
		}
	}

	logger.Info("successfully handled HPA update")
	return ctrl.Result{}, nil
}
