# Warden architecture

<!-- TODO-doc: it's only draft diagram -->
![Architecture](../../assets/user_architecture.svg)

1. Apply pod on cluster to the protected namespace.
2. Kubernetes trigger Warden verification.
3. Warden retrieves signature of image(s) used in pod from Notary.
4. Warden retrieves image(s) manifest from Docker registry.
5. Warden allow (or disallow) create pod when image(s) is properly signed.

## Image verification

Warden verifies that images used in pods are signed by a Notary server by comparing the digest of the image in the Docker registry with the digest stored in the Notary server.
For multiplatform images, Warden verifies the digest of index of images.