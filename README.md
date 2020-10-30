# Baker website

The website is made with [Hugo](https://gohugo.io/) and the [Docsy theme](https://github.com/google/docsy)

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

```bash
make dev
```

### Build the website

```bash
make build
```
