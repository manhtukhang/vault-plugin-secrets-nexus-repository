.DEFAULT_GOAL := all

all: fmt check test

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
	docker run --rm -v $(shell pwd):/$(APPNAME) -w /$(APPNAME)/. \
	  golangci/golangci-lint golangci-lint run --sort-results -v

## Test
test:
	gotest -v ./src/...

test-coverage:
	go clean -testcache &&\
		gotest -coverprofile=c.out -v ./src/...


.PHONY: fmt gofmt gofumpt goimports tidy check statccheck lint local-lint test test-coverage
