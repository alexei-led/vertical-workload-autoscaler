package controller

import (
	"context"
	"time"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var timeNow = time.Now

// shouldDelayUpdate checks if the update should be delayed based on the configuration
func (r *VerticalWorkloadAutoscalerReconciler) shouldDelayUpdate(wa vwav1.VerticalWorkloadAutoscaler) (time.Duration, bool) {
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

func (r *VerticalWorkloadAutoscalerReconciler) updateStatus(ctx context.Context, wa *vwav1.VerticalWorkloadAutoscaler, newResources map[string]v1.ResourceRequirements) error {
	wa.Status.UpdateCount++
	wa.Status.LastUpdated = metav1.Now()
	wa.Status.CurrentStatus = "Updated"
	wa.Status.RecommendedRequests = newResources
	wa.Status.StepSize = wa.Spec.StepSize
	wa.Status.QualityOfService = wa.Spec.QualityOfService

	return r.Status().Update(ctx, wa)
}

func (r *VerticalWorkloadAutoscalerReconciler) updateStatusOnError(ctx context.Context, wa *vwav1.VerticalWorkloadAutoscaler, err error) {
	wa.Status.CurrentStatus = "Error"
	wa.Status.Errors = append(wa.Status.Errors, err.Error())
	r.Status().Update(ctx, wa) // nolint:errcheck
}
