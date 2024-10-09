# Configuration

## System configuration

To enable Warden on a system namespace add the `namespaces.warden.kyma-project.io/validate: system` label to the namespace.
Now we support also depreciated `namespaces.warden.kyma-project.io/warden: enabled` label which is equivalent.

Warden configuration for system mode is defined in [configmap.yaml](../../../charts/warden/templates/configmap.yaml) in section `data/config.yaml`.
The following properties can be set:

| Name                                 | Description                                                                                                                                                                                                                 | Default value                                |
|--------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------|
| `notary.URL`                         | URL of the Notary server used for image verification.                                                                                                                                                                       | "https://signing-dev.repositories.cloud.sap" |
| `notary.allowedRegistries`           | Comma-separated list of allowed registries prefixes.                                                                                                                                                                        | ""                                           |
| `notary.timeout`                     | Timeout for Notary server connection.                                                                                                                                                                                       | "30s"                                        |
| `admission.systemNamespace`          | Namespace where the Warden admission controller is deployed.                                                                                                                                                                | "default"                                    |
| `admission.serviceName`              | Name of the Warden admission controller service.                                                                                                                                                                            | "warden-admission"                           |
| `admission.secretName`               | Name of the secret containing the certificate for the Warden admission controller.                                                                                                                                          | "warden-admission-cert"                      |
| `admission.port`                     | Port on which the Warden admission controller listens.                                                                                                                                                                      | 8443                                         |
| `admission.timeout`                  | Timeout for the Warden admission controller.                                                                                                                                                                                | "2s"                                         |
| `admission.strictMode`               | If set to `true`, Warden will reject all images when the Notary server is unavailable. If set to `false`, Warden will add label `pods.warden.kyma-project.io/validate: pending` to the pod and will retry validation later. | "false"                                      |
| `operator.metricsBindAddress`        | Address on which the Warden operator serves Prometheus metrics.                                                                                                                                                             | ":8080"                                      |
| `operator.healthProbeBindAddress`    | Address on which the Warden operator serves health probes.                                                                                                                                                                  | ":8081"                                      |
| `operator.leaderElect`               | If set to `true`, Warden operator will use leader election for high availability.                                                                                                                                           | false                                        |
| `operator.podReconcilerRequeueAfter` | Time after which the pod reconciler will requeue pods that failed validation.                                                                                                                                               | "1h"                                         |
| `logging.level`                      | Log level for Warden.                                                                                                                                                                                                       | "info"                                       |
| `logging.format`                     | Log format for Warden.                                                                                                                                                                                                      | "text"                                       |

## User configuration

For user configuration see [here](../../user/tutorial/01-10-configure-user.md).