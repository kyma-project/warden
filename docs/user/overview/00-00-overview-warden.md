# Warden overview

Warden verifies that images used in pods are signed by a Notary server.
Warden module is enabled by default on BTP, and configured to check integrity of all pods in the `kyma-system` namespace.
Warden offers two validation modes: user and system. The system validation is reserved for internal Kyma operation. The user validation can be enabled on any namespace and is controlled by the user.

<!-- TODO-doc: is it enabled by default on open-source? -->
