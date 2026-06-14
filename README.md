# Hotel Key PMS

Cloud-native Property Management System for hotels in Japan and China markets.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Clients: React Dashboard (web) · Staff Mobile PWA · WeChat MP  │
└───────────────────────────────┬─────────────────────────────────┘
                                │ HTTPS
                    ┌───────────▼───────────┐
                    │  Istio Gateway (Kong)  │
                    │  mTLS service mesh     │
                    └───────────┬───────────┘
                                │
          ┌─────────────────────┼─────────────────────┐
          │                     │                     │
    ┌─────▼──────┐   ┌──────────▼─────┐   ┌──────────▼────────┐
    │  pms-core  │   │  reservation   │   │  room-management  │
    │  /v1/hotels│   │/v1/reservations│   │     /v1/rooms     │
    └─────┬──────┘   └──────────┬─────┘   └──────────┬────────┘
          │                     │                     │
    ┌─────▼──────┐   ┌──────────▼─────┐   ┌──────────▼────────┐
    │  billing   │   │ housekeeping   │   │   maintenance     │
    └────────────┘   └────────────────┘   └───────────────────┘
          │
    ┌─────▼──────────────────────────────────────┐
    │  notification · channel-manager · wechat   │
    └────────────────────────────────────────────┘
          │
    ┌─────▼───────────────────────────────────────┐
    │  PostgreSQL 15.5 (RLS) · Redis 7.1 · Kafka  │
    └─────────────────────────────────────────────┘
```

## Key Design Decisions

| Concern | Approach |
|---------|----------|
| Multi-tenancy | PostgreSQL Row Level Security (`app.current_tenant_id`) |
| Double-booking | Redis `SETNX` lock per room-type+date range |
| Billing integrity | Append-only folio; DB triggers reject UPDATE/DELETE |
| Auth | Keycloak 24 with custom TOTP theme |
| Service mesh | Istio with mTLS STRICT mode |
| IaC | Terraform modules: VPC, EKS, RDS, ElastiCache |
| GitOps | ArgoCD syncs `k8s/overlays/dev` on push to `develop` |
| Per-person pricing | Hook in `room_types.per_person_pricing_enabled` (Japan Ryokan) |

## Quick Start (local)

```bash
# Prerequisites: Docker, Go 1.22+, Node 20+
make setup       # installs deps, starts infra, runs migrations, seeds data
make docker-up   # start all services
```

Services start on:

| Service | Port |
|---------|------|
| pms-core | 8081 |
| reservation | 8082 |
| room-management | 8083 |
| billing | 8084 |
| housekeeping | 8085 |
| maintenance | 8086 |
| notification | 8087 |
| channel-manager | 8088 |
| wechat-connector | 8089 |
| Keycloak | 8443 |
| React Dashboard | 5173 |

## Deploy to AWS (dev)

```bash
# 1. Bootstrap Terraform state bucket (one-time)
aws s3 mb s3://hotel-key-terraform-state --region ap-northeast-1
aws dynamodb create-table \
  --table-name hotel-key-terraform-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region ap-northeast-1

# 2. Apply infrastructure
make terraform-apply

# 3. Bootstrap ArgoCD
REPO_URL=https://github.com/your-org/hotel-key-pms \
ARGOCD_SERVER=argocd.your-domain.internal \
ARGOCD_AUTH_TOKEN=your-token \
make argocd-setup
```

Push to `develop` branch to trigger automated deployment via GitHub Actions.

## Environment Variables

All services read configuration from environment variables. See `k8s/overlays/dev/secrets.env` for the full list. In production, values are injected from AWS Secrets Manager via the External Secrets Operator.

## Security

- All inter-service traffic encrypted by Istio mTLS
- RLS ensures tenants can never read each other's data
- Folio entries are immutable (DB trigger enforced)
- No secrets in code or git history
- Trivy + gosec + Gitleaks run in CI on every push
