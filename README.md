# Vault Mutating Admission Webhook (Go)

This project contains two components:

- A mutating admission webhook that injects a vault-env-runner into annotated Pods.
- A small `vault-env-runner` binary that authenticates to Vault, fetches KV v2 secrets, sets environment variables and execs the original command.

See `k8s/` for Kubernetes manifests and `docs/certs.md` for TLS instructions.

Build:

```bash
go build ./cmd/webhook
go build ./cmd/runner
# Replace <OWNER> with your GitHub org/user
docker build -f Dockerfile.webhook -t ghcr.io/<OWNER>/vault-webhook:webhook .
docker build -f Dockerfile.runner -t ghcr.io/<OWNER>/vault-webhook:runner .
```

Security:

- TLS is required for the webhook. Use cert-manager or the OpenSSL guidance in `docs/certs.md`.
- The runner uses Kubernetes auth and will exchange the service account JWT for a Vault token. TLS verification is enabled by default. Use `VAULT_INSECURE_SKIP_VERIFY=true` only for local development.
## Install as a CRD-managed operator

Apply the CRD and controller to let cluster admins configure the webhook via `VaultInjector` resources:

```bash
kubectl apply -f k8s/crd.yaml
kubectl apply -f k8s/controller-deployment.yaml
```

Create a `VaultInjector` resource to point the controller at the webhook service and CA secret. Example:

```yaml
apiVersion: vault.example.com/v1alpha1
kind: VaultInjector
metadata:
	name: default-injector
spec:
	serviceName: vault-webhook
	serviceNamespace: default
	caSecret: vault-webhook-tls
```

This controller will ensure a `MutatingWebhookConfiguration` is created/updated using the CA in the provided secret.

## Install via Helm Chart

### Prerequisites

Before installing the Helm chart, ensure you have:

- Kubernetes cluster (v1.20+)
- Helm 3.x installed
- `kubectl` configured to access your cluster
- A running Vault instance with KV v2 secrets engine enabled
- Kubernetes authentication method enabled in Vault
- cert-manager or manually provisioned TLS certificates

### Add Helm Repository

```bash
# If using a Helm repo (adjust URL as needed)
helm repo add vault-injector https://your-repo-url
helm repo update
```

### Install the Helm Chart

**Basic installation** (default namespace):

```bash
helm install vault-injector ./helm/vault-injector \
  --namespace default
```

**Production installation** (custom namespace with image overrides):

```bash
# Create namespace
kubectl create namespace vault-system

# Install with custom values
helm install vault-injector ./helm/vault-injector \
  --namespace vault-system \
  --set image.controller.repository=ghcr.io/your-org/vault-controller \
  --set image.webhook.repository=ghcr.io/your-org/vault-webhook \
  --set image.runner.repository=ghcr.io/your-org/vault-runner \
  --set webhook.tlsSecretName=vault-webhook-tls
```

### Configure TLS Certificates

The webhook requires valid TLS certificates. Two options:

**Option 1: Using cert-manager (recommended)**

```yaml
# Add to your Helm values.yaml
certManager:
  enabled: true
  issuerRef:
    name: selfsigned-issuer
    kind: Issuer
```

**Option 2: Manual certificate creation**

See `docs/certs.md` for OpenSSL instructions, then create a secret:

```bash
kubectl create secret tls vault-webhook-tls \
  --cert=path/to/cert.pem \
  --key=path/to/key.pem \
  --namespace default
```

### Create VaultInjector Resource

After installation, create a `VaultInjector` resource to configure the webhook:

```yaml
apiVersion: vault.example.com/v1alpha1
kind: VaultInjector
metadata:
  name: vault-config
  namespace: default
spec:
  serviceName: vault-webhook
  serviceNamespace: default
  caSecret: vault-webhook-tls  # Must contain the TLS CA certificate
```

Apply it:

```bash
kubectl apply -f - <<EOF
apiVersion: vault.example.com/v1alpha1
kind: VaultInjector
metadata:
  name: vault-config
  namespace: default
spec:
  serviceName: vault-webhook
  serviceNamespace: default
  caSecret: vault-webhook-tls
EOF
```

## Usage Guide for End Users

### Prerequisites

- Service account with Kubernetes authentication enabled in Vault
- Service account token available as a volume mount (automatic in Kubernetes)
- Read permissions in Vault for the KV v2 secrets you need to inject

