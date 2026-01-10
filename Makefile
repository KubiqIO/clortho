.PHONY: all build test clean check-vuln test-e2e

all: build

build:
	go build -v -o clortho-server ./cmd/server/main.go

check-vuln:
	govulncheck ./cmd/... ./internal/...

test:
	go test -v $$(go list ./... | grep -v /scripts)

test-e2e:
	go test -v ./internal/api -run TestLicenseLifecycle

test-all:
	$(MAKE) test
	$(MAKE) test-e2e
	$(MAKE) check-vuln

clean:
	rm -f clortho-server

migrate-up:
	go run scripts/migrate.go -direction up

migrate-down:
	go run scripts/migrate.go -direction down

db-reset:
	go run scripts/migrate.go -direction drop
	go run scripts/migrate.go -direction up
