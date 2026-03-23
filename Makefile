.DEFAULT_GOAL := help
.PHONY: help build build-full build-frontend test fmt lint generate clean install dev dev-backend dev-frontend pglint pglint-all pglint-json run-chinook run-pagila run-adventureworks run-airlines run-northwind run-re run-re-full demo

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

BINARY := pgdesigner
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)
RPC_PORT := 9990

build: ## Build binary
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/pgdesigner/

build-full: build-frontend build ## Build with embedded frontend

build-frontend: ## Build Vue frontend (pnpm build)
	cd frontend && pnpm build

test: ## Run all tests
	go clean -testcache
	go test ./...

fmt: ## Format code (golangci-lint fmt)
	@golangci-lint fmt

lint: ## Lint code (golangci-lint run)
	@golangci-lint version
	@golangci-lint config verify
	@golangci-lint run

generate: ## Generate zenrpc code
	go generate ./pkg/rpc/

ts-client: build ## Generate TypeScript RPC client
	./$(BINARY) -ts_client > frontend/src/api/factory.generated.ts
	@sed -i '' 's/Ipgd\./I/g' frontend/src/api/factory.generated.ts
	@sed -i '' '2s/$$/\'$$'\n\/\/ @ts-nocheck/' frontend/src/api/factory.generated.ts
	@echo "Generated frontend/src/api/factory.generated.ts"

clean: ## Remove built binary
	rm -f $(BINARY)

install: ## Install binary to GOPATH/bin
	CGO_ENABLED=0 go install -ldflags="$(LDFLAGS)" ./cmd/pgdesigner/

dev: dev-backend ## Run dev server (backend)

dev-backend: build ## Run Go backend on :$(RPC_PORT)
	./$(BINARY) --port $(RPC_PORT) pkg/pgd/testdata/pagila.pgd

dev-frontend: ## Run Vite dev server (frontend)
	cd frontend && pnpm dev

pglint: build ## Validate pagila.pgd schema (warnings+)
	./$(BINARY) lint -s warning pkg/pgd/testdata/pagila.pgd

pglint-all: build ## Validate all .pgd test files
	@for f in pkg/pgd/testdata/*.pgd; do \
		echo "=== $$f ==="; \
		./$(BINARY) lint -s warning "$$f"; \
		echo; \
	done

pglint-json: build ## Validate pagila.pgd as JSON
	./$(BINARY) lint -f json pkg/pgd/testdata/pagila.pgd

demo: build-full ## Prepare demo/ directory with binary and sample schemas
	@mkdir -p demo/schemas/{pgd,sql,pdd} demo/diff/{add-table,add-column,move-column,modify-index}
	cp $(BINARY) demo/$(BINARY)
	@# PGD schemas (native format)
	cp pkg/pgd/testdata/chinook.pgd demo/schemas/pgd/
	cp pkg/pgd/testdata/northwind.pgd demo/schemas/pgd/
	cp pkg/pgd/testdata/pagila.pgd demo/schemas/pgd/
	cp pkg/pgd/testdata/airlines.pgd demo/schemas/pgd/
	cp pkg/pgd/testdata/adventureworks.pgd demo/schemas/pgd/
	@# SQL (pg_dump)
	cp pkg/format/sql/testdata/chinook.sql demo/schemas/sql/
	cp pkg/format/sql/testdata/northwind.sql demo/schemas/sql/
	cp pkg/format/sql/testdata/pagila.sql demo/schemas/sql/
	cp pkg/format/sql/testdata/airlines.sql demo/schemas/sql/
	cp pkg/format/sql/testdata/adventureworks.sql demo/schemas/sql/
	@# PDD (MicroOLAP sample databases)
	cp pkg/format/pdd/testdata/Chinook.pdd demo/schemas/pdd/
	cp pkg/format/pdd/testdata/AdventureWorks.pdd demo/schemas/pdd/
	cp pkg/format/pdd/testdata/pagila-light.pdd demo/schemas/pdd/
	@# Diff examples
	cp pkg/designer/diff/testdata/diff/plf-751-add-table/old.pgd demo/diff/add-table/old.pgd
	cp pkg/designer/diff/testdata/diff/plf-751-add-table/new.pgd demo/diff/add-table/new.pgd
	cp pkg/designer/diff/testdata/diff/plf-885-add-column/old.pgd demo/diff/add-column/old.pgd
	cp pkg/designer/diff/testdata/diff/plf-885-add-column/new.pgd demo/diff/add-column/new.pgd
	cp pkg/designer/diff/testdata/diff/plf-801-move-column/old.pgd demo/diff/move-column/old.pgd
	cp pkg/designer/diff/testdata/diff/plf-801-move-column/new.pgd demo/diff/move-column/new.pgd
	cp pkg/designer/diff/testdata/diff/plf-890-modify-index/old.pgd demo/diff/modify-index/old.pgd
	cp pkg/designer/diff/testdata/diff/plf-890-modify-index/new.pgd demo/diff/modify-index/new.pgd
	@echo "Demo ready! cd demo && make help"

run-chinook: build ## Open ERD: Chinook (11 tables)
	./$(BINARY) pkg/pgd/testdata/chinook.pgd

run-pagila: build ## Open ERD: Pagila (15 tables, partitions, triggers)
	./$(BINARY) pkg/pgd/testdata/pagila.pgd

run-adventureworks: build ## Open ERD: AdventureWorks (68 tables, 5 schemas)
	./$(BINARY) pkg/pgd/testdata/adventureworks.pgd

run-airlines: build ## Open ERD: Airlines (8 tables, bookings schema)
	./$(BINARY) pkg/pgd/testdata/airlines.pgd

run-northwind: build ## Open ERD: Northwind (14 tables)
	./$(BINARY) pkg/pgd/testdata/northwind.pgd

# reverse engineering from live PostgreSQL
run-re: build ## Open ERD from live PG (requires PGD_DSN)
	./$(BINARY) --schema public "$(PGD_DSN)"

run-re-full: build ## Open ERD from live PG with full introspection
	./$(BINARY) --schema public --full "$(PGD_DSN)"
