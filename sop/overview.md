# WorkloadAutoscaler Controller Implementation Overview

This document outlines the steps required to implement a working version of the WorkloadAutoscaler controller.

## 1. Define the WorkloadAutoscaler Custom Resource Definition (CRD)

- [x] Define the WorkloadAutoscaler spec and status in `api/v1alpha1/workloadautoscaler_types.go`
- [x] Ensure the spec includes fields for:
  - [x] Target resource (using selectors or direct references)
  - [x] VPA reference
  - [x] Update frequency
  - [x] Allowed update windows
  - [x] Step size
  - [x] Grace period
- [x] Ensure the status includes detailed fields for tracking the state of the autoscaler

## 2. Generate CRD manifests and code

- [x] Run `make manifests` to generate CRD manifests
- [x] Run `make generate` to update generated code

## 3. Implement the controller logic

Update `internal/controller/workloadautoscaler_controller.go`:

- [ ] Implement the Reconcile function:
  - [ ] Fetch the WorkloadAutoscaler object
  - [ ] Fetch the target resource (Deployment, StatefulSet, CronJob, or DaemonSet)
  - [ ] Fetch the associated VPA object
  - [ ] Check if an update is needed based on VPA recommendations and WorkloadAutoscaler configuration
  - [ ] If an update is needed:
    - [ ] Calculate new resource values
    - [ ] Update the target resource
    - [ ] Force pod recreation by updating a specific attribute
  - [ ] Update WorkloadAutoscaler status

## 4. Implement helper functions

Create new files in the `internal/controller/` directory:

- [ ] `vpa.go`: Functions to interact with VPA objects
- [ ] `resources.go`: Functions to calculate and update resource requirements
- [ ] `update_checker.go`: Functions to determine if an update is allowed based on configuration

## 5. Update RBAC permissions

- [ ] Modify `config/rbac/role.yaml` to include necessary permissions:
  - [ ] Add permissions to watch and modify Deployments, StatefulSets, CronJobs, and DaemonSets
  - [ ] Add permissions to read VPA objects

## 6. Implement logging

- [ ] Use the controller-runtime's logger to log all actions and decisions

## 7. Update the main.go file

- [ ] Modify `cmd/main.go` to set up the controller with the manager

## 8. Write unit tests

Create test files in the `internal/controller/` directory:

- [ ] `workloadautoscaler_controller_test.go`
- [ ] `vpa_test.go`
- [ ] `resources_test.go`
- [ ] `update_checker_test.go`

## 9. Write integration tests

- [ ] Update `test/e2e/e2e_test.go` to include integration tests for the WorkloadAutoscaler controller

## 10. Update project documentation

- [ ] Update `README.md` with project description and usage instructions
- [ ] Create example WorkloadAutoscaler CR in `config/samples/`

## 11. Build and test

- [ ] Run `make test` to run unit tests
- [ ] Run `make docker-build` to build the controller image
- [ ] Deploy the controller to a test cluster and run integration tests

## 12. Finalize and package

- [ ] Review and refine the implementation
- [ ] Ensure all tests pass
- [ ] Prepare for deployment
