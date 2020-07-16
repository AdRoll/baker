.PHONY: build help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

test-short: ## Run tests with -short flag
	go test -short -race ./...

test: ## Run tests
	go test -race ./...

cover: ## Run tests and open coverage report in browser
	go test -cover -coverprofile cover.out ./...
	go tool cover -html cover.out

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

build: ## Build an example baker binary
	go build -v -o baker-bin-example ./examples/advanced/
