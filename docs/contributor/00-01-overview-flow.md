# Warden Flow

Warden works during the Pod create and update operations. It also reconciles Pods that are already running in the cluster.

## Verification Modes

Warden protects namespaces in two modes: system and user. 
The system mode is designed for the internal Kyma operation namespaces. These namespaces are now marked (labeled) automatically by Lifecycle Manager (`kyma-system` namespace) and Istio (namespace for Istio).
The user mode can be enabled on any namespace and is controlled by the user.
Both of these modes are exclusive.

Warden watches only namespaces with the `namespaces.warden.kyma-project.io/validate` label.
When such a label is changed, Warden reconfigures itself to work in the mode defined by the label. 
After the label is removed, Warden stops watching the namespace and Pods in these namespaces are not verified anymore.

For more information on configuring modes, see the [configuration](01-10-configure_system.md) section.

## How It Works

For the module overview, see [Warden Module](../user/README.md).
