# konfig

konfig enables Serverless workloads running on GCP to reference Kubernetes secrets stored in GKE clusters at runtime. konfig currently supports Cloud Run and Cloud Functions workloads.

## Usage

konfig is enabled via a single import statement:

```
import (
    ...

    _ "github.com/kelseyhightower/konfig"
)
```

At deployment time references to Kubernetes secrets can be made when defining environment variables using the [reference syntax](docs/reference-syntax.md)

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

### Create the `env` secrets

With `k0` GKE cluster in place it's time to create the secrets that will be referenced later in the tutorial.  

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

At this point the `env` secret can be referenced from either Cloud Run or Cloud Functions using the `konfig` library.

### Cloud Run Tutorial

```
CLUSTER_ID=$(gcloud container clusters describe k0 \
  --zone us-central1-a \
  --format='value(selfLink)')
```

```
CLUSTER_ID=${CLUSTER_ID#"https://container.googleapis.com/v1"}
```

```
gcloud alpha run deploy env \
  --allow-unauthenticated \
  --concurrency 50 \
  --image gcr.io/hightowerlabs/env:0.0.1 \
  --memory 2G \
  --region us-central1 \
  --set-env-vars "GOOGLE_CLOUD_REGION=us-central1,FOO=\$SecretKeyRef:${CLUSTER_ID}/namespaces/default/secrets/env/keys/foo,CONFIG_FILE=\$SecretKeyRef:${CLUSTER_ID}/namespaces/default/secrets/env/keys/config.json?tempFile=true"
```

```
ENV_SERVICE_URL=$(gcloud alpha run services describe env \
  --namespace hightowerlabs \
  --region us-central1 \
  --format='value(status.domain)')
```

```
curl -i $ENV_SERVICE_URL
```

Output:
```
HTTP/2 200
config_file: /tmp/env116970659
foo: bar
google_cloud_project: hightowerlabs
google_cloud_region: us-central1
home: /home
k_configuration: env
k_revision: env-6aa1a472-5608-471b-a4cd-6b3a236c9e34
k_service: env
path: /usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
port: 8080
content-type: text/plain; charset=utf-8
x-cloud-trace-context: df0aec2bbdf0df373b1a0248969851d8;o=1
date: Wed, 13 Mar 2019 18:53:40 GMT
server: Google Frontend
content-length: 79
alt-svc: quic=":443"; ma=2592000; v="46,44,43,39"

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
  --service-account $SERVICE_ACCOUNT_EMAIL \
  --set-env-vars "FOO=\$SecretKeyRef:${CLUSTER_ID}/namespaces/default/secrets/env/keys/foo,CONFIG_FILE=\$SecretKeyRef:${CLUSTER_ID}/namespaces/default/secrets/env/keys/config.json?tempFile=true" \
  --max-instances 10 \
  --memory 128MB \
  --region us-central1 \
  --runtime go111 \
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
curl -i $HTTPS_TRIGGER_URL
```

```
HTTP/2 200
code_location: /srv
config_file: /tmp/461942329
content-type: text/plain; charset=utf-8
debian_frontend: noninteractive
entry_point: F
foo: bar
function-execution-id: i1hoszlw0o14
function_identity: konfig@hightowerlabs.iam.gserviceaccount.com
function_memory_mb: 128
function_name: env
function_region: us-central1
function_timeout_sec: 30
function_trigger_type: HTTP_TRIGGER
gcloud_project: hightowerlabs
gcp_project: hightowerlabs
home: /root
node_env: production
path: /usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
port: 8080
pwd: /srv/files/
supervisor_hostname: 169.254.8.129
supervisor_internal_port: 8081
worker_port: 8091
x_google_code_location: /srv
x_google_container_logging_enabled: true
x_google_entry_point: F
x_google_function_identity: konfig@hightowerlabs.iam.gserviceaccount.com
x_google_function_memory_mb: 128
x_google_function_name: env
x_google_function_region: us-central1
x_google_function_timeout_sec: 30
x_google_function_trigger_type: HTTP_TRIGGER
x_google_function_version: 6
x_google_gcloud_project: hightowerlabs
x_google_gcp_project: hightowerlabs
x_google_load_on_start: false
x_google_supervisor_hostname: 169.254.8.129
x_google_supervisor_internal_port: 8081
x_google_worker_port: 8091
x-cloud-trace-context: 55ddf3b7b5259783faa8113c8823d707;o=1
date: Thu, 14 Mar 2019 14:30:51 GMT
server: Google Frontend
content-length: 79
alt-svc: quic=":443"; ma=2592000; v="46,44,43,39"

{
  "database": {
    "username": "user",
    "password": "123456789"
  }
}
```
