# TLS for webhook

Option A: cert-manager (recommended for production)
- Install cert-manager and configure ClusterIssuer
- Create Certificate referencing the Service `vault-webhook` and mount the secret into the Deployment at `/tls`

Option B: OpenSSL (local/testing)

Generate CA and server cert:

```bash
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -subj "/CN=example-ca" -days 3650 -out ca.crt

openssl genrsa -out server.key 2048
openssl req -new -key server.key -subj "/CN=vault-webhook.default.svc" -out server.csr
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 365

kubectl create secret tls vault-webhook-tls --cert=server.crt --key=server.key
```

Then replace `caBundle` in `k8s/mutatingwebhook.yaml` with the base64 of `ca.crt`.
