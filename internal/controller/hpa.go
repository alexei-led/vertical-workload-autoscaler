package controller

import (
	"context"
	"fmt"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// listHPAs lists all HPAs in the same namespace as the target resource
func (r *VerticalWorkloadAutoscalerReconciler) listHPAs(ctx context.Context, namespace string) ([]autoscalingv2.HorizontalPodAutoscaler, error) {
	var hpaList autoscalingv2.HorizontalPodAutoscalerList
	if err := r.List(ctx, &hpaList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list HPAs in namespace %s: %w", namespace, err)
	}
	return hpaList.Items, nil
}

// detectHPAConflicts detects conflicts between HPAs and VPAs
func (r *VerticalWorkloadAutoscalerReconciler) detectHPAConflicts(ctx context.Context, targetResource client.Object) (map[string]bool, error) {
	conflicts := map[string]bool{"cpu": false, "memory": false}
	hpas, err := r.listHPAs(ctx, targetResource.GetNamespace())
	if err != nil {
		return conflicts, err
	}

	for _, hpa := range hpas {
		if hpa.Spec.ScaleTargetRef.Name == targetResource.GetName() && hpa.Spec.ScaleTargetRef.Kind == targetResource.GetObjectKind().GroupVersionKind().Kind {
			for _, metric := range hpa.Spec.Metrics {
				if metric.Type == autoscalingv2.ResourceMetricSourceType {
					if metric.Resource.Name == "cpu" {
						conflicts["cpu"] = true
					}
					if metric.Resource.Name == "memory" {
						conflicts["memory"] = true
					}
				}
			}
		}
	}
	return conflicts, nil
}

// updateStatusWithConflict updates the VerticalWorkloadAutoscaler status with conflict information
func (r *VerticalWorkloadAutoscalerReconciler) updateStatusWithConflict(ctx context.Context, wa vwav1.VerticalWorkloadAutoscaler, resource string) error {
	wa.Status.Conflicts = append(wa.Status.Conflicts, vwav1.Conflict{
		Resource:     resource,
		ConflictWith: "HorizontalPodAutoscaler",
		Reason:       fmt.Sprintf("HPA scales %s", resource),
	})
	if err := r.Status().Update(ctx, &wa); err != nil {
		return fmt.Errorf("failed to update VerticalWorkloadAutoscaler status with conflict: %w", err)
	}
	return nil
}
