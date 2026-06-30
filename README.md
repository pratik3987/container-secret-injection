# Vault Mutating Admission Webhook (Go)

This project contains two components:

- A mutating admission webhook that injects a vault-env-runner into annotated Pods.
- A small `vault-env-runner` binary that authenticates to Vault, fetches KV v2 secrets, sets environment variables and execs the original command.

See `k8s/` for Kubernetes manifests and `docs/certs.md` for TLS instructions.

Build:

```bash
go build ./cmd/webhook
go build ./cmd/runner
docker build -f Dockerfile.webhook -t example/vault-webhook:webhook .
docker build -f Dockerfile.runner -t example/vault-webhook:runner .
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

# container-secret-injection