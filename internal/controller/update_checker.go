package controller

import (
	"context"
	"time"

	autoscalingk8siov1alpha1 "github.com/alexei-led/workload-autoscaler/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var timeNow = time.Now

// shouldDelayUpdate checks if the update should be delayed based on the configuration
func (r *WorkloadAutoscalerReconciler) shouldDelayUpdate(wa autoscalingk8siov1alpha1.WorkloadAutoscaler) (time.Duration, bool) {
	now := timeNow()

	// If no allowed update windows are set, update immediately
	if len(wa.Spec.AllowedUpdateWindows) == 0 {
		return 0, false
	}

	for _, window := range wa.Spec.AllowedUpdateWindows {
		loc, err := time.LoadLocation(window.TimeZone)
		if err != nil {
			continue
		}
		start, err := time.ParseInLocation("15:04", window.StartTime, loc)
		if err != nil {
			continue
		}
		end, err := time.ParseInLocation("15:04", window.EndTime, loc)
		if err != nil {
			continue
		}
		if now.Weekday().String() == window.DayOfWeek && now.After(start) && now.Before(end) {
			return 0, false
		}
	}

	// Calculate the next allowed update time
	var nextUpdate time.Time
	for _, window := range wa.Spec.AllowedUpdateWindows {
		loc, _ := time.LoadLocation(window.TimeZone)
		start, _ := time.ParseInLocation("15:04", window.StartTime, loc)
		if now.Weekday().String() == window.DayOfWeek && now.Before(start) {
			nextUpdate = start
			break
		}
	}

	if nextUpdate.IsZero() {
		// No allowed update window today, find the next one
		for _, window := range wa.Spec.AllowedUpdateWindows {
			loc, _ := time.LoadLocation(window.TimeZone)
			start, _ := time.ParseInLocation("15:04", window.StartTime, loc)
			nextUpdate = start
			break
		}
	}

	return nextUpdate.Sub(now), true
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
