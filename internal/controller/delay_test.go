package controller

import (
	"testing"
	"time"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShouldDelayUpdateFrequency(t *testing.T) {
	tests := []struct {
		name           string
		wa             vwav1.VerticalWorkloadAutoscaler
		currentTime    time.Time
		expectedDelay  time.Duration
		expectedResult bool
	}{
		{
			name: "No last updated time",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					LastUpdated: nil,
				},
			},
			currentTime:    time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			expectedDelay:  0,
			expectedResult: false,
		},
		{
			name: "Update frequency not reached",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					LastUpdated: &metav1.Time{Time: time.Date(2023, 10, 10, 9, 0, 0, 0, time.UTC)},
				},
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					UpdateFrequency: &metav1.Duration{Duration: 2 * time.Hour},
				},
			},
			currentTime:    time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			expectedDelay:  time.Hour,
			expectedResult: true,
		},
		{
			name: "Update frequency reached",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					LastUpdated: &metav1.Time{Time: time.Date(2023, 10, 10, 8, 0, 0, 0, time.UTC)},
				},
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					UpdateFrequency: &metav1.Duration{Duration: 2 * time.Hour},
				},
			},
			currentTime:    time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			expectedDelay:  0,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeNow = func() time.Time { return tt.currentTime }
			r := &VerticalWorkloadAutoscalerReconciler{}
			delay, result := r.shouldDelayUpdateFrequency(tt.wa)
			assert.Equal(t, tt.expectedDelay, delay)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestParseWindowTimes(t *testing.T) {
	tests := []struct {
		name          string
		now           time.Time
		window        vwav1.UpdateWindow
		expectedStart time.Time
		expectedEnd   time.Time
		expectError   bool
	}{
		{
			name: "Valid window times",
			now:  time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			window: vwav1.UpdateWindow{
				StartTime: "09:00",
				EndTime:   "11:00",
				TimeZone:  "UTC",
			},
			expectedStart: time.Date(2023, 10, 10, 9, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2023, 10, 10, 11, 0, 0, 0, time.UTC),
			expectError:   false,
		},
		{
			name: "Invalid start time format",
			now:  time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			window: vwav1.UpdateWindow{
				StartTime: "invalid",
				EndTime:   "11:00",
				TimeZone:  "UTC",
			},
			expectError: true,
		},
		{
			name: "Invalid end time format",
			now:  time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			window: vwav1.UpdateWindow{
				StartTime: "09:00",
				EndTime:   "invalid",
				TimeZone:  "UTC",
			},
			expectError: true,
		},
		{
			name: "Invalid time zone",
			now:  time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			window: vwav1.UpdateWindow{
				StartTime: "09:00",
				EndTime:   "11:00",
				TimeZone:  "Invalid/Zone",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := parseWindowTimes(tt.now, tt.window)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStart, start)
				assert.Equal(t, tt.expectedEnd, end)
			}
		})
	}
}

