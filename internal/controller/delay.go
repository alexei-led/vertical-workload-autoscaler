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

	var nextUpdate time.Time
	var withinWindow bool

	// Map to convert day names to their corresponding integer values
	dayOfWeekMap := map[string]time.Weekday{
		"Sunday":    time.Sunday,
		"Monday":    time.Monday,
		"Tuesday":   time.Tuesday,
		"Wednesday": time.Wednesday,
		"Thursday":  time.Thursday,
		"Friday":    time.Friday,
		"Saturday":  time.Saturday,
	}

	for _, window := range wa.Spec.AllowedUpdateWindows {
		start, end, err := parseWindowTimes(now, window)
		if err != nil {
			continue
		}

		// Check if current time is within the update window
		if now.After(start) && now.Before(end) {
			withinWindow = true
			break
		}

		// Calculate the next update time considering the day of the week
		for i := 0; i < 7; i++ {
			dayOffset := (int(dayOfWeekMap[window.DayOfWeek]) - int(now.Weekday()) + i) % 7
			nextStart := start.AddDate(0, 0, dayOffset)
			if nextUpdate.IsZero() || nextStart.Before(nextUpdate) {
				nextUpdate = nextStart
			}
		}
	}

	if withinWindow {
		return 0, false
	}

	if nextUpdate.IsZero() {
		return 0, false
	}

	return nextUpdate.Sub(now), true
}

func parseWindowTimes(now time.Time, window vwav1.UpdateWindow) (time.Time, time.Time, error) {
	loc, err := time.LoadLocation(window.TimeZone)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	startTimeStr := now.Format("2006-01-02") + " " + window.StartTime
	endTimeStr := now.Format("2006-01-02") + " " + window.EndTime

	start, err := time.ParseInLocation("2006-01-02 15:04", startTimeStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	end, err := time.ParseInLocation("2006-01-02 15:04", endTimeStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return start, end, nil
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
