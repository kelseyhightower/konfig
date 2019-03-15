# konfig

konfig enables serverless workloads running on GCP to reference Kubernetes configmaps and secrets stored in GKE clusters at runtime. konfig currently supports Cloud Run and Cloud Functions workloads.

## Usage

konfig is enabled via a single import statement:

```
import (
    ...

    _ "github.com/kelseyhightower/konfig"
)
```

At deployment time references to Kubernetes configmaps and secrets can be made when defining environment variables using the [reference syntax](docs/reference-syntax.md).

## Tutorials

A GKE cluster is used to store configmaps and secrets referenced by Cloud Run and Cloud Function workloads. Ideally an existing cluster can be used. For the purpose of this tutorial create the smallest GKE cluster possible in the `us-central1-a` zone:

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

At this point the `env` secret and configmap can be referenced from either Cloud Run or Cloud Functions using the `konfig` library.

### Cloud Run Tutorial

In this section Cloud Run will be used to deploy the `gcr.io/hightowerlabs/env:0.0.1` container image which responds to HTTP requests with the contents of the `ENVIRONMENT`, `FOO` and `CONFIG_FILE` environment variables, which reference the `env` secret and configmap created in the previous section.

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

Create the `env` Cloud Run service and set the `ENVIRONMENT`, `FOO` and `CONFIG_FILE` env vars to reference the `env` configmaps and secrets in the `k0` GKE cluster:

```
gcloud alpha run deploy env \
  --allow-unauthenticated \
  --concurrency 50 \
  --image gcr.io/hightowerlabs/env:0.0.1 \
  --memory 2G \
  --region us-central1 \
  --set-env-vars "FOO=\$SecretKeyRef:${CLUSTER_ID}/namespaces/default/secrets/env/keys/foo,CONFIG_FILE=\$SecretKeyRef:${CLUSTER_ID}/namespaces/default/secrets/env/keys/config.json?tempFile=true,ENVIRONMENT=\$ConfigMapKeyRef:${CLUSTER_ID}/namespaces/default/configmaps/env/keys/environment"
```

> The `CONFIG_FILE` env var reference uses the `tempFile` option to write the contents of the `config.json` secret key to a temp file. The `CONFIG_FILE` env var will hold the path to the temp file which can be read during normal program execution.

Retreive the `env` service HTTP endpoint:

```
ENV_SERVICE_URL=$(gcloud alpha run services describe env \
  --namespace hightowerlabs \
  --region us-central1 \
  --format='value(status.domain)')
```

Make an HTTP request to the `env` service:

```
curl $ENV_SERVICE_URL
```

Output:
```
CONFIG_FILE: /tmp/363780357
ENVIRONMENT: production
FOO: bar

# /tmp/363780357
{
  "database": {
    "username": "user",
    "password": "123456789"
  }
}
```

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

Enable the `konfig` GCP service account to access the `env` secret in the `k0` Kubernetes cluster:

```
SERVICE_ACCOUNT_EMAIL="konfig@${PROJECT_ID}.iam.gserviceaccount.com"
```

```
kubectl create role konfig \
  --verb get \
  --resource secrets \
  --resource configmaps \
  --resource-name env
```

```
kubectl create rolebinding konfig \
  --role konfig \
  --user ${SERVICE_ACCOUNT_EMAIL}
```

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

```
gcloud alpha functions add-iam-policy-binding env \
  --member allUsers \
  --role roles/cloudfunctions.invoker
```

```
HTTPS_TRIGGER_URL=$(gcloud beta functions describe env \
  --format 'value(httpsTrigger.url)')
```

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
