# WorkloadAutoscaler Controller Implementation Steps

This document outlines the detailed steps required to implement the WorkloadAutoscaler controller logic and helper functions.

## 1. Implement the Reconcile Function

- [x] Open `internal/controller/workloadautoscaler_controller.go`
- [x] Fetch the WorkloadAutoscaler object
- [x] Fetch the target resource (Deployment, StatefulSet, CronJob, or DaemonSet) from the VPA configuration
- [x] Fetch the associated VPA object
- [x] Watch for VPA recommendation updates
- [x] For the first run, discover and process existing VPA recommendations
- [x] Check if an update is needed based on VPA recommendations and WorkloadAutoscaler configuration
- [x] If an update is needed:
  - [x] Calculate new resource values based on StepSize configuration (round up to the nearest StepSize bucket)
  - [x] Update the target resource
  - [x] Force pod recreation by updating the `spec.template.metadata.annotations` with a timestamp or unique value, e.g., `workloadautoscaler.kubernetes.io/restartedAt: <current-timestamp>`
  - [x] Add ArgoCD annotation to the target resource to prevent conflicts:

    ```yaml
    metadata:
      annotations:
        argocd.argoproj.io/compare-options: IgnoreResourceRequests
    ```

- [x] If the update is not allowed right now, keep the recommended value and retry until allowed and successful
- [x] Record progress (retry attempts and success or failure) statuses on the WorkloadAutoscaler object status
- [x] Update WorkloadAutoscaler status
- [x] Implement support for delayed updates based on allowed update windows, update frequency, and grace period, considering timezones. If no allowed update windows are set, update immediately.

## 2. Implement Helper Functions

### 2.1. VPA Interaction Functions

- [x] Create `internal/controller/vpa.go`
- [x] Implement functions to:
  - [x] Fetch VPA objects
  - [x] Parse VPA recommendations
  - [x] Watch for VPA recommendation updates

### 2.2. Resource Calculation and Update Functions

- [x] Create `internal/controller/resources.go`
- [x] Implement functions to:
  - [x] Calculate new resource values based on VPA recommendations and StepSize configuration
  - [x] Update resource requirements for target resources

### 2.3. Update Checker Functions

- [x] Create `internal/controller/update_checker.go`
- [x] Implement functions to:
  - [x] Determine if an update is allowed based on WorkloadAutoscaler configuration (time windows, grace periods, update frequency)
  - [x] If the update is not allowed, wait and retry (make wait time configurable)

## 3. Testing

### 3.1. Unit Tests

- [ ] Create `internal/controller/workloadautoscaler_controller_test.go`
- [ ] Write unit tests for the Reconcile function

### 3.2. Helper Function Tests

- [ ] Create `internal/controller/vpa_test.go`
- [ ] Write unit tests for VPA interaction functions
- [ ] Create `internal/controller/resources_test.go`
- [ ] Write unit tests for resource calculation and update functions
- [ ] Create `internal/controller/update_checker_test.go`
- [ ] Write unit tests for update checker functions

## 4. Integration Tests

- [ ] Update `test/e2e/e2e_test.go`
- [ ] Write integration tests for the WorkloadAutoscaler controller
