/*
Copyright 2024 Alexei Ledenev.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"reflect"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// VerticalWorkloadAutoscalerReconciler reconciles a VerticalWorkloadAutoscaler object
type VerticalWorkloadAutoscalerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=autoscaling.workload.io,resources=verticalworkloadautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling.workload.io,resources=verticalworkloadautoscalers/finalizers,verbs=update
// +kubebuilder:rbac:groups=autoscaling.workload.io,resources=verticalworkloadautoscalers/status,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=autoscaling.k8s.io,resources=verticalpodautoscalers,verbs=get;list;watch
// +kubebuilder:rbac:groups=autoscaling.k8s.io,resources=verticalpodautoscalers/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments;replicasets;statefulsets;daemonsets,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=batch,resources=jobs;cronjobs,verbs=get;list;watch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *VerticalWorkloadAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the VWA object
	vwa, err := r.getVWA(ctx, req.NamespacedName)
	if err != nil {
		logger.Error(err, "failed to get VerticalWorkloadAutoscaler", "namespacedName", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if vwa == nil {
		logger.Info("VerticalWorkloadAutoscaler not found, skipping", "namespacedName", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// Handle the VWA reconciliation
	return r.handleVWAChange(ctx, vwa)
}

func (r *VerticalWorkloadAutoscalerReconciler) getVWA(ctx context.Context, namespacedName client.ObjectKey) (*vwav1.VerticalWorkloadAutoscaler, error) {
	var vwa vwav1.VerticalWorkloadAutoscaler
	err := r.Get(ctx, namespacedName, &vwa)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &vwa, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VerticalWorkloadAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("vwa-controller-manager")
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&vwav1.VerticalWorkloadAutoscaler{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldVWA, okOld := e.ObjectOld.(*vwav1.VerticalWorkloadAutoscaler)
				newVWA, okNew := e.ObjectNew.(*vwav1.VerticalWorkloadAutoscaler)
				if !okOld || !okNew {
					return false
				}
				// Trigger only if VWA spec changed, ignore status updates
				return !reflect.DeepEqual(oldVWA.Spec, newVWA.Spec)
			},
			CreateFunc:  func(e event.CreateEvent) bool { return true },   // Trigger on create
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },  // Ignore delete
			GenericFunc: func(e event.GenericEvent) bool { return false }, // Ignore generic
		})).
		Watches(
			&vpav1.VerticalPodAutoscaler{},
			handler.EnqueueRequestsFromMapFunc(r.findVWAForVPA),
			builder.WithPredicates(predicate.Funcs{
				UpdateFunc: func(e event.UpdateEvent) bool {
					oldVPA, okOld := e.ObjectOld.(*vpav1.VerticalPodAutoscaler)
					newVPA, okNew := e.ObjectNew.(*vpav1.VerticalPodAutoscaler)
					if !okOld || !okNew {
						return false
					}
					// Trigger only on updates to the status recommendation field
					return !reflect.DeepEqual(oldVPA.Status.Recommendation, newVPA.Status.Recommendation)
				},
				CreateFunc:  func(e event.CreateEvent) bool { return false },  // Ignore create
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },  // Ignore delete
				GenericFunc: func(e event.GenericEvent) bool { return false }, // Ignore generic
			})).
		// Map HPA updates to VWA reconciliation
		Watches(
			&autoscalingv2.HorizontalPodAutoscaler{},
			handler.EnqueueRequestsFromMapFunc(r.findVWAForHPA),
			builder.WithPredicates(predicate.Funcs{
				UpdateFunc:  func(e event.UpdateEvent) bool { return true },   // Trigger on updates
				CreateFunc:  func(e event.CreateEvent) bool { return true },   // Trigger on create
				DeleteFunc:  func(e event.DeleteEvent) bool { return true },   // Trigger on delete
				GenericFunc: func(e event.GenericEvent) bool { return false }, // Ignore generic
			})).
		Complete(r); err != nil {
		log.Log.Error(err, "failed to setup controller with manager")
		return err
	}
	return nil
}

// checks if there is any other VWA referencing the same VPA
func (r *VerticalWorkloadAutoscalerReconciler) ensureNoDuplicateVWA(ctx context.Context, wa *vwav1.VerticalWorkloadAutoscaler) error {
	// List all VerticalWorkloadAutoscaler objects
	var waList vwav1.VerticalWorkloadAutoscalerList
	if err := r.List(ctx, &waList); err != nil {
		return err
	}

	// Iterate through VWA objects and check for duplicates
	for _, existingWA := range waList.Items {
		if existingWA.Spec.VPAReference.Name == wa.Spec.VPAReference.Name && existingWA.Name != wa.Name {
			return fmt.Errorf("VPA '%s' is already referenced by another VWA object '%s'", wa.Spec.VPAReference.Name, existingWA.Name)
		}
	}
	return nil
}

func (r *VerticalWorkloadAutoscalerReconciler) handleVWAChange(ctx context.Context, wa *vwav1.VerticalWorkloadAutoscaler) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Ensure no duplicate VWA exists
	if err := r.ensureNoDuplicateVWA(ctx, wa); err != nil {
		logger.Error(err, "duplicate VWA found", "VWA", wa.Name)
		r.updateStatusCondition(ctx, wa, ConditionTypeError, metav1.ConditionTrue, ReasonVPAReferenceConflict, fmt.Sprintf("VPA '%s' is already referenced by another VWA object", wa.Spec.VPAReference.Name)) // nolint:errcheck
		return ctrl.Result{}, err                                                                                                                                                                              // Return error to indicate failure and retry                                                                                                                                                                              // Avoid calling reconcile again
	}

	// Check if an update is allowed now or should be delayed
	if delay, shouldDelay := r.shouldDelayUpdate(*wa); shouldDelay {
		logger.Info("delaying update", "RequeueAfter", delay)
		r.recordEvent(wa, "Normal", "UpdateDelayed", fmt.Sprintf("update delayed for %s", delay))
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	// Fetch the associated VPA object
	vpa, err := r.fetchVPA(ctx, *wa)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("VPA not found: ignoring since object must be deleted", "VPA", wa.Spec.VPAReference.Name)
			r.recordEvent(wa, "Normal", "VPAReferenceNotFound", "VPA not found")                                                    // nolint:errcheck
			r.updateStatusCondition(ctx, wa, ConditionTypeError, metav1.ConditionTrue, ReasonVPAReferenceNotFound, "VPA not found") // nolint:errcheck
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to fetch VPA", "VPA", wa.Spec.VPAReference.Name)
		r.updateStatusCondition(ctx, wa, ConditionTypeError, metav1.ConditionTrue, ReasonAPIError, "failed to fetch VPA") // nolint:errcheck
		return ctrl.Result{}, err                                                                                         // Retry on error
	}

	// Update the VWA status with the VPA reference
	r.updateStatusCondition(ctx, wa, ConditionTypeReady, metav1.ConditionTrue, ReasonVPAFound, "VPA found") // nolint:errcheck
	// Record that VPA was found
	r.recordEvent(wa, "Normal", "VPAFound", fmt.Sprintf("VPA '%s' found", vpa.Name))

	// if VPA has no recommendations, nothing to do
	if vpa.Status.Recommendation == nil {
		logger.Info("VPA has no recommendations", "VPA", vpa.Name)
		r.updateStatusCondition(ctx, wa, ConditionTypeReady, metav1.ConditionFalse, ReasonNoRecommendation, "VPA has no recommendations yet") // nolint:errcheck
		return ctrl.Result{}, nil                                                                                                             // Avoid calling reconcile again
	}

	// Fetch the target object from the VPA configuration
	targetObject, err := r.fetchTargetObject(ctx, vpa)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("target object not found; ignoring since object must be deleted", "VPA", vpa.Name)
			r.updateStatusCondition(ctx, wa, ConditionTypeError, metav1.ConditionTrue, ReasonTargetObjectNotFound, "target object not found") //nolint:errcheck
			return ctrl.Result{}, nil
		}
		logger.Error(err, "failed to fetch target object", "VPA", vpa.Name)
		r.updateStatusCondition(ctx, wa, ConditionTypeError, metav1.ConditionTrue, ReasonAPIError, "failed to fetch target object") //nolint:errcheck
		return ctrl.Result{}, err                                                                                                   // Retry on error
	}

	// Update VWA Status.ScaleTargetRef if different from VPA TargetRef
	if wa.Status.ScaleTargetRef.Name != vpa.Spec.TargetRef.Name || wa.Status.ScaleTargetRef.Kind != vpa.Spec.TargetRef.Kind {
		wa.Status.ScaleTargetRef = autoscalingv2.CrossVersionObjectReference{
			Kind:       vpa.Spec.TargetRef.Kind,
			Name:       vpa.Spec.TargetRef.Name,
			APIVersion: vpa.Spec.TargetRef.APIVersion,
		}
		if err = r.Status().Update(ctx, wa); err != nil {
			logger.Error(err, "failed to update VWA status with new ScaleTargetRef")
			return ctrl.Result{}, err // Retry on error
		}
		r.recordEvent(wa, "Normal", "ScaleTargetRefUpdated", fmt.Sprintf("ScaleTargetRef updated to '%s'", vpa.Spec.TargetRef.Name))
		r.updateStatusCondition(ctx, wa, ConditionTypeReady, metav1.ConditionTrue, ReasonTargetObjectFound, "target object found") //nolint:errcheck
	}

	// fetch current resources of the target object
	currentResources, err := r.fetchCurrentResources(targetObject)
	if err != nil {
		logger.Error(err, "failed to fetch current resources")
		r.updateStatusCondition(ctx, wa, ConditionTypeError, metav1.ConditionTrue, ReasonAPIError, "failed to fetch current resources") // nolint:errcheck
		return ctrl.Result{}, err                                                                                                       // Retry on error
	}

	// find the HPA associated with the target object
	var ignoreCPU, ignoreMemory bool
	hpa, err := r.findHPAForVWA(ctx, wa)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "unexpected error while finding HPA")
			r.updateStatusCondition(ctx, wa, ConditionTypeError, metav1.ConditionTrue, ReasonAPIError, "failed to find HPA") //nolint:errcheck
			return ctrl.Result{}, err                                                                                        // Retry on error
		}
	}
	// Check if the HPA has CPU or Memory metrics
	if hpa != nil {
		ignoreCPU, ignoreMemory = r.getIgnoreFlags(hpa)
	}
	// requeue if ignore flags were updated
	if wa.Spec.IgnoreCPURecommendations != ignoreCPU || wa.Spec.IgnoreMemoryRecommendations != ignoreMemory {
		r.recordEvent(wa, "Normal", "IgnoreFlagsUpdated", "ignore flags updated")
		r.updateStatusCondition(ctx, wa, ConditionTypeReconciled, metav1.ConditionTrue, ReasonUpdatedResources, "updated ignore flags") //nolint:errcheck
		return ctrl.Result{Requeue: true}, nil
	}

	// Calculate new resource values based on VPA recommendations and VWA configuration
	newResources := r.calculateNewResources(*wa, currentResources, vpa.Status.Recommendation)

	// Update the target resource
	updated, err := r.updateTargetObject(ctx, targetObject, wa, newResources)
	if err != nil {
		logger.Error(err, "failed to update target resource")
		r.recordEvent(wa, "Error", "UpdateFailed", "failed to update target resource")
		r.updateStatusCondition(ctx, wa, ConditionTypeError, metav1.ConditionTrue, ReasonAPIError, "failed to update target resource") // nolint:errcheck
		return ctrl.Result{}, err                                                                                                      // Retry on error
	}

	if updated {
		// Update VerticalWorkloadAutoscaler status
		if err = r.updateStatus(ctx, wa, newResources); err != nil {
			logger.Error(err, "failed to update VerticalWorkloadAutoscaler status")
			return ctrl.Result{}, err // Retry on error
		}
		r.recordEvent(wa, "Normal", "ResourcesUpdated", "resources updated")
		r.updateStatusCondition(ctx, wa, ConditionTypeReconciled, metav1.ConditionTrue, ReasonUpdatedResources, "updated resources") //nolint:errcheck
	} else {
		r.recordEvent(wa, "Normal", "WaitingForRecommendations", "waiting for VPA recommendations")
		r.updateStatusCondition(ctx, wa, ConditionTypeReconciled, metav1.ConditionFalse, ReasonWaitingForRecommendations, "waiting for VPA recommendations") //nolint:errcheck
	}
	return ctrl.Result{}, nil
}
