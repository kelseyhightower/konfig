# konfig

konfig enables serverless workloads running on GCP to reference Kubernetes configmaps and secrets stored in GKE clusters at runtime. konfig currently supports Cloud Functions workloads.

## Usage

konfig is enabled via a single import statement:

```
import (
    ...

    _ "github.com/kelseyhightower/konfig"
)
```

## How Does it Work

The side effect of importing the `konfig` library will cause konfig to:

* call the Cloud Functions API to get a list of env vars to process. We avoid scanning the running environment as any library can set env vars before konfig runs.
* retrieve the GKE endpoint based on the secret or configmap reference
* retrieve configmap and secret keys from the GKE cluster using the service account provided to the Cloud Function instance.
* substitute the reference string with the value of the configmap or secret key.

References to Kubernetes configmaps and secrets can be made when defining Cloud Functions environment variables using the [reference syntax](docs/reference-syntax.md).

## Tutorials

A GKE cluster is used to store configmaps and secrets referenced by Cloud Function workloads. Ideally an existing cluster can be used. For the purpose of this tutorial create the smallest GKE cluster possible in the `us-central1-a` zone:

```
gcloud container clusters create k0 \
  --cluster-version latest \
  --no-enable-basic-auth \
  --no-enable-ip-alias \
  --metadata disable-legacy-endpoints=true \
  --no-issue-client-certificate \
  --num-nodes 1 \
  --machine-type g1-small \
  --scopes gke-default \
  --zone us-central1-a
```

Download the credentials for the `k0` cluster:

```
gcloud container clusters get-credentials k0 \
  --zone us-central1-a
```

We only need the Kubernetes API server as we only plan to use Kubernetes as an secrets and config store, so delete the default node pool.

```
gcloud container node-pools delete default-pool \
  --cluster k0 \
  --zone us-central1-a
```

With the `k0` GKE cluster in place it's time to create the secrets that will be referenced later in the tutorial.  

```
cat > config.json <<EOF
{
  "database": {
    "username": "user",
    "password": "123456789"
  }
}
EOF
```

Create the `env` secret with two keys `foo` and `config.json` which holds the contents of the configuration file created in the previous step:

```
kubectl create secret generic env \
  --from-literal foo=bar \
  --from-file config.json
```

Create the `env` configmap with a single key `environment`:

```
kubectl create configmap env \
  --from-literal environment=production
```

At this point the `env` secret and configmap can be referenced from Cloud Functions using the `konfig` library.

In this section Cloud Functions will be used to deploy the env function which responds to HTTP requests with the contents of the `ENVIRONMENT`, `FOO` and `CONFIG_FILE` environment variables, which reference the `env` secret and configmap created in the previous section.

A GKE cluster ID is required when referencing configmaps and secrets. Extract the cluster ID for the `k0` GKE cluster:

```
CLUSTER_ID=$(gcloud container clusters describe k0 \
  --zone us-central1-a \
  --format='value(selfLink)')
```

Strip the `https://container.googleapis.com/v1` from the previous response and store the results:

```
CLUSTER_ID=${CLUSTER_ID#"https://container.googleapis.com/v1"}
```

> The CLUSTER_ID env var should hold the fully qualified path to the k0 cluster. Assuming `hightowerlabs` as the project ID the value would be `/projects/hightowerlabs/zones/us-central1-a/clusters/k0`.

### Cloud Functions Tutorial

konfig pulls referenced secrets and configmaps from GKE clusters using the GCP service account assigned to a Cloud Function. Create the `konfig` service account with the following IAM roles:

* roles/iam.serviceAccountTokenCreator
* roles/cloudfunctions.viewer
* roles/container.viewer

```
PROJECT_ID=$(gcloud config get-value core/project)
```

```
SERVICE_ACCOUNT_NAME="konfig"
```

```
gcloud iam service-accounts create ${SERVICE_ACCOUNT_NAME} \
  --quiet \
  --display-name "konfig service account"
```

```
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --quiet \
  --member="serviceAccount:${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role='roles/iam.serviceAccountTokenCreator'
```

```
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --quiet \
  --member="serviceAccount:${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role='roles/cloudfunctions.viewer'
```

```
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --quiet \
  --member="serviceAccount:${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role='roles/container.viewer'
```

Enable the `konfig` GCP service account to access the `env` secret and configmap created in previous section:

```
SERVICE_ACCOUNT_EMAIL="konfig@${PROJECT_ID}.iam.gserviceaccount.com"
```

Create the `konfig` role in the `k0` GKE cluster:

```
kubectl create role konfig \
  --verb get \
  --resource secrets \
  --resource configmaps \
  --resource-name env
```

Bind the `konfig` GCP service account and `konfig` role:

```
kubectl create rolebinding konfig \
  --role konfig \
  --user ${SERVICE_ACCOUNT_EMAIL}
```

At this point the `konfig` GCP service account has access to the configmap and secret named `env` in the default namespace in the `k0` GKE cluster.

> The `konfig` Kubernetes role limits the `konfig` GCP service to the defined `env` secret and configmap in a single namespace. Access to additional secrets and configmaps will require additional permissions.

Deploy the `env` function.

```
cd examples/cloudfunctions/env/
```

```
gcloud alpha functions deploy env \
  --entry-point F \
  --max-instances 10 \
  --memory 128MB \
  --region us-central1 \
  --runtime go111 \
  --service-account $SERVICE_ACCOUNT_EMAIL \
  --set-env-vars "FOO=\$SecretKeyRef:${CLUSTER_ID}/namespaces/default/secrets/env/keys/foo,CONFIG_FILE=\$SecretKeyRef:${CLUSTER_ID}/namespaces/default/secrets/env/keys/config.json?tempFile=true,ENVIRONMENT=\$ConfigMapKeyRef:${CLUSTER_ID}/namespaces/default/configmaps/env/keys/environment" \
  --timeout 30s \
  --trigger-http
```

Enable unauthenticated access to the `env` function HTTP endpoint:

```
gcloud alpha functions add-iam-policy-binding env \
  --member allUsers \
  --role roles/cloudfunctions.invoker
```

Retrieve the HTTPS trigger URL:

```
HTTPS_TRIGGER_URL=$(gcloud beta functions describe env \
  --format 'value(httpsTrigger.url)')
```

Make an HTTP request to the `env` function:

```
curl $HTTPS_TRIGGER_URL
```

```
CONFIG_FILE: /tmp/813067742
ENVIRONMENT: production
FOO: bar

# /tmp/813067742
{
  "database": {
    "username": "user",
    "password": "123456789"
  }
}
```
