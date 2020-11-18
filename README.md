# Baker website

The website is made with [Hugo](https://gohugo.io/) and the [Docsy theme](https://github.com/google/docsy)

> The docsy theme requires hugo extended version! Check the [release page](https://github.com/gohugoio/hugo/releases)
> and download the latest hugo extended version for your OS.

## Add content

[Official documentation](https://www.docsy.dev/docs/adding-content/content/)

## Hugo+Docsy setup

[Official documentation](https://www.docsy.dev/docs/getting-started/)

### Install theme before starting

```sh
git submodule add --force https://github.com/google/docsy.git themes/docsy
git submodule update --init --recursive
npm install postcss-cli autoprefixer postcss
```

### Local server

Hugo will listen on a local port, auto updating the pages while editing the files:

```sh
make dev
```

Or, with docker: 

```sh
make docker-dev
```

### Build the website

```sh
make build
```

Or, with docker, use the following command to build the production version:

```sh
make docker-build-prod
```

And finally, this starts the server locally inside of Docker, with the website bound to host port 80::

```sh
make docker-run-prod
```
