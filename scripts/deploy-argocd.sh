#!/usr/bin/env bash
# Bootstrap ArgoCD and register the Hotel Key application.
# Run once per cluster. Assumes kubectl is configured for the target cluster.
set -euo pipefail

ARGOCD_VERSION="${ARGOCD_VERSION:-v2.11.3}"
NAMESPACE="argocd"
APP_NAMESPACE="hotel-key"
REPO_URL="${REPO_URL:?Set REPO_URL to the git remote URL}"
ARGOCD_SERVER="${ARGOCD_SERVER:?Set ARGOCD_SERVER}"
ARGOCD_AUTH_TOKEN="${ARGOCD_AUTH_TOKEN:?Set ARGOCD_AUTH_TOKEN}"

echo "==> Installing ArgoCD ${ARGOCD_VERSION}"
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -n "$NAMESPACE" \
  -f "https://raw.githubusercontent.com/argoproj/argo-cd/${ARGOCD_VERSION}/manifests/install.yaml"

echo "==> Waiting for ArgoCD server to be ready"
kubectl rollout status deploy/argocd-server -n "$NAMESPACE" --timeout=120s

echo "==> Logging in"
argocd login "$ARGOCD_SERVER" \
  --auth-token "$ARGOCD_AUTH_TOKEN" \
  --grpc-web \
  --insecure

echo "==> Registering hotel-key-dev application"
argocd app create hotel-key-dev \
  --repo "$REPO_URL" \
  --path "k8s/overlays/dev" \
  --dest-server "https://kubernetes.default.svc" \
  --dest-namespace "$APP_NAMESPACE" \
  --sync-policy automated \
  --auto-prune \
  --self-heal \
  --revision HEAD \
  --upsert

echo "==> Triggering initial sync"
argocd app sync hotel-key-dev --grpc-web

echo ""
echo "ArgoCD application registered. Access the UI at https://${ARGOCD_SERVER}"
