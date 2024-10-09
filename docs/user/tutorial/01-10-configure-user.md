# User configuration

To enable Warden on a user namespace add the `namespaces.warden.kyma-project.io/validate: user` label to the namespace.
User can configure Warden on each namespace by adding annotations to the namespace.
The following annotations can be set:

| Name                                                   | Required | Description                                                                                                                                                                                                                 | Default value |
| ------------------------------------------------------ | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| `namespaces.warden.kyma-project.io/notary-url`         | Yes      | URL of the Notary server used for image verification.                                                                                                                                                                       | ""            |
| `namespaces.warden.kyma-project.io/allowed-registries` | No       | Comma-separated list of allowed registries prefixes.                                                                                                                                                                        | ""            |
| `namespaces.warden.kyma-project.io/notary-timeout`     | No       | Timeout for Notary server connection.                                                                                                                                                                                       | "30s"         |
| `namespaces.warden.kyma-project.io/strict-mode`        | No       | If set to `true`, Warden will reject all images when the Notary server is unavailable. If set to `false`, Warden will add label `pods.warden.kyma-project.io/validate: pending` to the pod and will retry validation later. | "true"        |

# Example

Example namespace configuration verified by Warden:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: my-namespace
  labels:
    namespaces.warden.kyma-project.io/validate: "user"
  annotations:
    namespaces.warden.kyma-project.io/notary-url: "https://notary.example.com"
    namespaces.warden.kyma-project.io/allowed-registries: "registry1.io,registry2.io/nginx"
    namespaces.warden.kyma-project.io/notary-timeout: "30s"
    namespaces.warden.kyma-project.io/strict-mode: "true"
```
