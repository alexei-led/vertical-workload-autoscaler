package controller

import (
	"context"

	autoscalingk8siov1alpha1 "github.com/alexei-led/workload-autoscaler/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (r *WorkloadAutoscalerReconciler) isUpdateNeeded(wa autoscalingk8siov1alpha1.WorkloadAutoscaler, recommendations autoscalingk8siov1alpha1.Recommendation) bool {
	return wa.Status.CurrentRequests.CPU != recommendations.CPU
}

func (r *WorkloadAutoscalerReconciler) recordProgress(ctx context.Context, wa autoscalingk8siov1alpha1.WorkloadAutoscaler) error {
	wa.Status.UpdateCount++
	if err := r.Status().Update(ctx, &wa); err != nil {
		return err
	}
	return nil
}

func (r *WorkloadAutoscalerReconciler) updateStatus(ctx context.Context, wa autoscalingk8siov1alpha1.WorkloadAutoscaler) error {
	wa.Status.LastUpdated = metav1.Now()
	if err := r.Status().Update(ctx, &wa); err != nil {
		return err
	}
	return nil
}
