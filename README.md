# kube-application-snapshot

Kube Application snapshot is a controller to provide backup and restore option for persitent volumes in kubernetes.

The controller works using volumeSnapshot API supported in kubernetes through csidrivers.

Prerequisites to install and run this controller is to install supported csi-driver based on your StorageClass provider.

## Description
Kube Application Snapshot is a Kubernetes controller that provides automated backup and restore capabilities for Persistent Volumes in Kubernetes clusters. It leverages the [Volume Snapshot API](https://kubernetes.io/docs/concepts/storage/volume-snapshots/) to create and manage point-in-time copies of your persistent data.

- Create volume snapshots on-demand or on schedule
- Restore volumes from existing snapshots
- Support for multiple storage providers through CSI drivers
- Cross-namespace volume restoration

## Getting Started

### Prerequisites
- go version v1.23.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

Before installing this controller, ensure you have:

1. A Kubernetes cluster with Volume Snapshot API enabled (v1.20+)
2. CSI driver installed that supports volume snapshots
   - [AWS EBS CSI driver](https://github.com/kubernetes-sigs/aws-ebs-csi-driver)
   - [GCE PD CSI driver](https://github.com/kubernetes-sigs/gcp-compute-persistent-disk-csi-driver)
   - [Azure Disk CSI driver](https://github.com/kubernetes-sigs/azuredisk-csi-driver)
   - [Local Path Provisioner](https://github.com/rancher/local-path-provisioner) (for development/testing)

3. Volume Snapshot CRDs and controller installed. You can install them using:
   ```bash
   make install-snapshot
   ```
Volume Snapshot CRDs is currently only defined for kind cluster and is in progress.

### Test locally 

You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

PVC need to be specified to backup and restore volumes.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/kube-application-snapshot:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/kube-application-snapshot:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

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

Following the options to release and provide this solution to the users.

### By providing a bundle with all YAML files

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/kube-application-snapshot:tag
```

**NOTE:** The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without its
dependencies.

2. Using the installer

Users can just run 'kubectl apply -f <URL for YAML BUNDLE>' to install
the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/kube-application-snapshot/<tag or branch>/dist/install.yaml
```

### By providing a Helm Chart

1. Build the chart using the optional helm plugin

```sh
kubebuilder edit --plugins=helm/v1-alpha
```

2. See that a chart was generated under 'dist/chart', and users
can obtain this solution from there.

**NOTE:** If you change the project, you need to update the Helm Chart
using the same command above to sync the latest changes. Furthermore,
if you create webhooks, you need to use the above command with
the '--force' flag and manually ensure that any custom configuration
previously added to 'dist/chart/values.yaml' or 'dist/chart/manager/manager.yaml'
is manually re-applied afterwards.

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

