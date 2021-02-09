.PHONY: help setup git-update setup-git dev build docker-setup docker-dev docker-build docker-run-prod clear

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

setup: setup-git ## Setup Hugo locally 
	npm install postcss-cli autoprefixer postcss

dev: git-update ## Run local server to check you content while writing
	hugo server

build: git-update ## Build the static files of the website
	HUGO_ENV=production hugo

docker-setup: setup-git ## Setup Hugo with Docker

docker-dev: git-update docker-base ## Run local server with Docker to check you content while writing
	docker run -w /baker -v $$PWD:/baker -p 1313:1313 --user $$(id -u):$$(id -g) -it baker-docs:base server --bind=0.0.0.0

docker-build: git-update docker-base ## Build the static files of the website with Docker 
	docker run -w /baker -v $$PWD:/baker -p 1313:1313 --user $$(id -u):$$(id -g) -e HUGO_ENV=production -it baker-docs:base

docker-run-prod: git-update docker-base ## Run the website in production on port :80 with Docker
	docker run -w /baker -v $$PWD:/baker -p 80:1313 --user $$(id -u):$$(id -g) -e HUGO_ENV=production -it baker-docs:base server --bind=0.0.0.0

clear: ## Delete all created files
	rm -fr resources node_modules themes/docsy

setup-git:
	git submodule add --force https://github.com/google/docsy.git themes/docsy
	git submodule update --init --recursive	

docker-base:
	docker build -t baker-docs:base .

git-update:
	git submodule update --init --recursive