func TestShouldDelayUpdateWindow(t *testing.T) {
	tests := []struct {
		name           string
		wa             vwav1.VerticalWorkloadAutoscaler
		currentTime    time.Time
		expectedDelay  time.Duration
		expectedResult bool
	}{
		{
			name: "No allowed update windows",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					AllowedUpdateWindows: []vwav1.UpdateWindow{},
				},
			},
			currentTime:    time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			expectedDelay:  0,
			expectedResult: false,
		},
		{
			name: "Within allowed update window",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					AllowedUpdateWindows: []vwav1.UpdateWindow{
						{
							DayOfWeek: "Tuesday",
							StartTime: "09:00",
							EndTime:   "11:00",
							TimeZone:  "UTC",
						},
					},
				},
			},
			currentTime:    time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			expectedDelay:  0,
			expectedResult: false,
		},
		{
			name: "Outside allowed update window",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					AllowedUpdateWindows: []vwav1.UpdateWindow{
						{
							DayOfWeek: "Tuesday",
							StartTime: "09:00",
							EndTime:   "11:00",
							TimeZone:  "UTC",
						},
					},
				},
			},
			currentTime:    time.Date(2023, 10, 10, 8, 0, 0, 0, time.UTC),
			expectedDelay:  time.Hour,
			expectedResult: true,
		},
		{
			name: "Invalid time zone",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					AllowedUpdateWindows: []vwav1.UpdateWindow{
						{
							DayOfWeek: "Tuesday",
							StartTime: "09:00",
							EndTime:   "11:00",
							TimeZone:  "Invalid/Zone",
						},
					},
				},
			},
			currentTime:    time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			expectedDelay:  0,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeNow = func() time.Time { return tt.currentTime }
			r := &VerticalWorkloadAutoscalerReconciler{}
			delay, result := r.shouldDelayUpdateWindow(tt.wa)
			assert.Equal(t, tt.expectedDelay, delay)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestShouldDelayUpdate(t *testing.T) {
	tests := []struct {
		name           string
		wa             vwav1.VerticalWorkloadAutoscaler
		currentTime    time.Time
		expectedDelay  time.Duration
		expectedResult bool
	}{
		{
			name: "No delay conditions",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					AllowedUpdateWindows: []vwav1.UpdateWindow{},
					UpdateFrequency:      &metav1.Duration{Duration: 0},
				},
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					LastUpdated: nil,
				},
			},
			currentTime:    time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			expectedDelay:  0,
			expectedResult: false,
		},
		{
			name: "Delay due to update window",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					AllowedUpdateWindows: []vwav1.UpdateWindow{
						{
							DayOfWeek: "Tuesday",
							StartTime: "09:00",
							EndTime:   "11:00",
							TimeZone:  "UTC",
						},
					},
				},
			},
			currentTime:    time.Date(2023, 10, 10, 8, 0, 0, 0, time.UTC),
			expectedDelay:  time.Hour,
			expectedResult: true,
		},
		{
			name: "Delay due to update frequency",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					LastUpdated: &metav1.Time{Time: time.Date(2023, 10, 10, 9, 0, 0, 0, time.UTC)},
				},
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					UpdateFrequency: &metav1.Duration{Duration: 2 * time.Hour},
				},
			},
			currentTime:    time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			expectedDelay:  time.Hour,
			expectedResult: true,
		},
		{
			name: "No delay when within update window and frequency reached",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					AllowedUpdateWindows: []vwav1.UpdateWindow{
						{
							DayOfWeek: "Tuesday",
							StartTime: "09:00",
							EndTime:   "11:00",
							TimeZone:  "UTC",
						},
					},
					UpdateFrequency: &metav1.Duration{Duration: 2 * time.Hour},
				},
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					LastUpdated: &metav1.Time{Time: time.Date(2023, 10, 10, 8, 0, 0, 0, time.UTC)},
				},
			},
			currentTime:    time.Date(2023, 10, 10, 10, 0, 0, 0, time.UTC),
			expectedDelay:  0,
			expectedResult: false,
		},
		{
			name: "Delay due to both update window and frequency",
			wa: vwav1.VerticalWorkloadAutoscaler{
				Spec: vwav1.VerticalWorkloadAutoscalerSpec{
					AllowedUpdateWindows: []vwav1.UpdateWindow{
						{
							DayOfWeek: "Tuesday",
							StartTime: "09:00",
							EndTime:   "11:00",
							TimeZone:  "UTC",
						},
					},
					UpdateFrequency: &metav1.Duration{Duration: 2 * time.Hour},
				},
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					LastUpdated: &metav1.Time{Time: time.Date(2023, 10, 10, 9, 0, 0, 0, time.UTC)},
				},
			},
			currentTime:    time.Date(2023, 10, 10, 8, 0, 0, 0, time.UTC),
			expectedDelay:  time.Hour,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeNow = func() time.Time { return tt.currentTime }
			r := &VerticalWorkloadAutoscalerReconciler{}
			delay, result := r.shouldDelayUpdate(tt.wa)
			assert.Equal(t, tt.expectedDelay, delay)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
