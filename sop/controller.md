# VerticalWorkloadAutoscaler Controller Implementation Steps

This document outlines the steps required to implement the VerticalWorkloadAutoscaler controller logic and helper functions.

## 1. Reconcile Function Implementation

### 1.1 Core Reconcile Logic

- [x] Open `internal/controller/workloadautoscaler_controller.go`
- [x] Fetch the `VerticalWorkloadAutoscaler` object.
- [x] Fetch the target resource (e.g., Deployment, StatefulSet, CronJob, DaemonSet) referenced by the `VerticalWorkloadAutoscaler` CRD.
- [x] Fetch the associated `VPA` object.
- [x] Process existing VPA recommendations (on the first run) and watch for future updates.
- [x] Determine if an update is needed based on the VPA recommendations and `VerticalWorkloadAutoscaler` configuration (time windows, step size, etc.).
- [x] If an update is needed:
  - [x] Calculate new resource values (requests/limits) based on StepSize configuration.
  - [x] Update the target resource (e.g., `Deployment`) with new resource values.
  - [x] Force pod recreation by updating `spec.template.metadata.annotations` with a timestamp or unique value, e.g., `workloadautoscaler.kubernetes.io/restartedAt: <timestamp>`.
  - [x] Add the following ArgoCD annotation to prevent conflicts with GitOps tools:

    ```yaml
    metadata:
      annotations:
        argocd.argoproj.io/compare-options: IgnoreResourceRequests
    ```

- [x] If an update is not allowed due to configuration (e.g., outside allowed windows), store the new recommendation and retry later.
- [x] Record retry attempts and success/failure statuses in `VerticalWorkloadAutoscaler` status.
- [x] Update the `VerticalWorkloadAutoscaler` status after each reconcile loop.
- [x] Implement support for delayed updates, including allowed update windows, grace periods, and update frequencies (considering timezones).

### 1.2 HPA Conflict Resolution

- [x] Detect conflicting `HPA` objects:
  - [x] Watch for `HPA` creation, updates, and deletions.
  - [x] List all `HPA` objects in the same namespace as the target resource and filter based on those targeting the same resource (e.g., `Deployment`, `StatefulSet`).
  - [x] Identify HPAs that overlap with VPA on scaling metrics (CPU or memory).
- [x] Skip conflicting VPA recommendations:
  - [x] Skip applying VPA recommendations if a conflicting HPA is detected (e.g., HPA scaling CPU or memory).
  - [x] Log or update the `VerticalWorkloadAutoscaler` status to indicate that VPA recommendations were skipped due to HPA conflict:

    ```yaml
    conflicts:
      - resource: "CPU"
        conflictWith: "HorizontalPodAutoscaler"
        reason: "HPA scales CPU"
    ```

- [ ] Periodically check for conflicts to ensure consistency before applying updates.

## 2. Implement Helper Functions

### 2.1 VPA Interaction Functions

- [x] Create `internal/controller/vpa.go`
- [x] Implement functions to:
  - [x] Fetch VPA objects.
  - [x] Parse VPA recommendations.
  - [x] Watch for VPA recommendation changes.

### 2.2 Resource Calculation and Update Functions

- [x] Create `internal/controller/resources.go`
- [x] Implement functions to:
  - [x] Calculate resource values based on VPA recommendations and `VerticalWorkloadAutoscaler` StepSize configuration.
  - [x] Update resource requirements for the target resource.

### 2.3 Update Checker Functions

- [x] Create `internal/controller/update_checker.go`
- [x] Implement functions to:
  - [x] Check if an update is allowed based on the `VerticalWorkloadAutoscaler` configuration (e.g., time windows, grace period, update frequency).
  - [x] Store and retry later if the update is not allowed.

### 2.4 HPA Conflict Detection Functions

- [x] Create `internal/controller/hpa.go`
- [x] Implement functions to:
  - [x] List HPAs in the same namespace as the target resource.
  - [x] Detect conflicts between HPAs and VPAs.
  - [x] Skip conflicting VPA recommendations when conflicts are detected.
  - [x] Update the `VerticalWorkloadAutoscaler` status with conflict information.

## 3. Testing

### 3.1 Unit Tests for Core Functions

- [x] Create `internal/controller/workloadautoscaler_controller_test.go`
- [x] Write unit tests for the Reconcile function.

### 3.2 Unit Tests for Helper Functions

- [x] Create `internal/controller/vpa_test.go` for VPA interaction functions.
- [x] Create `internal/controller/resources_test.go` for resource calculation functions.
- [ ] Create `internal/controller/update_checker_test.go` for update checker functions.
- [ ] Create `internal/controller/hpa_test.go` for HPA conflict detection functions.

## 4. Integration Testing

### 4.1 End-to-End Tests

- [ ] Update `test/e2e/e2e_test.go`:
  - [ ] Write integration tests for the `VerticalWorkloadAutoscaler` controller's core logic.
  - [ ] Write tests to validate HPA conflict resolution.
