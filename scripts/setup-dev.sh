#!/usr/bin/env bash
set -euo pipefail

echo "==> Hotel Key PMS — dev setup"

# Check prerequisites
for cmd in docker docker-compose go node npm terraform kubectl; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "ERROR: $cmd not found in PATH" >&2
    exit 1
  fi
done

echo "==> Installing Go module dependencies"
for svc in pms-core reservation room-management billing housekeeping maintenance notification channel-manager wechat-connector; do
  echo "  -> $svc"
  (cd "services/$svc" && go mod tidy)
done

echo "==> Installing frontend dependencies"
(cd frontend/react-dashboard && npm install)
(cd frontend/staff-mobile-pwa && npm install)

echo "==> Starting local infrastructure (postgres, redis, kafka, keycloak)"
docker-compose up -d postgres redis zookeeper kafka keycloak

echo "==> Waiting for postgres to be ready..."
until docker-compose exec -T postgres pg_isready -U hotelkey_admin -d hotelkey; do
  sleep 2
done

echo "==> Running migrations"
docker-compose exec -T postgres psql -U hotelkey_admin -d hotelkey \
  -f /migrations/001_initial_schema.sql

echo "==> Seeding test data"
docker-compose exec -T postgres psql -U hotelkey_admin -d hotelkey \
  -f /scripts/seed-test-data.sql

echo ""
echo "Dev environment ready."
echo "  Dashboard: http://localhost:5173"
echo "  Keycloak:  http://localhost:8443"
echo "  PMS Core:  http://localhost:8081"
