package controller

import (
	"context"
	"testing"
	"time"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFindCondition(t *testing.T) {
	tests := []struct {
		name           string
		conditions     []metav1.Condition
		conditionType  string
		expectedResult *metav1.Condition
	}{
		{
			name: "Condition exists",
			conditions: []metav1.Condition{
				{Type: ConditionTypeReady},
				{Type: ConditionTypeWarning},
			},
			conditionType:  ConditionTypeWarning,
			expectedResult: &metav1.Condition{Type: ConditionTypeWarning},
		},
		{
			name: "Condition does not exist",
			conditions: []metav1.Condition{
				{Type: ConditionTypeReady},
			},
			conditionType:  ConditionTypeWarning,
			expectedResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findCondition(tt.conditions, tt.conditionType)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestUpdateStatusCondition(t *testing.T) {
	tests := []struct {
		name               string
		initialConditions  []metav1.Condition
		conditionType      string
		status             metav1.ConditionStatus
		reason             string
		message            string
		expectedConditions []metav1.Condition
	}{
		{
			name:              "Add new condition",
			initialConditions: []metav1.Condition{},
			conditionType:     ConditionTypeReady,
			status:            metav1.ConditionTrue,
			reason:            "Initialized",
			message:           "VWA is ready",
			expectedConditions: []metav1.Condition{
				{
					Type:               ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(timeNow()),
					Reason:             "Initialized",
					Message:            "VWA is ready",
				},
			},
		},
		{
			name: "Update existing condition",
			initialConditions: []metav1.Condition{
				{
					Type:               ConditionTypeReady,
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.NewTime(timeNow().Add(-time.Hour)),
					Reason:             "NotReady",
					Message:            "VWA is not ready",
				},
			},
			conditionType: ConditionTypeReady,
			status:        metav1.ConditionTrue,
			reason:        "Initialized",
			message:       "VWA is ready",
			expectedConditions: []metav1.Condition{
				{
					Type:               ConditionTypeReady,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(timeNow()),
					Reason:             "Initialized",
					Message:            "VWA is ready",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the scheme and add the VerticalWorkloadAutoscaler resource
			scheme := runtime.NewScheme()
			vwav1.AddToScheme(scheme)

			// Create the fake client with the status subresource enabled
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&vwav1.VerticalWorkloadAutoscaler{}). // Enable status subresource
				Build()

			// Create the reconciler with the fake client
			r := &VerticalWorkloadAutoscalerReconciler{
				Client: client,
				Scheme: scheme,
			}

			// Create a new VerticalWorkloadAutoscaler instance
			wa := &vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vwa", Namespace: "default"},
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					Conditions: tt.initialConditions,
				},
			}

			// Create the object in the fake client
			err := client.Create(context.Background(), wa)
			assert.NoError(t, err)

			// Call the method under test
			err = r.updateStatusCondition(context.TODO(), wa, tt.conditionType, tt.status, tt.reason, tt.message)
			assert.NoError(t, err)

			// Compare conditions (ignoring LastTransitionTime)
			for i := range wa.Status.Conditions {
				wa.Status.Conditions[i].LastTransitionTime = tt.expectedConditions[i].LastTransitionTime
			}
			assert.Equal(t, tt.expectedConditions, wa.Status.Conditions)
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	tests := []struct {
		name               string
		initialConditions  []metav1.Condition
		newResources       map[string]corev1.ResourceRequirements
		expectedConditions []metav1.Condition
		expectedSkipped    bool
		expectedSkipReason string
	}{
		{
			name:              "Reconcile with new resources",
			initialConditions: []metav1.Condition{},
			newResources: map[string]corev1.ResourceRequirements{
				"resource1": {Requests: corev1.ResourceList{"cpu": resource.MustParse("100m")}},
			},
			expectedConditions: []metav1.Condition{
				{
					Type:               ConditionTypeReconciled,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(timeNow()),
					Reason:             "Reconciled",
					Message:            "VWA reconciled successfully",
				},
			},
			expectedSkipped:    false,
			expectedSkipReason: "",
		},
		{
			name:              "Reconcile with no new resources",
			initialConditions: []metav1.Condition{},
			newResources:      map[string]corev1.ResourceRequirements{},
			expectedConditions: []metav1.Condition{
				{
					Type:               ConditionTypeReconciled,
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(timeNow()),
					Reason:             "Reconciled",
					Message:            "VWA reconciled successfully",
				},
			},
			expectedSkipped:    true,
			expectedSkipReason: "No new resource recommendations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the scheme and add the VerticalWorkloadAutoscaler resource
			scheme := runtime.NewScheme()
			vwav1.AddToScheme(scheme)

			// Create the fake client with the status subresource enabled
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&vwav1.VerticalWorkloadAutoscaler{}). // Enable status subresource
				Build()

			// Create the reconciler with the fake client
			r := &VerticalWorkloadAutoscalerReconciler{
				Client: client,
				Scheme: scheme,
			}

			// Create a new VerticalWorkloadAutoscaler instance
			wa := &vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vwa", Namespace: "default"},
				Status: vwav1.VerticalWorkloadAutoscalerStatus{
					Conditions: tt.initialConditions,
				},
			}

			// Create the object in the fake client
			err := client.Create(context.Background(), wa)
			assert.NoError(t, err)

			// Call the method under test
			err = r.updateStatus(context.TODO(), wa, tt.newResources)
			assert.NoError(t, err)

			// Compare conditions (ignoring LastTransitionTime)
			for i := range wa.Status.Conditions {
				wa.Status.Conditions[i].LastTransitionTime = tt.expectedConditions[i].LastTransitionTime
			}
			assert.Equal(t, tt.expectedConditions, wa.Status.Conditions)
			assert.Equal(t, tt.expectedSkipped, wa.Status.SkippedUpdates)
			assert.Equal(t, tt.expectedSkipReason, wa.Status.SkipReason)
		})
	}
}
