# Defining and Generating the VerticalWorkloadAutoscaler CRD

This document outlines the steps to define the VerticalWorkloadAutoscaler Custom Resource Definition (CRD) and generate the necessary manifests and code.

## 1. Define the VerticalWorkloadAutoscaler CRD

- [x] Open the file `api/v1alpha1/workloadautoscaler_types.go`
- [x] Update the `VerticalWorkloadAutoscaler` struct:
  - [x] Add a field for the VPA reference
  - [x] Add a field for update frequency
  - [x] Add a field for allowed update windows
  - [x] Add a field for step size
  - [x] Add a field for grace period
  - [x] Add a field for quality of service
  - [x] Add a field to avoid CPU limit
- [x] Update the `VerticalWorkloadAutoscaler` struct:
  - [x] Add fields to track the current state of the autoscaler
  - [x] Add detailed status fields:
    - [x] CurrentStatus
    - [x] TargetResource
    - [x] LastUpdated
    - [x] CurrentRequests
    - [x] RecommendedRequests
    - [x] SkippedUpdates
    - [x] SkipReason
    - [x] StepSize
    - [x] Errors
    - [x] UpdateCount
    - [x] Conditions
    - [x] QualityOfService
- [x] Add any necessary comments for kubebuilder markers

## 2. Generate CRD Manifests and Code

- [x] Open a terminal and navigate to the project root directory
- [x] Run the command to generate CRD manifests:

```
bash
make manifests
```

- [x] Verify that the CRD manifest has been updated in `config/crd/bases/autoscaling.k8s.io_workloadautoscalers.yaml`
- [x] Run the command to generate code:

```
bash
make generate
```

- [x] Verify that the generated code has been updated in `api/v1alpha1/zz_generated.deepcopy.go`

## 3. Review and Adjust

- [x] Review the generated CRD manifest in `config/crd/bases/autoscaling.k8s.io_workloadautoscalers.yaml`
- [x] Make any necessary adjustments to the `workloadautoscaler_types.go` file
- [x] If changes were made, repeat steps 2 and 3 until satisfied with the result

## 4. Update Sample CR

- [x] Open the file `config/samples/autoscaling.k8s.io_v1alpha1_workloadautoscaler.yaml`
- [x] Update the sample CR to include example values for all fields defined in the `VerticalWorkloadAutoscaler`

## 5. Commit Changes

- [x] Review all changes made ito the following files:
  - `api/v1alpha1/workloadautoscaler_types.go`
  - `api/v1alpha1/zz_generated.deepcopy.go`
  - `config/crd/bases/autoscaling.k8s.io_workloadautoscalers.yaml`
  - `config/samples/autoscaling.k8s.io_v1alpha1_workloadautoscaler.yaml`
- [x] Commit the changes to version control
