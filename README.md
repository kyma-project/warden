# Warden

K8s image authenticity validator

## Status

[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/warden)](https://api.reuse.software/info/github.com/kyma-project/warden)

## Description

Image signing and image signature verification is an important countermeasure against security threats.
While image signing happens in the automated CD workflow, Warden realizes image signature verification happening in the k8s cluster.

Warden allows for configuring a target notary service (via Helm values) and utilizes the Kubernetes label selector mechanism to look for protected namespaces (`namespaces.warden.kyma-project.io/validate: enabled`).

If an image was not signed by the configured notary service, and it is used to schedule a pod in the protected namespace, the pod admission will be rejected.

## Getting Started

You must have a Kubernetes cluster to run against. You can use [kind](https://sigs.k8s.io/kind) to get a local cluster for testing or run against a remote cluster.

### How it Works

Warden realizes image verification by its two components:

 -  Warden admission  -  intercepts scheduling of any pods into the protected namespaces and rejects it if notary service indicates that the image was not signed at all or signing is invalid. If the signature cannot be verified at that stage, the verification status is set to `PENDING`. 

 - Warden operator - a controller that watches already scheduled pods and verifies their signature if the signature status has not been determined (for example, because of a temporary downtime of notary service).

### Run locally
Install Helm charts:

```sh
make install
```

Install Helm charts on the k3d instance with locally built images:

```sh
make install-local
```

Uninstall Helm charts:

```sh
make uninstall
```

### Modifying the API definitions

If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make help` for more information on all potential `make` targets.

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

### Test strategy

Run unit tests
```sh
make verify
```
Start the k3d instance locally and run integration tests
```sh
make k3d-integration-test
```

If you have the k8s instance with Warden installed already, run integration tests with the following command:
```sh
make run-integration-tests
```

### How to release new version

In this project we follow git flow. Every minor release is maintaned in its separate branch.

If you want to patch a version, cherry pick all fix commits into the release branch.

If you want to create a new release, create a new branch from main for the release.

Run a [`create release`](https://github.com/kyma-project/warden/actions/workflows/create-release.yaml) action providing the release name (semantic version `x.x.x`; no `v` prefix) and selecting release branch.

## Contributing

See the [Contributing Rules](CONTRIBUTING.md).

## Code of Conduct

See the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## Licensing

See the [license](./LICENSE) file.
