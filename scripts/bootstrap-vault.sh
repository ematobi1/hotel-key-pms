#!/usr/bin/env bash
# Bootstrap Vault secrets for Hotel Key PMS.
# Run after Vault is initialized and unsealed.
# Requires: vault CLI, AWS credentials, kubectl access to the cluster.
set -euo pipefail

VAULT_ADDR="${VAULT_ADDR:?Set VAULT_ADDR}"
VAULT_TOKEN="${VAULT_TOKEN:?Set VAULT_TOKEN}"
export VAULT_ADDR VAULT_TOKEN

echo "==> Enabling KV v2 secrets engine"
vault secrets enable -path=secret kv-v2 2>/dev/null || echo "  (already enabled)"

echo "==> Writing secrets (values sourced from environment — never hard-coded)"
vault kv put secret/hotelkey/postgres \
  password="${POSTGRES_PASSWORD:?}"

vault kv put secret/hotelkey/redis \
  password="${REDIS_PASSWORD:?}"

vault kv put secret/hotelkey/jwt \
  secret="${JWT_SECRET:?}"

vault kv put secret/hotelkey/stripe \
  api_key="${STRIPE_API_KEY:?}"

vault kv put secret/hotelkey/wechat \
  app_secret="${WECHAT_APP_SECRET:?}"

vault kv put secret/hotelkey/keycloak \
  client_secret="${KEYCLOAK_CLIENT_SECRET:?}"

echo "==> Creating Vault policy for hotel-key-app"
vault policy write hotel-key-app - <<'POLICY'
path "secret/data/hotelkey/*" {
  capabilities = ["read"]
}
path "secret/metadata/hotelkey/*" {
  capabilities = ["list"]
}
POLICY

echo "==> Enabling Kubernetes auth method"
vault auth enable kubernetes 2>/dev/null || echo "  (already enabled)"

KUBE_HOST=$(kubectl config view --raw -o jsonpath='{.clusters[0].cluster.server}')
KUBE_CA=$(kubectl config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' | base64 -d)
SA_JWT=$(kubectl create token vault-auth -n hotel-key --duration=8760h)

vault write auth/kubernetes/config \
  kubernetes_host="$KUBE_HOST" \
  kubernetes_ca_cert="$KUBE_CA" \
  token_reviewer_jwt="$SA_JWT"

echo "==> Binding hotel-key service accounts to policy"
vault write auth/kubernetes/role/hotel-key-app \
  bound_service_account_names="pms-core,reservation,room-management,billing,housekeeping,maintenance,notification,channel-manager,wechat-connector" \
  bound_service_account_namespaces="hotel-key" \
  policies="hotel-key-app" \
  ttl="1h"

echo "==> Enabling Vault audit log → file"
vault audit enable file file_path=/vault/logs/audit.log 2>/dev/null || echo "  (already enabled)"

echo "==> Vault bootstrap complete."
echo "    Secrets are now accessible to hotel-key service accounts via Vault CSI driver."
echo "    Rotate credentials with: vault kv put secret/hotelkey/<name> <key>=<new_value>"
