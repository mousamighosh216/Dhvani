# `make test` runs everything that works with no network access (nodeps build).
# After `go mod tidy` on a normal machine, plain `go test ./...` also covers
# go-mp3 and pgx.

.PHONY: test test-backend test-frontend test-integration samples run-nodeps build

test: test-backend test-frontend

test-backend:
	cd backend && go vet -tags nodeps ./... && go test -tags nodeps ./...

test-frontend:
	cd frontend && npm test

# Needs a migrated Postgres (docker compose up -d), then:
#   DHWANI_TEST_DB=postgres://postgres:postgres@localhost:5432/dhwani?sslmode=disable make test-integration
test-integration:
	cd backend && go test -tags integration ./metadata/

samples:
	cd backend && go run ./cmd/gensamples -out ../samples

build:
	cd frontend && npm install && npm run build
	cd backend && go build -tags nodeps -o ../dhwani-server .

# Run without PostgreSQL (in-memory, data lost on exit).
run-nodeps: build
	cd backend && go run -tags nodeps . -static ../frontend/dist
