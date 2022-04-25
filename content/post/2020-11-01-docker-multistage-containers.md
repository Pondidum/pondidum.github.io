---
date: "2020-11-01T00:00:00Z"
tags: docker
title: Isolated Docker Multistage Images
---

Often when building applications, I will use a multistage docker build for output container size and efficiency, but will run the build in two halves, to make use of the extra assets in the builder container, something like this:

```bash
docker build \
  --target builder \
  -t builder:$GIT_COMMIT \
  .

docker run --rm \
  -v "$PWD/artefacts/tests:/artefacts/tests" \
  builder:$GIT_COMMIT \
  yarn ci:test

docker run --rm \
  -v "$PWD/artefacts/lint:/artefacts/lint" \
  builder:$GIT_COMMIT \
  yarn ci:lint

docker build \
  --cache-from builder:$GIT_COMMIT \
  --target output \
  -t app:$GIT_COMMIT \
  .
```

This usually works fine, but sometimes the `.dockerignore` file won't have everything set correctly, and docker will decide that when it runs the last `build` command, that it needs to rebuild the `builder` container too, which is pretty irritating.

The first solution is to try and figure out what you need to add to your `.dockerignore` file, which depending on your repository structure and container usage, might be more hassle than it's worth.

The second solution is to prevent docker invalidating the first layers at all, by splitting the build into separate files.

## Splitting the Dockerfile

Let's start with an example docker file, which is a generic yarn based application with multistage build configured:

```dockerfile
FROM node:15.0.1-alpine3.12 as builder
WORKDIR /app

COPY . ./
RUN yarn install --frozen-lockfile && yarn cache clean

RUN yarn ci:build


FROM node:15.0.1-alpine3.12 as output
WORKDIR /app

COPY package.json yarn.lock /app
RUN yarn install --frozen-lockfile --production && yarn cache clean

COPY --from builder /app/dist /app
```

The first file will be our `Docker.builder`, which is a direct copy paste:

```dockerfile
FROM node:15.0.1-alpine3.12 as builder
WORKDIR /app

COPY . ./
RUN yarn install --frozen-lockfile && yarn cache clean

RUN yarn ci:build
```

The second file can also be a direct copy paste, saved as `Dockerfile.output`, but it has a problem:

```dockerfile
FROM node:15.0.1-alpine3.12 as output
WORKDIR /app

COPY package.json yarn.lock /app
RUN yarn install --frozen-lockfile --production && yarn cache clean

COPY --from builder /app/dist /app
```

We want to copy from a different container, not a different stage, and while the `COPY` command does let you specify another container in the `--from` parameter, but we really want to specify which container it is at build time.  The first attempt at solving this was using a buildarg:

```dockerfile
ARG builder_image
COPY --from ${builder_image} /app/dist /app
```

But alas, this doesn't work either, as the `--from` parameter doesn't support variables. The solution turns out to be that `FROM` command _does_ support parameterisation, so we can (ab)use that:

```dockerfile
ARG builder_image
FROM ${builder_image} as builder

FROM node:15.0.1-alpine3.12 as output
WORKDIR /app

COPY package.json yarn.lock /app
RUN yarn install --frozen-lockfile --production && yarn cache clean

COPY --from builder /app/dist /app
```

Now our build script can use the `--build-arg` parameter to force the right container:

```diff
docker build \
-  --target builder \
+  --file Dockerfile.builder \
  -t builder:$GIT_COMMIT \
  .

docker run --rm \
  -v "$PWD/artefacts/tests:/artefacts/tests" \
  builder:$GIT_COMMIT \
  yarn ci:test

docker run --rm \
  -v "$PWD/artefacts/lint:/artefacts/lint" \
  builder:$GIT_COMMIT \
  yarn ci:lint

docker build \
-  --cache-from builder:$GIT_COMMIT \
+  --build-arg "builder_image=builder:$GIT_COMMIT" \
+  --file Dockerfile.output \
  -t app:$GIT_COMMIT \
  .
```

We can now safely modfiy the working directory to our heart's content without worring about invalidating the layer caches.
