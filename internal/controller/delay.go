package controller

import (
	"time"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
)

var (
	timeNow = time.Now
)

// shouldDelayUpdateWindow checks if the current time is within the allowed update window
func (r *VerticalWorkloadAutoscalerReconciler) shouldDelayUpdateWindow(wa vwav1.VerticalWorkloadAutoscaler) (time.Duration, bool) {
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

// shouldDelayUpdateFrequency checks if the update frequency has been reached
func (r *VerticalWorkloadAutoscalerReconciler) shouldDelayUpdateFrequency(wa vwav1.VerticalWorkloadAutoscaler) (time.Duration, bool) {
	if wa.Status.LastUpdated == nil {
		return 0, false
	}

	now := timeNow()
	nextUpdate := wa.Status.LastUpdated.Add(wa.Spec.UpdateFrequency.Duration)
	if now.Before(nextUpdate) {
		return nextUpdate.Sub(now), true
	}

	return 0, false
}

func (r *VerticalWorkloadAutoscalerReconciler) shouldDelayUpdate(wa vwav1.VerticalWorkloadAutoscaler) (time.Duration, bool) {
	if delay, shouldDelay := r.shouldDelayUpdateWindow(wa); shouldDelay {
		return delay, true
	}

	if delay, shouldDelay := r.shouldDelayUpdateFrequency(wa); shouldDelay {
		return delay, true
	}

	return 0, false
}
