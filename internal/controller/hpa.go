package controller

import (
	"context"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *VerticalWorkloadAutoscalerReconciler) findVWAForHPA(ctx context.Context, hpa client.Object) []reconcile.Request {
	requests := make([]reconcile.Request, 0)
	hpaObj, ok := hpa.(*autoscalingv2.HorizontalPodAutoscaler)
	if !ok {
		return requests
	}
	// Create a list to store matching VWAs
	var vwaList vwav1.VerticalWorkloadAutoscalerList

	// Fetch all VWAs that reference the same scale target as the HPA using the index
	if err := r.List(ctx, &vwaList,
		client.InNamespace(hpaObj.Namespace),
		client.MatchingFields{
			statusScaleTargetRefName: hpaObj.Spec.ScaleTargetRef.Name,
			statusScaleTargetRefKind: hpaObj.Spec.ScaleTargetRef.Kind,
		}); err != nil {
		log.Log.Error(err, "failed to list VerticalWorkloadAutoscaler objects")
		return requests
	}

	// Create reconcile requests for all matched VWAs
	for _, vwa := range vwaList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: vwa.Namespace,
				Name:      vwa.Name,
			},
		})
	}

	return requests
}

func (r *VerticalWorkloadAutoscalerReconciler) findHPAForVWA(ctx context.Context, vwa *vwav1.VerticalWorkloadAutoscaler) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	var hpaList autoscalingv2.HorizontalPodAutoscalerList

	// Use client.MatchingFields to search for HPAs with the same scale target reference
	if err := r.List(ctx, &hpaList,
		client.InNamespace(vwa.Namespace),
		client.MatchingFields{
			hpaSpecScaleTargetRefName: vwa.Status.ScaleTargetRef.Name,
			hpaSpecScaleTargetRefKind: vwa.Status.ScaleTargetRef.Kind,
		}); err != nil {
		return nil, err
	}

	if len(hpaList.Items) > 0 {
		// Assuming only one HPA matches (if there could be more, handle that case)
		return &hpaList.Items[0], nil
	}

	return nil, errors.NewNotFound(autoscalingv2.Resource("horizontalpodautoscalers"), "no matching HPA found")
}

func (r *VerticalWorkloadAutoscalerReconciler) getIgnoreFlags(hpa *autoscalingv2.HorizontalPodAutoscaler) (bool, bool) {
	ignoreCPU := false
	ignoreMemory := false
	for _, metric := range hpa.Spec.Metrics {
		if metric.Type == autoscalingv2.ResourceMetricSourceType {
			if metric.Resource.Name == corev1.ResourceCPU {
				ignoreCPU = true
			} else if metric.Resource.Name == corev1.ResourceMemory {
				ignoreMemory = true
			}
		}
	}
	return ignoreCPU, ignoreMemory
}
