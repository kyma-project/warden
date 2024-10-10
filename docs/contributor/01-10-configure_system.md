# System and User Configuration

## System Configuration

To enable Warden on a system namespace add the `namespaces.warden.kyma-project.io/validate: system` label to the namespace.
For now, we also support the deprecated `namespaces.warden.kyma-project.io/warden: enabled` equivalent label.

The Warden configuration for the system mode is defined in [configmap.yaml](../../charts/warden/templates/configmap.yaml) in the section `data/config.yaml`.
You can set the following properties:

| Name                                 | Description                                                                                                                                                                                                                 | Default value                                |
|--------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------|
| `notary.URL`                         | URL of the Notary server used for image verification.                                                                                                                                                                       | "https://signing-dev.repositories.cloud.sap" |
| `notary.allowedRegistries`           | Comma-separated list of allowed registry prefixes.                                                                                                                                                                        | ""                                           |
| `notary.timeout`                     | Timeout for the Notary server connection.                                                                                                                                                                                       | "30s"                                        |
| `admission.systemNamespace`          | Namespace where the Warden admission controller is deployed.                                                                                                                                                                | "default"                                    |
| `admission.serviceName`              | Name of the Warden admission controller service.                                                                                                                                                                            | "warden-admission"                           |
| `admission.secretName`               | Name of the Secret containing the certificate for the Warden admission controller.                                                                                                                                          | "warden-admission-cert"                      |
| `admission.port`                     | Port on which the Warden admission controller listens.                                                                                                                                                                      | 8443                                         |
| `admission.timeout`                  | Timeout for the Warden admission controller.                                                                                                                                                                                | "2s"                                         |
| `admission.strictMode`               | If set to `true`, Warden rejects all images when the Notary server is unavailable. If set to `false`, Warden adds the label `pods.warden.kyma-project.io/validate: pending` to the Pod and retries the validation later. | "false"                                      |
| `operator.metricsBindAddress`        | Address on which the Warden operator serves Prometheus metrics.                                                                                                                                                             | ":8080"                                      |
| `operator.healthProbeBindAddress`    | Address on which the Warden operator serves health probes.                                                                                                                                                                  | ":8081"                                      |
| `operator.leaderElect`               | If set to `true`, Warden operator uses leader election for high availability.                                                                                                                                           | false                                        |
| `operator.podReconcilerRequeueAfter` | Time after which the pod reconciler re-queues the Pods that failed the validation.                                                                                                                                               | "1h"                                         |
| `logging.level`                      | Log level for Warden.                                                                                                                                                                                                       | "info"                                       |
| `logging.format`                     | Log format for Warden.                                                                                                                                                                                                      | "text"                                       |

## User Configuration

For the user configuration see [User Configuration](../user/01-10-configure-user.md).