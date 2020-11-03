.PHONY: help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

setup: ## Setup the website for local development or to add content
	git submodule add --force https://github.com/google/docsy.git themes/docsy
	git submodule update --init --recursive
	npm install postcss-cli autoprefixer postcss

dev: ## Run local server to check you content while writing
	hugo server

build: ## Build the static files of the website
	hugo

docker-base: 
	docker build -f Dockerfile.base -t baker-docs:base .

docker-dev: docker-base ## Use docker for baker website development
	docker run -w /baker -v $$PWD:/baker -p 1313:1313 baker-docs

docker-build-prod: docker-base
	docker build -t baker-docs:prod .

docker-run-prod: docker-build-prod ## Build baker website production version and runs hugo server on port 80
	docker run -p 80:1313 -t baker-docs:prod

.PHONY: docker-base