### Annotate Your Pods

Add the `vault.example.com/inject: "true"` annotation to any Pod to enable secret injection:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    vault.example.com/inject: "true"
    vault.example.com/vault-addr: "https://vault.example.com:8200"
    vault.example.com/vault-path: "kv/data/myapp/secrets"
    vault.example.com/vault-secret-keys: "DATABASE_PASSWORD,API_KEY"
spec:
  serviceAccountName: my-app-sa
  containers:
  - name: app
    image: myapp:latest
    env:
    - name: DATABASE_HOST
      value: "postgres.default.svc.cluster.local"
```

### Complete Example Deployment

```yaml
---
# Service Account for the application
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-app-sa
  namespace: default

---
# Deployment with secret injection
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
      annotations:
        vault.example.com/inject: "true"
        vault.example.com/vault-addr: "https://vault.example.com:8200"
        vault.example.com/vault-path: "kv/data/myapp/config"
        vault.example.com/vault-secret-keys: "DB_PASSWORD,API_TOKEN"
    spec:
      serviceAccountName: my-app-sa
      containers:
      - name: app
        image: myapp:latest
        ports:
        - containerPort: 8080
        env:
        - name: APP_ENV
          value: "production"
        # Injected secrets will be available as environment variables:
        # DB_PASSWORD, API_TOKEN
```

### Annotation Reference

| Annotation | Required | Example | Description |
|-----------|----------|---------|-------------|
| `vault.example.com/inject` | Yes | `"true"` | Enable secret injection for this Pod |
| `vault.example.com/vault-addr` | Yes | `"https://vault.example.com:8200"` | Vault server address |
| `vault.example.com/vault-path` | Yes | `"kv/data/myapp/secrets"` | Path to KV v2 secret in Vault |
| `vault.example.com/vault-secret-keys` | Yes | `"DB_PASS,API_KEY"` | Comma-separated list of secret keys to inject |
| `vault.example.com/vault-role` | No | `"my-app-role"` | Kubernetes auth role (defaults to service account name) |
| `vault.example.com/vault-insecure` | No | `"false"` | Set to `"true"` only for local dev (disables TLS verification) |

### Vault Setup

Ensure Vault is configured for Kubernetes authentication:

```bash
# Enable Kubernetes auth method
vault auth enable kubernetes

# Configure with cluster details
vault write auth/kubernetes/config \
  token_reviewer_jwt=@/var/run/secrets/kubernetes.io/serviceaccount/token \
  kubernetes_host=https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT \
  kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt

# Create a policy for your application
vault policy write myapp-policy - <<EOF
path "kv/data/myapp/*" {
  capabilities = ["read"]
}
EOF

# Create a Kubernetes auth role
vault write auth/kubernetes/role/my-app-sa \
  bound_service_account_names=my-app-sa \
  bound_service_account_namespaces=default \
  policies=myapp-policy \
  ttl=24h
```

### Verify Injection

Check if the sidecar was successfully injected:

```bash
# View the pod
kubectl get pod my-app -o jsonpath='{.spec.containers[*].name}'
# Should output: vault-env-runner app

# Check the sidecar logs
kubectl logs my-app -c vault-env-runner

# Verify environment variables in the app container
kubectl exec my-app -c app -- env | grep -E "DB_PASSWORD|API_KEY"
```

### Troubleshooting

**Webhook not injecting sidecars:**
- Verify the annotation `vault.example.com/inject: "true"` is present
- Check webhook pod logs: `kubectl logs -l app=vault-webhook`
- Ensure `MutatingWebhookConfiguration` is created: `kubectl get mutatingwebhookconfigurations`

**Runner sidecar failing to authenticate:**
- Check runner logs: `kubectl logs <pod> -c vault-env-runner`
- Verify Vault Kubernetes auth is configured correctly
- Ensure service account has Vault role binding

**TLS certificate errors:**
- Verify TLS certificate is valid: `kubectl get secret vault-webhook-tls -o yaml`
- Check webhook service endpoint: `kubectl get service vault-webhook`
- Review cert-manager status if using auto-provisioning: `kubectl describe certificate`

## Uninstall

```bash
# Remove Helm release
helm uninstall vault-injector --namespace default

# Remove CRD (optional, preserves existing resources)
kubectl delete crd vaultinjectors.vault.example.com
```

# container-secret-injection