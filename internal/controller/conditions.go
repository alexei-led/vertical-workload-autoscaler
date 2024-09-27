package controller

import (
	"context"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ConditionTypeReady is the condition type for readiness
	ConditionTypeReady = "Ready"
	// ConditionTypeWarning is the condition type for warning
	ConditionTypeWarning = "Warning"
	// ConditionTypeError is the condition type for error
	ConditionTypeError = "Error"
	// ConditionTypeReconciled is the condition type for reconciliation
	ConditionTypeReconciled = "Reconciled"
	// ReasonVPAReferenceConflict is the condition reason for VPA reference conflict
	ReasonVPAReferenceConflict = "VPAReferenceConflict"
	// ReasonVPAReferenceNotFound is the condition reason for VPA reference not found
	ReasonVPAReferenceNotFound = "VPAReferenceNotFound"
	// ReasonTargetObjectNotFound is the condition reason for target object not found
	ReasonTargetObjectNotFound = "TargetObjectNotFound"
	// ReasonAPIError is the condition reason for Kubernetes API error
	ReasonAPIError = "APIError"
)

// updateStatusCondition updates the VWA status with a new condition
//
//nolint:unparam
func (r *VerticalWorkloadAutoscalerReconciler) updateStatusCondition(ctx context.Context, wa *vwav1.VerticalWorkloadAutoscaler, conditionType string, status metav1.ConditionStatus, reason, message string) error {
	// Create a new condition
	newCondition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Update the VWA status with the new condition
	conditions := wa.Status.Conditions
	existingCondition := findCondition(conditions, conditionType)

	if existingCondition == nil {
		// If the condition doesn't exist, add it
		wa.Status.Conditions = append(wa.Status.Conditions, newCondition)
	} else if *existingCondition != newCondition {
		// Update existing condition if something has changed
		*existingCondition = newCondition
	}

	return r.Status().Update(ctx, wa)
}

// findCondition helps to find an existing condition in the conditions array
func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

func (r *VerticalWorkloadAutoscalerReconciler) updateStatus(ctx context.Context, wa *vwav1.VerticalWorkloadAutoscaler, newResources map[string]corev1.ResourceRequirements) error {
	now := metav1.NewTime(timeNow())
	wa.Status.LastUpdated = &now
	wa.Status.UpdateCount++
	wa.Status.RecommendedRequests = newResources

	// Example logic to determine if updates were skipped
	if len(newResources) == 0 {
		wa.Status.SkippedUpdates = true
		wa.Status.SkipReason = "No new resource recommendations"
	} else {
		wa.Status.SkippedUpdates = false
		wa.Status.SkipReason = ""
	}

	// Update conditions
	return r.updateStatusCondition(ctx, wa, ConditionTypeReconciled, metav1.ConditionTrue, "Reconciled", "VWA reconciled successfully")
}
