PLUGIN_NAME ?= vault-plugin-secrets-nexus-repository

.DEFAULT_GOAL := all

all: fmt check build test

## Format
fmt: gofmt gofumpt goimports tidy

# Exclude auto-generated code to be formatted by gofmt, gofumpt & goimports.
FIND=find . \( -path "./examples" -path "./scripts" -o -path "./test" \) -prune -false -o -name '*.go'

gofmt:
	$(FIND) -exec gofmt -l -w {} \;

gofumpt:
	$(FIND) -exec gofumpt -w {} \;

goimports:
	$(FIND) -exec goimports -w {} \;

tidy:
	go mod tidy

## Check
check: staticcheck lint local-lint

staticcheck:
	staticcheck ./...

lint:
	golint ./...

local-lint:
	docker run --rm -v $(shell pwd):/$(PLUGIN_NAME) -w /$(PLUGIN_NAME)/. \
	  golangci/golangci-lint golangci-lint run --sort-results -v

## Build
build:
	mkdir -p dist/bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o dist/bin/$(PLUGIN_NAME) ./src/cmd/$(PLUGIN_NAME)/main.go

## Test
test:
	gotest -v ./src/...

test-coverage:
	go clean -testcache &&\
		gotest -coverprofile=c.out -v -tags=test ./src/...

test-acceptance:
	VAULT_PLUGIN_DIR="./dist/bin" bats test/acceptance-tests.bats

.PHONY: fmt gofmt gofumpt goimports tidy check statccheck lint local-lint build test test-coverage
