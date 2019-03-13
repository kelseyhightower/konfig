# konfig

## Usage


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

```
kubectl create secret generic env \
  --from-literal foo=bar \
  --from-file config.json
```

```
gcloud alpha run deploy env \
  --set-env-vars 'GOOGLE_CLOUD_REGION=us-central1,FOO=$SecretKeyRef:/projects/hightowerlabs/locations/us-central1/clusters/api/namespaces/default/secrets/env/keys/foo,CONFIG_FILE=$SecretKeyRef:/projects/hightowerlabs/locations/us-central1/clusters/api/namespaces/default/secrets/env/keys/config.json?tempFile=true' \
  --concurrency 50 \
  --image gcr.io/hightowerlabs/env:0.0.1 \
  --memory 2G \
  --region us-central1
```

```
gcloud alpha run services add-iam-policy-binding env \
  --member="allUsers" \
  --role="roles/run.invoker"
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
