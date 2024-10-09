# Warden flow

Warden can work on pod create, and update operations. It can also reconcile pods that are already running in the cluster.

## Verification modes

Warden is dedicated to protect namespaces in two modes: system and user. 
System mode was designed for internal Kyma operation namespaces. These namespaces now are marked (labeled) automatically by Lifecycle Manager (`kyma-system` namespace) and Istio (namespace for Istio).
User mode can be enabled on any namespace and is controlled by the user.  
Both of these modes are exclusive.

Warden watches only namespaces with the `namespaces.warden.kyma-project.io/validate` label.
When such label is changed, Warden will reconfigure itself to work in the mode defined by the label. 
After the label is removed, Warden will stop watching the namespace and pods in this namespaces will not be verified anymore.

For more information on how to configure modes, see the [configuration](../tutorial/01-10-configure.md) section.

## How it works

For general description see [here](../../user/overview/00-01-overview-flow.md).
