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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

	// Try to get the VerticalWorkloadAutoscaler instance
	vwa, err := r.getVWA(ctx, req.NamespacedName)
	if err != nil {
		logger.Error(err, "failed to get VerticalWorkloadAutoscaler", "namespacedName", req.NamespacedName)
		return ctrl.Result{}, nil
	}
	if vwa != nil {
		// Handle the VWA object
		return r.handleVWAChange(ctx, vwa)
	}

	// Try to get the VerticalPodAutoscaler instance
	vpa, err := r.getVPA(ctx, req.NamespacedName)
	if err != nil {
		logger.Error(err, "failed to get VerticalPodAutoscaler", "namespacedName", req.NamespacedName)
		return ctrl.Result{}, nil
	}
	if vpa != nil {
		// Handle the VPA object
		return r.handleVPAUpdate(ctx, vpa)
	}

	// Try to get the HorizontalPodAutoscaler instance
	hpa, err := r.getHPA(ctx, req.NamespacedName)
	if err != nil {
		logger.Error(err, "failed to get HorizontalPodAutoscaler", "namespacedName", req.NamespacedName)
		return ctrl.Result{}, nil
	}
	if hpa != nil {
		// Handle the HPA object
		return r.handleHPAUpdate(ctx, hpa)
	}

	logger.Info("unrecognized object type, ignoring", "Object", req.NamespacedName)
	return ctrl.Result{}, nil
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

func (r *VerticalWorkloadAutoscalerReconciler) getVPA(ctx context.Context, namespacedName client.ObjectKey) (*vpav1.VerticalPodAutoscaler, error) {
	var vpa vpav1.VerticalPodAutoscaler
	err := r.Get(ctx, namespacedName, &vpa)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &vpa, nil
}

func (r *VerticalWorkloadAutoscalerReconciler) getHPA(ctx context.Context, namespacedName client.ObjectKey) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	var hpa autoscalingv2.HorizontalPodAutoscaler
	err := r.Get(ctx, namespacedName, &hpa)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &hpa, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VerticalWorkloadAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("vwa-controller-manager")
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&vwav1.VerticalWorkloadAutoscaler{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Watches(
			&vpav1.VerticalPodAutoscaler{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForVPA),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc:  func(e event.CreateEvent) bool { return false },  // Ignore create
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },  // Ignore delete
				UpdateFunc:  func(e event.UpdateEvent) bool { return true },   // Trigger only on update
				GenericFunc: func(e event.GenericEvent) bool { return false }, // Ignore generic
			})).
		Watches(
			&autoscalingv2.HorizontalPodAutoscaler{},
			&handler.EnqueueRequestForObject{},
		).
		Complete(r); err != nil {
		log.Log.Error(err, "failed to setup controller with manager")
		return err
	}
	return nil
}

func (r *VerticalWorkloadAutoscalerReconciler) findObjectsForVPA(_ context.Context, obj client.Object) []reconcile.Request {
	requests := make([]reconcile.Request, 0)
	var vwaList vwav1.VerticalWorkloadAutoscalerList
	if err := r.List(context.Background(), &vwaList); err != nil {
		log.Log.Error(err, "failed to list VerticalWorkloadAutoscaler objects")
		return requests
	}
	vpa, ok := obj.(*vpav1.VerticalPodAutoscaler)
	if !ok {
		return requests
	}
	for _, vwa := range vwaList.Items {
		if vwa.Spec.VPAReference.Name == vpa.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: vwa.Namespace,
					Name:      vwa.Name,
				},
			})
		}
	}
	return requests
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

	// Calculate new resource values based on StepSize configuration
	newResources := r.calculateNewResources(*wa, currentResources, vpa.Status.Recommendation)

	// Update the target resource
	err = r.updateTargetObject(ctx, targetObject, wa, newResources)
	if err != nil {
		logger.Error(err, "failed to update target resource")
		r.recordEvent(wa, "Error", "UpdateFailed", "failed to update target resource")
		r.updateStatusCondition(ctx, wa, ConditionTypeError, metav1.ConditionTrue, ReasonAPIError, "failed to update target resource") // nolint:errcheck
		return ctrl.Result{}, err                                                                                                      // Retry on error
	}

	// Update VerticalWorkloadAutoscaler status
	if err = r.updateStatus(ctx, wa, newResources); err != nil {
		logger.Error(err, "failed to update VerticalWorkloadAutoscaler status")
		return ctrl.Result{}, err // Retry on error
	}

	r.recordEvent(wa, "Normal", "ResourcesUpdated", "resources updated")
	r.updateStatusCondition(ctx, wa, ConditionTypeReconciled, metav1.ConditionTrue, ReasonUpdatedResources, "updated resources") //nolint:errcheck
	// Record that the VWA was reconciled
	logger.Info("successfully reconciled VerticalWorkloadAutoscaler")
	return ctrl.Result{}, nil
}
