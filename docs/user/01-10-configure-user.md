# User Configuration

To enable the Warden module in your namespace, add the `namespaces.warden.kyma-project.io/validate: user` label to the namespace.
You can configure Warden on each namespace by adding the following annotations to the namespace:

| Name                                                   | Required | Description                                                                                                                                                                                                                 | Default value |
| ------------------------------------------------------ | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| `namespaces.warden.kyma-project.io/notary-url`         | Yes      | URL of the Notary server used for image verification.                                                                                                                                                                       | ""            |
| `namespaces.warden.kyma-project.io/allowed-registries` | No       | Comma-separated list of allowed registry prefixes.                                                                                                                                                                        | ""            |
| `namespaces.warden.kyma-project.io/notary-timeout`     | No       | Timeout for the Notary server connection.                                                                                                                                                                                       | "30s"         |
| `namespaces.warden.kyma-project.io/strict-mode`        | No       | If set to `true`, Warden rejects all images when the Notary server is unavailable. If set to `false`, Warden adds the label `pods.warden.kyma-project.io/validate: pending` to the Pod and retries the validation later. | "true"        |

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
