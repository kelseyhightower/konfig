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
