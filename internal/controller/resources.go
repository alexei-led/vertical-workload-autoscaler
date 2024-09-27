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

func (r *VerticalWorkloadAutoscalerReconciler) fetchTargetResource(ctx context.Context, vpa *vpav1.VerticalPodAutoscaler) (client.Object, error) {
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

// calculateNewResources calculates the new resource requirements based on the VPA recommendations
// and the VWA configuration (tolerance, quality of service, etc.)
func (r *VerticalWorkloadAutoscalerReconciler) calculateNewResources(wa vwav1.VerticalWorkloadAutoscaler, recommendations *vpav1.RecommendedPodResources) map[string]corev1.ResourceRequirements {
	newResources := make(map[string]corev1.ResourceRequirements)

	cpuTolerance := 0.10    // default 10%
	memoryTolerance := 0.10 // default 10%

	if wa.Spec.UpdateTolerance != nil {
		if wa.Spec.UpdateTolerance.CPU > 0 {
			cpuTolerance = float64(wa.Spec.UpdateTolerance.CPU) / 100
		}
		if wa.Spec.UpdateTolerance.Memory > 0 {
			memoryTolerance = float64(wa.Spec.UpdateTolerance.Memory) / 100
		}
	}

	for _, containerRec := range recommendations.ContainerRecommendations {
		newReq := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{},
			Limits:   corev1.ResourceList{},
		}

		applyUpdate := func(current, recommended resource.Quantity, tolerance float64) bool {
			if current.IsZero() {
				return true
			}
			change := float64(recommended.MilliValue()-current.MilliValue()) / float64(current.MilliValue())
			return change >= tolerance || change <= -tolerance
		}

		if wa.Spec.QualityOfService == vwav1.GuaranteedQualityOfService {
			if applyUpdate(containerRec.Target[corev1.ResourceCPU], containerRec.Target[corev1.ResourceCPU], cpuTolerance) {
				newReq.Requests[corev1.ResourceCPU] = containerRec.Target[corev1.ResourceCPU]
				if !wa.Spec.AvoidCPULimit {
					newReq.Limits[corev1.ResourceCPU] = containerRec.Target[corev1.ResourceCPU]
				}
			}
			if applyUpdate(containerRec.Target[corev1.ResourceMemory], containerRec.Target[corev1.ResourceMemory], memoryTolerance) {
				newReq.Requests[corev1.ResourceMemory] = containerRec.Target[corev1.ResourceMemory]
				newReq.Limits[corev1.ResourceMemory] = containerRec.Target[corev1.ResourceMemory]
			}
		} else if wa.Spec.QualityOfService == vwav1.BurstableQualityOfService {
			lowerBoundCPU := containerRec.LowerBound[corev1.ResourceCPU]
			lowerBoundMemory := containerRec.LowerBound[corev1.ResourceMemory]
			upperBoundCPU := containerRec.UpperBound[corev1.ResourceCPU]
			upperBoundMemory := containerRec.UpperBound[corev1.ResourceMemory]

			if applyUpdate(containerRec.Target[corev1.ResourceCPU], lowerBoundCPU, cpuTolerance) {
				newReq.Requests[corev1.ResourceCPU] = lowerBoundCPU
				if !wa.Spec.AvoidCPULimit {
					newReq.Limits[corev1.ResourceCPU] = upperBoundCPU
				}
			}
			if applyUpdate(containerRec.Target[corev1.ResourceMemory], lowerBoundMemory, memoryTolerance) {
				newReq.Requests[corev1.ResourceMemory] = lowerBoundMemory
				newReq.Limits[corev1.ResourceMemory] = upperBoundMemory
			}
		}

		newResources[containerRec.ContainerName] = newReq
	}

	return newResources
}

func resourceRequirementsEqual(a, b corev1.ResourceRequirements) bool {
	return a.Requests.Cpu().Equal(*b.Requests.Cpu()) &&
		a.Requests.Memory().Equal(*b.Requests.Memory()) &&
		a.Limits.Cpu().Equal(*b.Limits.Cpu()) &&
		a.Limits.Memory().Equal(*b.Limits.Memory())
}

func (r *VerticalWorkloadAutoscalerReconciler) updateTargetResource(ctx context.Context, targetResource client.Object, newResources map[string]corev1.ResourceRequirements) (bool, error) {
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

	switch resource := targetResource.(type) {
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
		return false, errors.NewBadRequest(fmt.Sprintf("unsupported target resource type: %T", targetResource))
	}

	if needsUpdate {
		if err := r.Update(ctx, targetResource); err != nil {
			return false, errors.NewInternalError(fmt.Errorf("failed to update target resource: %w", err))
		}
	}
	return needsUpdate, nil
}

func (r *VerticalWorkloadAutoscalerReconciler) updateAnnotations(ctx context.Context, targetResource client.Object) error {
	annotations := targetResource.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["verticalworkloadautoscaler.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
	annotations["argocd.argoproj.io/compare-options"] = "IgnoreResourceRequests"
	annotations["fluxcd.io/ignore"] = "true"
	targetResource.SetAnnotations(annotations)
	if err := r.Update(ctx, targetResource); err != nil {
		return err
	}
	return nil
}
