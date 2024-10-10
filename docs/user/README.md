# Warden Module

## Overview

The Warden module verifies that images used in Pods are signed by the Notary server.
The module is enabled by default on BTP and configured to check the integrity of all Pods in the `kyma-system` namespace.
It offers two validation modes: user and system. The system validation is reserved for internal Kyma operation. The user validation can be enabled on any namespace and is controlled by the user.
For more information on the Warden module flow, see [Warden Flow](00-01-overview-flow.md)

## Useful Links

[User Configuration](tutorials/01-10-configure-user.md)

## Warden Architecture

![Architecture](../assets/user_architecture.svg)

1. User applies Pod on a cluster to the protected namespace.
2. Kubernetes triggers Warden verification.
3. Warden retrieves the signature of the image or images used in the Pod from Notary.
4. Warden retrieves the image or images manifest from the Docker registry.
5. Warden allows (or disallows) creating Pod when the image is properly signed.

### Image Verification

Warden verifies that images used in Pods are signed by the Notary server by comparing the digest of the image in the Docker registry with the digest stored in the Notary server.
For multiplatform images, Warden verifies the digest of the index of images.
