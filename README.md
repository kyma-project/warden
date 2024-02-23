# Warden
K8s image authenticity validator

## Status
[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/warden)](https://api.reuse.software/info/github.com/kyma-project/warden)

## Description

Image signing and image signature verification is an important countermeasure against security threats.
While image signing happens in the automated CD workflow, warden realizes image signature verification happening in the k8s cluster.

Warden allows configuring a target notary service (via helm values) and utilises kubernetes label selector mechanism to lookup for protected namespaces ( `namespaces.warden.kyma-project.io/validate: enabled`).

If an image was not signed by configured notary service, and it is used to schedule a pod in protected namespace, the pod admission will be rejected.



## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.



### How it works

Warden realises image verification by it's two components:

 -  warden admission  -  intercepts scheduling of any pods into the protected namespaces and rejects it if notary services indicates that image was not signed at all or signing is invalid. If the signature cannot be verified at that stage, the verification status is set to `PENDING`. 

 - warden operator - a controller that will watch already scheduled pods and will verify the signature for those where signature status was so far not determined (i.e because of a temporary downtime of notary service).


### Run locally
Install helm charts via:

```sh
make install
```

Install helm charts with locally build images:

```sh
make install-local
```

Uninstall helm charts via:

```sh
make uninstall
```

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

### Test strategy

Run unit tests
```sh
make verify
```
Start k3d instance locally and run integration  tests
```sh
make k3d-integration-test
```

Run integration tests on your current kubeconfig context
```sh
make verify-on-cluster
```

### How to release new version

In this project we follow git flow. Every minor release is maintaned in its separate branch.

If you want to patch a version, cherry pick all fix commits into the release branch.

If you want to create a new release, create a new branch from main for the release.

Run a [`create release`](https://github.com/kyma-project/warden/actions/workflows/create-release.yaml) action providing the release name (semantic version `x.x.x`; no `v` prefix) and selecting release branch.
## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

