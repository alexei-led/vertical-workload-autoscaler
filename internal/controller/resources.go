package controller

import (
	"context"
	"fmt"
	"time"

	autoscalingk8siov1alpha1 "github.com/alexei-led/workload-autoscaler/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *WorkloadAutoscalerReconciler) fetchTargetResource(ctx context.Context, vpa *vpav1.VerticalPodAutoscaler) (client.Object, error) {
	var targetResource client.Object

	switch vpa.Spec.TargetRef.Kind {
	case "Deployment":
		targetResource = &appsv1.Deployment{}
	case "StatefulSet":
		targetResource = &appsv1.StatefulSet{}
	case "CronJob":
		targetResource = &batchv1.CronJob{}
	case "ReplicaSet":
		targetResource = &appsv1.ReplicaSet{}
	case "DaemonSet":
		targetResource = &appsv1.DaemonSet{}
	default:
		return nil, fmt.Errorf("unsupported target resource kind: %s", vpa.Spec.TargetRef.Kind)
	}

	err := r.Get(ctx, client.ObjectKey{Name: vpa.Spec.TargetRef.Name, Namespace: vpa.Namespace}, targetResource)
	if err != nil {
		return nil, fmt.Errorf("failed to get target resource %s/%s: %w", vpa.Namespace, vpa.Spec.TargetRef.Name, err)
	}
	return targetResource, nil
}

func (r *WorkloadAutoscalerReconciler) calculateNewResources(wa autoscalingk8siov1alpha1.WorkloadAutoscaler, recommendations *vpav1.RecommendedPodResources) map[string]*corev1.ResourceRequirements {
	newResources := make(map[string]*corev1.ResourceRequirements)

	for _, containerRec := range recommendations.ContainerRecommendations {
		newReq := &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{},
			Limits:   corev1.ResourceList{},
		}

		stepCPU := resource.MustParse(wa.Spec.StepSize.CPU)
		stepMemory := resource.MustParse(wa.Spec.StepSize.Memory)

		if wa.Spec.QualityOfService == "Guaranteed" {
			targetCPU := roundUp(containerRec.Target[corev1.ResourceCPU], stepCPU)
			targetMemory := roundUp(containerRec.Target[corev1.ResourceMemory], stepMemory)

			newReq.Requests[corev1.ResourceCPU] = targetCPU
			newReq.Requests[corev1.ResourceMemory] = targetMemory

			if !wa.Spec.AvoidCPULimit {
				newReq.Limits[corev1.ResourceCPU] = targetCPU
			}
			newReq.Limits[corev1.ResourceMemory] = targetMemory
		} else if wa.Spec.QualityOfService == "Burstable" {
			lowerBoundCPU := roundUp(containerRec.LowerBound[corev1.ResourceCPU], stepCPU)
			lowerBoundMemory := roundUp(containerRec.LowerBound[corev1.ResourceMemory], stepMemory)
			upperBoundCPU := roundUp(containerRec.UpperBound[corev1.ResourceCPU], stepCPU)
			upperBoundMemory := roundUp(containerRec.UpperBound[corev1.ResourceMemory], stepMemory)

			newReq.Requests[corev1.ResourceCPU] = lowerBoundCPU
			newReq.Requests[corev1.ResourceMemory] = lowerBoundMemory

			if !wa.Spec.AvoidCPULimit {
				newReq.Limits[corev1.ResourceCPU] = upperBoundCPU
			}
			newReq.Limits[corev1.ResourceMemory] = upperBoundMemory
		}

		newResources[containerRec.ContainerName] = newReq
	}

	return newResources
}

func roundUp(quantity resource.Quantity, step resource.Quantity) resource.Quantity {
	value := quantity.Value()
	stepValue := step.Value()
	roundedValue := ((value + stepValue - 1) / stepValue) * stepValue
	return *resource.NewQuantity(roundedValue, quantity.Format)
}

func (r *WorkloadAutoscalerReconciler) updateTargetResource(ctx context.Context, targetResource client.Object, newResources map[string]*corev1.ResourceRequirements) error {
	updateContainers := func(containers []corev1.Container) {
		for _, container := range containers {
			if recommendedResources, ok := newResources[container.Name]; ok {
				recommendedResources.Requests.DeepCopyInto(&container.Resources.Requests)
			}
		}
	}

	switch resource := targetResource.(type) {
	case *appsv1.Deployment:
		updateContainers(resource.Spec.Template.Spec.Containers)
	case *appsv1.StatefulSet:
		updateContainers(resource.Spec.Template.Spec.Containers)
	case *batchv1.CronJob:
		updateContainers(resource.Spec.JobTemplate.Spec.Template.Spec.Containers)
	case *appsv1.ReplicaSet:
		updateContainers(resource.Spec.Template.Spec.Containers)
	case *appsv1.DaemonSet:
		updateContainers(resource.Spec.Template.Spec.Containers)
	default:
		return fmt.Errorf("unsupported target resource type: %T", targetResource)
	}

	if err := r.Update(ctx, targetResource); err != nil {
		return fmt.Errorf("failed to update target resource: %w", err)
	}
	return nil
}

func (r *WorkloadAutoscalerReconciler) forcePodRecreation(ctx context.Context, targetResource client.Object) error {
	annotations := targetResource.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["workloadautoscaler.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
	targetResource.SetAnnotations(annotations)
	if err := r.Update(ctx, targetResource); err != nil {
		return err
	}
	return nil
}

func (r *WorkloadAutoscalerReconciler) addArgoCDAnnotation(ctx context.Context, targetResource client.Object) error {
	annotations := targetResource.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["argocd.argoproj.io/compare-options"] = "IgnoreResourceRequests"
	annotations["fluxcd.io/ignore"] = "true"
	targetResource.SetAnnotations(annotations)
	if err := r.Update(ctx, targetResource); err != nil {
		return err
	}
	return nil
}
