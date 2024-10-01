# VerticalWorkloadAutoscaler (VWA)

The `VerticalWorkloadAutoscaler` (VWA) is a Kubernetes Custom Resource Definition (CRD) designed to manage the vertical scaling of Kubernetes workloads. By integrating with the VerticalPodAutoscaler (VPA), the VWA provides advanced control over resource allocation. It offers features such as scheduled update windows, custom tolerances, and support for various workload types like Deployments, StatefulSets, DaemonSets, CronJobs, and Jobs. The VWA helps optimize resource utilization, improve performance, and reduce costs by dynamically adjusting resource requests based on workload requirements. It can also work in conjunction with HorizontalPodAutoscalers (HPA) and other scaling controllers to prevent conflicts and ensure smooth scaling operations.

## Features

- **Allowed Update Windows**: Define time windows during which updates to resource requests are allowed, minimizing disruptions during peak usage times.
- **Avoid CPU Limit**: Option to avoid setting CPU limits, ensuring only resource requests are adjusted (useful for burstable workloads).
- **Custom Annotations**: Apply custom annotations to the target object, which can prevent GitOps tools from reverting updates made by VWA.
- **Quality of Service (QoS)**: Control the QoS class applied to managed resources, with support for `Guaranteed` and `Burstable` classes.
- **Resource Recommendation Filtering**: Options to ignore CPU or memory recommendations, allowing selective scaling.
- **Conflict Detection**: Track and report conflicts with HorizontalPodAutoscalers (HPA) and other scaling controllers.
- **Update Tolerance**: Fine-tune how sensitive the VWA is to changes in resource requests based on CPU and memory usage.

## CRD Overview

The VWA CRD includes the following key properties:

### `spec`:

- `allowedUpdateWindows`: Specifies time windows during which updates are allowed, minimizing disruptions at critical times.
- `avoidCPULimit`: A boolean field to disable CPU limit settings in the workload.
- `customAnnotations`: Annotations that will be added to the target workload resource.
- `ignoreCPURecommendations`: Disables the CPU-based scaling if set to true.
- `ignoreMemoryRecommendations`: Disables the memory-based scaling if set to true.
- `qualityOfService`: Defines the QoS class ("Guaranteed" or "Burstable") for the managed resources.
- `updateFrequency`: Controls how often the VWA checks and applies updates to resource requests (default: 5 minutes).
- `updateTolerance`: Defines thresholds for ignoring minor changes in CPU and memory recommendations.
- `vpaReference`: References the associated VPA object to manage vertical scaling.

### `status`:

- `recommendedRequests`: The current recommended resource requests for the managed resource.
- `scaleTargetRef`: Reference to the resource being managed (e.g., Deployment, StatefulSet, DaemonSet).
- `conflicts`: Lists any conflicts detected with other autoscalers (e.g., HPA).
- `skippedUpdates`: Indicates if updates were skipped.
- `updateCount`: Total number of updates applied.

## Example Usage

Here's an example of a VerticalWorkloadAutoscaler configuration:

```yaml
apiVersion: autoscaling.workload.io/v1alpha1
kind: VerticalWorkloadAutoscaler
metadata:
  name: example-vwa
  namespace: default
spec:
  vpaReference:
    name: example-vpa
  allowedUpdateWindows:
    - dayOfWeek: Monday
      startTime: "09:00"
      endTime: "17:00"
      timeZone: "America/New_York"
  avoidCPULimit: true
  customAnnotations:
    annotation-key: "annotation-value"
  ignoreCPURecommendations: false
  ignoreMemoryRecommendations: true
  qualityOfService: Guaranteed
  updateFrequency: 10m
  updateTolerance:
    cpu: 0.15  # 15% tolerance for CPU
    memory: 0.20  # 20% tolerance for memory
```

In this example:

- The VWA references a VerticalPodAutoscaler (`example-vpa`).
- Updates are only allowed on Mondays between 9 AM and 5 PM, in the `America/New_York` time zone.
- CPU limits are avoided, and memory recommendations are ignored.
- The VWA will check for updates every 10 minutes.
- CPU and memory requests will only be adjusted if they differ by more than 15% or 20%, respectively.

## Conflict Detection

The VWA will detect conflicts with other autoscaler controllers, such as HorizontalPodAutoscalers (HPA) and KEDA. When a conflict is detected, the VWA will ignore CPU and/or memory recommendations to prevent interference with other scaling controllers that use resource metrics. The VWA will report any conflicts in the `status.conflicts` field.

## Annotations for GitOps Compatibility

The VWA supports adding custom annotations to the target object. This is particularly useful in scenarios where GitOps tools like ArgoCD or Flux continuously manage the cluster state. By adding a specific annotation to the target object, the VWA can prevent these tools from reverting the changes made by the VWA.

## Installation

### VPA Recommender

The VerticalWorkloadAutoscaler requires the VerticalPodAutoscaler (VPA) Recommender to provide resource recommendations. The VPA Recommender should be installed in the cluster before deploying the VWA. You can install the VPA Recommender using Fairwinds' Helm chart:

```sh
helm repo add fairwinds-stable https://charts.fairwinds.com/stable
helm install vpa fairwinds-stable/vpa --version 4.6.0 --set "recommender.enabled=true,updater.enabled=false,admissionController.enabled=false,metrics-server.enabled=true,recommender.image.tag=1.2.1" --namespace vpa --create-namespace
```

The Fairwinds' VPA Helm chart can also install metrics-server, which is required for the VPA Recommender to function correctly.

### Prerequisites

- go version v1.22.0+
- docker version 25.05+.
- kubectl version v1.28+.
- Access to a Kubernetes v1.28+ cluster.

### To Deploy on the cluster

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/vertical-workload-autoscaler:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don't work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/vertical-workload-autoscaler:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples have default values to test it out.

### To Uninstall

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/vertical-workload-autoscaler:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

1. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/vertical-workload-autoscaler/<tag or branch>/dist/install.yaml
```

## Contributing

We welcome contributions to the VerticalWorkloadAutoscaler project. Please follow these steps to contribute:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature-branch`).
3. Make your changes.
4. Commit your changes (`git commit -am 'Add new feature'`).
5. Push to the branch (`git push origin feature-branch`).
6. Create a new Pull Request.

**NOTE:** Run `make help` for more information on all potential `make` targets.

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html).
