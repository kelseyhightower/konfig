# Reference Syntax

References to Kubernetes configmaps and secrets can be made when defining Cloud Functions environment variables using the following syntax with a slight variance between regional and zonal GKE clusters.

Regional GKE Clusters

```
$SecretKeyRef:{name=projects/*/locations/*/clusters/*}/{namespaces/*/secrets/*/keys/*}
```

```
$ConfigMapKeyRef:{name=projects/*/locations/*/clusters/*}/{namespaces/*/configmaps/*/keys/*}
```

Zonal GKE Clusters

```
$SecretKeyRef:{name=projects/*/zones/*/clusters/*}/{namespaces/*/secrets/*/keys/*}
```

```
$ConfigMapKeyRef:{name=projects/*/zones/*/clusters/*}/{namespaces/*/configmaps/*/keys/*}
```

### Options

* `tempFile` - When set the value of the configmap or secret key is written to a temp file instead of the env var. The env var is set to the full path to the temp file and can be used to load the file during normal program execution.

```
$SecretKeyRef:{name=projects/*/zones/*/clusters/*}/{namespaces/*/secrets/*/keys/*}?tempFile=true
```

```
$ConfigMapKeyRef:{name=projects/*/zones/*/clusters/*}/{namespaces/*/configmaps/*/keys/*}?tempFile=true
```

## Usage Examples

### Secrets

Referencing zonal clusters.

Reference the `foo` key in the `env` secret in the `default` namespace in the `k0` GKE zonal cluster running in the `us-central1-a` zone:

```
$SecretKeyRef:/projects/hightowerlabs/zones/us-central1-a/clusters/k0/namespaces/default/secrets/env/keys/foo
```

Referencing regional clusters.

Reference the `foo` key in the `env` secret in the `default` namespace in the `k0` GKE regional cluster running in the `us-central1` region:

```
$SecretKeyRef:/projects/hightowerlabs/locations/us-central1/clusters/k0/namespaces/default/secrets/env/keys/foo
```

### ConfigMaps

Referencing zonal clusters.

Reference the `environment` key in the `env` configmap in the `default` namespace in the `k0` GKE zonal cluster running in the `us-central1-a` zone:

```
$ConfigMapKeyRef:/projects/hightowerlabs/zones/us-central1-a/clusters/k0/namespaces/default/configmaps/env/keys/environment
```

### Using the tempfile option.

Write the `config.json` secret to a temp file and store the fully qualified file path in the `CONFIG_FILE` env var.

```
CONFIG_FILE=$SecretKeyRef:/projects/hightowerlabs/zones/us-central1-a/clusters/k0/namespaces/default/secrets/env/keys/config.json?tempFile=true
```

> Notice the `tempFile=true` option is appended to the secret reference.

Assuming the value of the `config.json` secret key is:

```
{
  "database": {
    "username": "user",
    "password": "123456789"
  }
}
```

After processing the secret reference, the value of the `CONFIG_FILE` env var would be the fully qualified file path to a temp file holding the value of the `config.json` secret key:

```
CONFIG_FILE=/tmp/813067742
```

The temp file can be read during normal program execution:

```
cat $CONFIG_FILE
```

```
{
  "database": {
    "username": "user",
    "password": "123456789"
  }
}
```
