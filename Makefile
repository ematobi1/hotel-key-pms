.PHONY: setup build test docker-up docker-down terraform-apply k8s-deploy lint

SERVICES := pms-core reservation room-management billing housekeeping maintenance notification channel-manager wechat-connector
GO_SERVICES_DIR := services

# ─── Dev setup ────────────────────────────────────────────────────────────────

setup:
	@echo "==> Setting up development environment"
	@bash scripts/setup-dev.sh

# ─── Build ────────────────────────────────────────────────────────────────────

build:
	@echo "==> Building all services"
	@for svc in $(SERVICES); do \
		echo "  -> Building $$svc"; \
		(cd $(GO_SERVICES_DIR)/$$svc && go build ./...) || exit 1; \
	done
	@echo "  -> Building react-dashboard"
	@(cd frontend/react-dashboard && npm run build)
	@echo "  -> Building staff-mobile-pwa"
	@(cd frontend/staff-mobile-pwa && npm run build)

# ─── Test ─────────────────────────────────────────────────────────────────────

test:
	@echo "==> Running Go tests"
	@for svc in $(SERVICES); do \
		echo "  -> Testing $$svc"; \
		(cd $(GO_SERVICES_DIR)/$$svc && go test -race -count=1 ./...) || exit 1; \
	done
	@echo "==> Running frontend tests"
	@(cd frontend/react-dashboard && npm test)

lint:
	@echo "==> Linting Go services"
	@for svc in $(SERVICES); do \
		echo "  -> Linting $$svc"; \
		(cd $(GO_SERVICES_DIR)/$$svc && go vet ./...) || exit 1; \
	done
	@echo "==> Linting frontend"
	@(cd frontend/react-dashboard && npm run lint)

# ─── Docker ───────────────────────────────────────────────────────────────────

docker-up:
	@echo "==> Starting local stack"
	docker-compose up -d

docker-down:
	@echo "==> Stopping local stack"
	docker-compose down

docker-logs:
	docker-compose logs -f --tail=100

docker-build:
	@echo "==> Building all Docker images"
	docker-compose build

# ─── Terraform ────────────────────────────────────────────────────────────────

terraform-init:
	@echo "==> Terraform init"
	(cd terraform && terraform init)

terraform-plan:
	@echo "==> Terraform plan"
	(cd terraform && terraform plan -var="environment=dev")

terraform-apply:
	@echo "==> Terraform apply (dev)"
	(cd terraform && terraform apply -var="environment=dev" -auto-approve)

terraform-destroy:
	@echo "==> Terraform destroy (dev) — DESTRUCTIVE"
	@read -p "Type 'yes' to confirm: " confirm && [ "$$confirm" = "yes" ]
	(cd terraform && terraform destroy -var="environment=dev" -auto-approve)

# ─── Kubernetes ───────────────────────────────────────────────────────────────

k8s-deploy:
	@echo "==> Applying k8s manifests (dev overlay)"
	kubectl apply -k k8s/overlays/dev

k8s-diff:
	kubectl diff -k k8s/overlays/dev

k8s-status:
	kubectl get pods -n hotel-key -o wide

argocd-setup:
	@echo "==> Bootstrapping ArgoCD"
	bash scripts/deploy-argocd.sh

# ─── Database ─────────────────────────────────────────────────────────────────

db-migrate:
	@echo "==> Running migrations"
	docker-compose exec postgres psql -U hotelkey_admin -d hotelkey \
	  -f /docker-entrypoint-initdb.d/001_initial_schema.sql

db-seed:
	@echo "==> Seeding test data"
	docker-compose exec postgres psql -U hotelkey_admin -d hotelkey \
	  -f /scripts/seed-test-data.sql

db-shell:
	docker-compose exec postgres psql -U hotelkey_admin -d hotelkey

# ─── Help ─────────────────────────────────────────────────────────────────────

help:
	@echo "Hotel Key PMS — available targets:"
	@echo "  setup             Install dependencies and start dev env"
	@echo "  build             Build all Go services and frontends"
	@echo "  test              Run all tests"
	@echo "  lint              Lint Go and frontend code"
	@echo "  docker-up         Start local Docker stack"
	@echo "  docker-down       Stop local Docker stack"
	@echo "  terraform-apply   Apply Terraform to dev environment"
	@echo "  k8s-deploy        Apply Kubernetes manifests"
	@echo "  argocd-setup      Bootstrap ArgoCD on cluster"
	@echo "  db-migrate        Run SQL migrations"
	@echo "  db-seed           Seed test data"
	@echo "  db-shell          Open psql shell"
