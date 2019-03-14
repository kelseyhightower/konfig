# Reference Syntax

At deployment time references to Kubernetes secrets can be made when defining environment variables using the following syntax with a slight variance between regional and zonal GKE clusters.

Regional GKE Clusters

```
$SecretKeyRef:{name=projects/*/locations/*/clusters/*}/{namespaces/*/secrets/*/keys/*}
```

Zonal GKE Clusters

```
$SecretKeyRef:{name=projects/*/zones/*/clusters/*}/{namespaces/*/secrets/*/keys/*}
```

## Usage Examples

Referencing zonal clusters.

To reference the `foo` key in the `env` secret in the `default` namespace in the `k0` GKE zonal cluster running in the `us-central1-a` zone:

```
$SecretKeyRef:/projects/hightowerlabs/zones/us-central1-a/clusters/k0/namespaces/default/secrets/env/keys/foo
```

Referencing regional clusters.

To reference the `foo` key in the `env` secret in the `default` namespace in the `k0` GKE regional cluster running in the `us-central1` region:

```
$SecretKeyRef:/projects/hightowerlabs/locations/us-central1/clusters/k0/namespaces/default/secrets/env/keys/foo
```
