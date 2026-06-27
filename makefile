build-go:
	go build -o aegis cmd/aegis/main.go

build-web:
	cd web && VITE_API_URL= VITE_API_BASE= pnpm build

build: build-web build-go

build-all: build

dev-web:
	cd web && pnpm dev


dev:
	go run cmd/aegis/main.go

run:
	./aegis

test:
	go test ./...

test-keypair:
	go run scripts/generate-test-keypair.go

test-integration:
	go test -tags=integration -v ./test/...

test-integration-developer:
	go test -tags=integration -v ./test/developer/...

test-integration-dashboard:
	go test -tags=integration -v ./test/dashboard/...

test-integration-ws:
	go test -tags=integration -v ./test/ws/...

test-integration-lifecycle:
	AEGIS_RUN_BOUNTY_LOG=1 go test -tags=integration -v -timeout=45m ./test/lifecycle/...

test-race:
	go test -race ./...

sqlc:
	sqlc generate

migrate-up:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/storage/migrations postgres "$${DATABASE_URL}" up

migrate-down:
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/storage/migrations postgres "$${DATABASE_URL}" down

docker-up:
	docker compose -f deployments/docker-compose.yaml up -d

docker-down:
	docker compose -f deployments/docker-compose.yaml down

demo-normal:
	./scripts/demo-normal.sh

demo-expired:
	./scripts/demo-expired-blockhash.sh

verify-lifecycle:
	./scripts/verify-lifecycle-log.sh

clean:
	rm -f aegis

.PHONY: build build-go build-all dev run test test-keypair test-integration test-integration-developer test-integration-dashboard test-integration-ws test-integration-lifecycle test-race sqlc migrate-up migrate-down docker-up docker-down demo-normal demo-expired verify-lifecycle clean air dev-web build-web

air:
	air --build.cmd "go build -o tmp/main cmd/aegis/main.go" --build.entrypoint "./tmp/main" --build.exclude_dir "tmp,build"
