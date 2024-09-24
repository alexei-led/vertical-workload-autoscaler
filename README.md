# vertical-workload-autoscaler

VerticalWorkloadAutoscaler (VWA) is a Kubernetes-native solution designed to enhance Vertical Pod Autoscaler (VPA) functionality by providing configurable and controlled resource updates for workloads. It offers granular control over update windows, step sizes, and compatibility with Horizontal Pod Autoscalers (HPA), KEDA, StatefulSets, and DaemonSets. The solution avoids immediate pod evictions, ensuring smooth resource adjustments for improved performance and cost efficiency.

## Description

VerticalWorkloadAutoscaler extends the capabilities of the Vertical Pod Autoscaler (VPA) by introducing more control over how and when resource updates are applied to your workloads. This includes defining specific update windows, step sizes for resource adjustments, and ensuring compatibility with other Kubernetes components like HPA, KEDA, StatefulSets, and DaemonSets. This approach helps in maintaining performance and cost efficiency without causing disruptions due to immediate pod evictions.

## Key Features

- **Controlled Updates**: Instead of immediate pod evictions, the Workload Autoscaler updates the resource requests in the Deployment/StatefulSet/DaemonSet spec, triggering controlled pod updates based on specified configurations.
- **Configurable Parameters**:
  - **Frequency of Updates**: Configure how often updates are applied to avoid too frequent changes.
  - **Allowed Update Windows**: Define time windows during which updates are allowed, reducing the risk of applying changes during peak usage times.
  - **Step Size**: Set the increment or decrement size for resource changes to avoid frequent minor adjustments and ensure updates are applied in meaningful steps.
  - **Grace Period**: Specify a grace period before applying changes to minimize disruptions.
  - **Min/Max Boundaries**: Define minimum and maximum resource limits to prevent resource exhaustion or over-provisioning.
  - **Compatibility Checks**: Ensure compatibility with HPA, KEDA, StatefulSets, and DaemonSets to avoid conflicts and manage scaling effectively.
  - **Timezone Support**: Ensure that allowed update windows are respected according to the specified timezones. If no allowed update windows are set, updates happen immediately.

## Getting Started

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

2. Using the installer

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

## License

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
