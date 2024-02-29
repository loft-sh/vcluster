# Documentation

This website is built using [Docusaurus 2](https://v2.docusaurus.io/), a modern static website generator.
That means, for installing and developing this docusaurus documentation, you will need to have node@16 or higher. 

## Local development

Execute these commands from the `/docs/` directory.

### Installation

From the `vcluster/docs` directory, execute:

```
$ yarn
```

### Local server

```
$ yarn start
```

This command starts a local development server and opens up a browser window. Most changes are reflected live without having to restart the server.

### Build

```
$ yarn build
```

This command generates static content into the `build` directory and can be served using any static contents hosting service.

## Pull request deploy preview

Netlify generates a deploy preview URL. To see your changes, append `docs/` to the generated URL. 

## Creating new versions

### 1. Generate command docs

```bash
cd ../ # main project directory
go run -mod=vendor ./hack/gen-docs.go
```

### 2. Create version

```bash
yarn run docusaurus docs:version 0.1
```
