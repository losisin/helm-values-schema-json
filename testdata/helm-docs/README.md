# helm-docs

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

Empty docs used to generate demo files for helm-docs

## Values

### Image

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| image.pullPolicy | string | `"Always"` | Image pull policy |
| image.repository | string | `"nginx"` | Docker image name |
| image.tag | string | Defaults to chart `appVersion` | Docker image tag |

### Other Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| labels."kubernetes.io/hostname" | string | `"localhost"` | Common label for Kubernetes Node hostname |
| labels.app | string | `"my-app"` | App name |
| replicas | integer | `1` | Number of replicas |
| service | string | `"ClusterIP"` | Kubernetes Service type |

