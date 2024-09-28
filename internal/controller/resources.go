package controller

import (
	"context"
	"fmt"
	"time"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	vpav1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultCPUTolerance    = 0.10
	defaultMemoryTolerance = 0.10
)

func (r *VerticalWorkloadAutoscalerReconciler) fetchTargetObject(ctx context.Context, vpa *vpav1.VerticalPodAutoscaler) (client.Object, error) {
	if vpa.Spec.TargetRef == nil {
		return nil, fmt.Errorf("targetRef is not set")
	}

	var targetObject client.Object

	switch vpa.Spec.TargetRef.Kind {
	case "Deployment":
		targetObject = &appsv1.Deployment{}
	case "StatefulSet":
		targetObject = &appsv1.StatefulSet{}
	case "CronJob":
		targetObject = &batchv1.CronJob{}
	case "ReplicaSet":
		targetObject = &appsv1.ReplicaSet{}
	case "DaemonSet":
		targetObject = &appsv1.DaemonSet{}
	default:
		return nil, fmt.Errorf("unsupported target resource kind: %s", vpa.Spec.TargetRef.Kind)
	}

	err := r.Get(ctx, client.ObjectKey{Name: vpa.Spec.TargetRef.Name, Namespace: vpa.Namespace}, targetObject)
	if err != nil {
		return nil, fmt.Errorf("failed to get target resource %s/%s: %w", vpa.Namespace, vpa.Spec.TargetRef.Name, err)
	}
	return targetObject, nil
}

// calculateNewResources calculates the new resource requirements based on the VPA recommendations
// and the VWA configuration (tolerance, quality of service, etc.)
func (r *VerticalWorkloadAutoscalerReconciler) calculateNewResources(wa vwav1.VerticalWorkloadAutoscaler, currentResources map[string]corev1.ResourceRequirements, recommendations *vpav1.RecommendedPodResources) map[string]corev1.ResourceRequirements {
	newResources := make(map[string]corev1.ResourceRequirements)

	cpuTolerance, memoryTolerance := getTolerances(wa)

	for _, containerRec := range recommendations.ContainerRecommendations {
		var newReq *corev1.ResourceRequirements
		currentReq := currentResources[containerRec.ContainerName]

		if wa.Spec.QualityOfService == vwav1.GuaranteedQualityOfService {
			newReq = updateGuaranteedResources(currentReq, containerRec, cpuTolerance, memoryTolerance, wa.Spec.AvoidCPULimit)
		} else if wa.Spec.QualityOfService == vwav1.BurstableQualityOfService {
			newReq = updateBurstableResources(currentReq, containerRec, cpuTolerance, memoryTolerance, wa.Spec.AvoidCPULimit)
		}

		newResources[containerRec.ContainerName] = *newReq
	}

	return newResources
}

// getTolerances returns the CPU and memory tolerances based on the VWA configuration
func getTolerances(wa vwav1.VerticalWorkloadAutoscaler) (cpuTolerance, memoryTolerance float64) {
	cpuTolerance, memoryTolerance = defaultCPUTolerance, defaultMemoryTolerance

	if wa.Spec.UpdateTolerance != nil {
		if wa.Spec.UpdateTolerance.CPU > 0 {
			cpuTolerance = float64(wa.Spec.UpdateTolerance.CPU) / 100
		}
		if wa.Spec.UpdateTolerance.Memory > 0 {
			memoryTolerance = float64(wa.Spec.UpdateTolerance.Memory) / 100
		}
	}
	return
}

// applyUpdate checks if the recommended resource is different from the current resource considering the tolerance
func applyUpdate(current, recommended resource.Quantity, tolerance float64) bool {
	if current.IsZero() {
		return true
	}
	change := float64(recommended.MilliValue()-current.MilliValue()) / float64(current.MilliValue())
	return change >= tolerance || change <= -tolerance
}

// updateGuaranteedResources updates the resource requirements for a container with guaranteed QoS
func updateGuaranteedResources(currentReq corev1.ResourceRequirements, containerRec vpav1.RecommendedContainerResources, cpuTolerance, memoryTolerance float64, avoidCPULimit bool) *corev1.ResourceRequirements {
	newReq := currentReq.DeepCopy()

	if applyUpdate(currentReq.Requests[corev1.ResourceCPU], containerRec.Target[corev1.ResourceCPU], cpuTolerance) {
		newReq.Requests[corev1.ResourceCPU] = containerRec.Target[corev1.ResourceCPU]
		if avoidCPULimit {
			delete(newReq.Limits, corev1.ResourceCPU)
		} else {
			newReq.Limits[corev1.ResourceCPU] = containerRec.Target[corev1.ResourceCPU]
		}
	}
	if applyUpdate(currentReq.Requests[corev1.ResourceMemory], containerRec.Target[corev1.ResourceMemory], memoryTolerance) {
		newReq.Requests[corev1.ResourceMemory] = containerRec.Target[corev1.ResourceMemory]
		newReq.Limits[corev1.ResourceMemory] = containerRec.Target[corev1.ResourceMemory]
	}
	return newReq
}

