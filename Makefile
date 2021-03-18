.PHONY: build help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

#
# Tests
#

test-short: ## Run tests with -short flag
	go test -timeout 30s -short -race ./...

test: ## Run tests
	go test -timeout 1m -race ./...

test-short-extra: ## Run tests with -short flag
	CGO_ENABLED=1 go test -tags cgo_sqlite -timeout 30s -short -race ./...

compile-tests: ## Compile test and benchmarks
	for pkg in $$(go list ./...) ; do \
		go test -c -bench . $$pkg ; \
	done

#
# Build
#

build: ## Build an example baker binary
	go build -v -o baker-bin-example ./examples/advanced/

build-extra: ## Build an example baker binary with the "extra" components (sqlite3 output atm)
	CGO_ENABLED=1 go build -v -tags cgo_sqlite -o baker-bin-example ./examples/advanced/

#
# Coverage reports
#

--ci-cover:
	@go test -coverprofile coverage.txt -covermode atomic ./...

cover: --ci-cover ## Create & open the unit-test coverage report
	@go tool cover -html coverage.txt

#
# Format / Lint / Static checks
#

gofmt: ## Run gofmt locally without overwriting any file
	gofmt -d -s $$(find . -name '*.go' | grep -v vendor)

gofmt-write: ## Run gofmt locally overwriting files
	gofmt -w -s $$(find . -name '*.go' | grep -v vendor)

govet: ## Run go vet on the project
	go vet ./...
