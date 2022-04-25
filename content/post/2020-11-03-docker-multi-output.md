---
layout: post
title: Forking Multi Container Docker Builds
tags: docker
---

Following on from [my last post on Isolated Multistage Docker Builds](/2020/11/01/docker-multistage-containers/), I thought it would be useful to cover another advantage to splitting your dockerfiles: building different output containers from a common base.

## The Problem

When I have an application which when built, needs to have all assets in one container, and a subset of assets in a second container.

For example, writing a node webapp, where you want the compiled/bundled static assets available in the container as a fallback, and also stored in an nginx container for serving.  One of the reasons to do this is separation of concerns: I don't want to put my backend code where it doesn't need to be.  There is also, in this case, the fact that the backend code and nginx version need different base containers, meaning deploying the same container twice won't work.

So let's see how we solve this!

## Creating Separate Dockerfiles

The first dockerfile to write is the common base, which I name `Dockerfile.builder`.  This is the same as the previous post - we are assuming that the `yarn ci:build` step transpiles the typescript, and generates the static assets for our application.

```dockerfile
FROM node:15.0.1-alpine3.12 as builder
WORKDIR /app

COPY . ./
RUN yarn install --frozen-lockfile && yarn cache clean

RUN yarn ci:build
```

Next up is the server container, which will be in the `Dockerfile.backend` file, as try to name the files based on their purpose, rather than their technology used.  As in the previous post, this installs the production dependencies for the application, and copies in the compiled output from the `builder` stage:

```dockerfile
ARG builder_image
FROM ${builder_image} as builder

FROM node:15.0.1-alpine3.12 as output
WORKDIR /app

COPY package.json yarn.lock /app
RUN yarn install --frozen-lockfile --production && yarn cache clean

COPY --from builder /app/dist /app
```

Now let's deal with the `Dockerfile.frontend`.  This uses `nginx:1.19.3-alpine` as a base, and copies in the `nginx.conf` file from the host, and the static assets directory from the `builder` container:

```dockerfile
ARG builder_image
FROM ${builder_image} as builder

FROM nginx:1.19.3-alpine as output

COPY ./nginx.conf /etc/nginx/nginx.conf
COPY --from builder /app/dist/static /app
```

## Building Containers

The reason we rely on the `builder` stage rather than the `backend` output stage is that we are now decoupled from layout/structural changes in that container, and we gain the ability to run the builds in parallel too (the `&` at the end of the lines), for a bit of a speed up on our build agents:

```bash
version="${GIT_COMMIT:0:7}"
builder_tag="builder:$version"

docker build --file Dockerfile.builder -t "$builder_tag" .

# run the builder container here to do tests, lint, static analysis etc.

docker build --file dockerfile.backend --build-arg "builder_image=$builder_tag" -t backend:$version . &
docker build --file Dockerfile.frontend --build-arg "builder_image=$builder_tag" -t frontend:$version . &

wait
```

The result of this is 3 containers, all labled with the short version of the current git commit:

- `builder:abc123e` - contains all packages, compiled output
- `backend:abc123e` - node based, contains the node backend and static assets
- `frontend:abc123e` - nginx based, contains the static assets

I can now publish the builder internally (so it can be cloned before builds for [caching and speed](/2020/05/14/docker-layer-sharing/)), and deploy the `backend` and `frontend` to their different locations.