// updateBurstableResources updates the resource requirements for a container with burstable QoS
func updateBurstableResources(currentReq corev1.ResourceRequirements, containerRec vpav1.RecommendedContainerResources, cpuTolerance, memoryTolerance float64, avoidCPULimit bool) *corev1.ResourceRequirements {
	newReq := currentReq.DeepCopy()
	lowerBoundCPU := containerRec.LowerBound[corev1.ResourceCPU]
	lowerBoundMemory := containerRec.LowerBound[corev1.ResourceMemory]
	upperBoundCPU := containerRec.UpperBound[corev1.ResourceCPU]
	upperBoundMemory := containerRec.UpperBound[corev1.ResourceMemory]

	if applyUpdate(currentReq.Requests[corev1.ResourceCPU], lowerBoundCPU, cpuTolerance) {
		newReq.Requests[corev1.ResourceCPU] = lowerBoundCPU
		if avoidCPULimit {
			delete(newReq.Limits, corev1.ResourceCPU)
		} else {
			newReq.Limits[corev1.ResourceCPU] = upperBoundCPU
		}
	}
	if applyUpdate(currentReq.Requests[corev1.ResourceMemory], lowerBoundMemory, memoryTolerance) {
		newReq.Requests[corev1.ResourceMemory] = lowerBoundMemory
		newReq.Limits[corev1.ResourceMemory] = upperBoundMemory
	}
	return newReq
}

func resourceRequirementsEqual(a, b corev1.ResourceRequirements) bool {
	return a.Requests.Cpu().Equal(*b.Requests.Cpu()) &&
		a.Requests.Memory().Equal(*b.Requests.Memory()) &&
		a.Limits.Cpu().Equal(*b.Limits.Cpu()) &&
		a.Limits.Memory().Equal(*b.Limits.Memory())
}

func (r *VerticalWorkloadAutoscalerReconciler) fetchCurrentResources(targetObject client.Object) (map[string]corev1.ResourceRequirements, error) {
	currentResources := make(map[string]corev1.ResourceRequirements)

	extractResources := func(containers []corev1.Container) {
		for _, container := range containers {
			currentResources[container.Name] = container.Resources
		}
	}

	switch resource := targetObject.(type) {
	case *appsv1.Deployment:
		extractResources(resource.Spec.Template.Spec.Containers)
	case *appsv1.StatefulSet:
		extractResources(resource.Spec.Template.Spec.Containers)
	case *appsv1.DaemonSet:
		extractResources(resource.Spec.Template.Spec.Containers)
	case *batchv1.CronJob:
		extractResources(resource.Spec.JobTemplate.Spec.Template.Spec.Containers)
	case *batchv1.Job:
		extractResources(resource.Spec.Template.Spec.Containers)
	case *appsv1.ReplicaSet:
		extractResources(resource.Spec.Template.Spec.Containers)
	default:
		return nil, fmt.Errorf("unsupported target resource type: %T", targetObject)
	}

	return currentResources, nil
}

func (r *VerticalWorkloadAutoscalerReconciler) updateTargetResource(ctx context.Context, targetObject client.Object, newResources map[string]corev1.ResourceRequirements) (bool, error) {
	needsUpdate := false

	updateContainers := func(containers []corev1.Container) {
		for _, container := range containers {
			if recommendedResources, ok := newResources[container.Name]; ok {
				if !resourceRequirementsEqual(container.Resources, recommendedResources) {
					recommendedResources.Requests.DeepCopyInto(&container.Resources.Requests)
					recommendedResources.Limits.DeepCopyInto(&container.Resources.Limits)
					needsUpdate = true
				}
			}
		}
	}

	switch resource := targetObject.(type) {
	case *appsv1.Deployment:
		updateContainers(resource.Spec.Template.Spec.Containers)
	case *appsv1.StatefulSet:
		updateContainers(resource.Spec.Template.Spec.Containers)
	case *batchv1.CronJob:
		updateContainers(resource.Spec.JobTemplate.Spec.Template.Spec.Containers)
	case *batchv1.Job:
		updateContainers(resource.Spec.Template.Spec.Containers)
	case *appsv1.ReplicaSet:
		updateContainers(resource.Spec.Template.Spec.Containers)
	case *appsv1.DaemonSet:
		updateContainers(resource.Spec.Template.Spec.Containers)
	default:
		return false, errors.NewBadRequest(fmt.Sprintf("unsupported target object type: %T", targetObject))
	}

	if needsUpdate {
		if err := r.Update(ctx, targetObject); err != nil {
			return false, errors.NewInternalError(fmt.Errorf("failed to update target object: %w", err))
		}
	}
	return needsUpdate, nil
}

func (r *VerticalWorkloadAutoscalerReconciler) updateAnnotations(ctx context.Context, targetObject client.Object) error {
	annotations := targetObject.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["verticalworkloadautoscaler.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
	annotations["argocd.argoproj.io/compare-options"] = "IgnoreResourceRequests"
	annotations["fluxcd.io/ignore"] = "true"
	targetObject.SetAnnotations(annotations)
	if err := r.Update(ctx, targetObject); err != nil {
		return err
	}
	return nil
}
