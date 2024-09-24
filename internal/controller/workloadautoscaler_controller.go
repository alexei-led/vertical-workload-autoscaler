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

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// VerticalWorkloadAutoscalerReconciler reconciles a VerticalWorkloadAutoscaler object
type VerticalWorkloadAutoscalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=autoscaling.k8s.io,resources=verticalworkloadautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling.k8s.io,resources=verticalworkloadautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=autoscaling.k8s.io,resources=verticalworkloadautoscalers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *VerticalWorkloadAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the VerticalWorkloadAutoscaler object
	var wa vwav1.VerticalWorkloadAutoscaler
	if err := r.Get(ctx, req.NamespacedName, &wa); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("VerticalWorkloadAutoscaler resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get VerticalWorkloadAutoscaler")
		return ctrl.Result{}, err
	}

	// Check if an update is allowed now or should be delayed
	if delay, shouldDelay := r.shouldDelayUpdate(wa); shouldDelay {
		logger.Info("Delaying update", "RequeueAfter", delay)
		return ctrl.Result{RequeueAfter: delay}, nil
	}

	// Fetch the associated VPA object
	vpa, err := r.fetchVPA(ctx, wa)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("VPA not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to fetch VPA")
		return ctrl.Result{}, err
	}

	// Fetch the target resource from the VPA configuration
	targetResource, err := r.fetchTargetResource(ctx, vpa)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Target resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to fetch target resource")
		return ctrl.Result{}, err
	}

	// Calculate new resource values based on StepSize configuration
	newResources := r.calculateNewResources(wa, vpa.Status.Recommendation)

	// Update the target resource
	updated, err := r.updateTargetResource(ctx, targetResource, newResources)
	if err != nil {
		logger.Error(err, "Failed to update target resource")
		return ctrl.Result{}, err
	}

	if updated {
		// Update annotations to force pod recreation and add GitOps conflict avoidance
		if err = r.updateAnnotations(ctx, targetResource); err != nil {
			logger.Error(err, "Failed to update annotations")
			return ctrl.Result{}, err
		}
	}

	// Record progress statuses on the VerticalWorkloadAutoscaler object status
	if err = r.recordProgress(ctx, wa); err != nil {
		logger.Error(err, "Failed to record progress")
		return ctrl.Result{}, err
	}

	// Update VerticalWorkloadAutoscaler status
	if err = r.updateStatus(ctx, wa); err != nil {
		logger.Error(err, "Failed to update VerticalWorkloadAutoscaler status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled VerticalWorkloadAutoscaler")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VerticalWorkloadAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&vwav1.VerticalWorkloadAutoscaler{}).
		Watches(
			&vpav1.VerticalPodAutoscaler{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForVPA),
			builder.WithPredicates(VPARecommendationChangedPredicate{})).
		Complete(r); err != nil {
		log.Log.Error(err, "Failed to setup controller with manager")
		return err
	}
	return nil
}

func (r *VerticalWorkloadAutoscalerReconciler) findObjectsForVPA(_ context.Context, obj client.Object) []reconcile.Request {
	var requests []reconcile.Request
	var waList vwav1.VerticalWorkloadAutoscalerList
	if err := r.List(context.Background(), &waList); err != nil {
		log.Log.Error(err, "Failed to list VerticalWorkloadAutoscaler objects")
		return requests
	}
	vpa, ok := obj.(*vpav1.VerticalPodAutoscaler)
	if !ok {
		return requests
	}
	for _, wa := range waList.Items {
		if wa.Spec.VPAReference.Name == vpa.Name && wa.Spec.VPAReference.Namespace == vpa.Namespace {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: wa.Namespace,
					Name:      wa.Name,
				},
			})
		}
	}
	return requests
}
