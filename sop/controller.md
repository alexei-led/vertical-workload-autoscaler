# WorkloadAutoscaler Controller Implementation Steps

This document outlines the detailed steps required to implement the WorkloadAutoscaler controller logic and helper functions.

## 1. Implement the Reconcile Function

- [ ] Open `internal/controller/workloadautoscaler_controller.go`
- [ ] Fetch the WorkloadAutoscaler object
- [ ] Fetch the target resource (Deployment, StatefulSet, CronJob, or DaemonSet)
- [ ] Fetch the associated VPA object
- [ ] Check if an update is needed based on VPA recommendations and WorkloadAutoscaler configuration
- [ ] If an update is needed:
  - [ ] Calculate new resource values
  - [ ] Update the target resource
  - [ ] Force pod recreation by updating the `spec.template.metadata.annotations` with a timestamp or unique value, e.g., `workloadautoscaler.kubernetes.io/restartedAt: <current-timestamp>`
  - [ ] Add ArgoCD annotation to the target resource to prevent conflicts:
    ```yaml
    metadata:
      annotations:
        argocd.argoproj.io/compare-options: IgnoreResourceRequests
    ```
- [ ] Update WorkloadAutoscaler status

## 2. Implement Helper Functions

### 2.1. VPA Interaction Functions

- [ ] Create `internal/controller/vpa.go`
- [ ] Implement functions to:
  - [ ] Fetch VPA objects
  - [ ] Parse VPA recommendations

### 2.2. Resource Calculation and Update Functions

- [ ] Create `internal/controller/resources.go`
- [ ] Implement functions to:
  - [ ] Calculate new resource values based on VPA recommendations
  - [ ] Update resource requirements for target resources

### 2.3. Update Checker Functions

- [ ] Create `internal/controller/update_checker.go`
- [ ] Implement functions to:
  - [ ] Determine if an update is allowed based on WorkloadAutoscaler configuration
  - [ ] Check allowed update windows
  - [ ] Check grace periods

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