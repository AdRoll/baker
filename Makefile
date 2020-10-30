# This Makefile can be used to obtain a Linux binary through docker (eg: from OSX)
# For normal development, Docker is not required. See README.md for build instructions.
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
