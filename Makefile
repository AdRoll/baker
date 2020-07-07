# This Makefile can be used to obtain a Linux binary through docker (eg: from OSX)
# For normal development, Docker is not required. See README.md for build instructions.
.PHONY: build help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

test-short: ## Run tests with -short flag in the local env
	go test -short -race ./...

test: ## Run tests in the local env
	go test -race ./...

compile-tests: ## Compile test and benchmarks
	for pkg in $$(go list ./...) ; do \
		go test -c -bench . $$pkg ; \
	done

gofmt: ## Run gofmt locally without overwriting any file
	gofmt -d -s $$(find . -name '*.go' | grep -v vendor)

gofmt-write: ## Run gofmt locally overwriting files
	gofmt -w -s $$(find . -name '*.go' | grep -v vendor)

govet: ## Run go vet on the project
	go vet ./...
