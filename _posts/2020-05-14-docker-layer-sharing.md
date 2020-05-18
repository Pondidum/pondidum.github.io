---
layout: post
title: Sharing Docker Layers Between Build Agents
tags: docker
---

Recently, I noticed that when we pull a new version of our application's docker container, it fetches all layers, not just the ones that change.

The problem is that we use ephemeral build agents, which means that each version of the application is built using a different agent, so Docker doesn't know how to share the layers used.  While we can pull the published container before we run the build, this only helps with the final stage of the build.  We want to cache the other stages of the build too, as the earlier layers don't change often, and can be quite slow to build.

We can achieve this by tweaking how we build our stages, which will also allow some other interesting optimisations.

## The Dockerfile

An example dockerfile is below.  There are two stages, `builder` and `prod`.  In the case we are looking at, both the OS packages and application dependencies rarely change, but can take quite a while to install.

```dockerfile
FROM node:14.2.0-alpine3.11 AS builder
WORKDIR /app

RUN apk add --no-cache make gcc g++ python

COPY package.json yarn.lock ./
RUN yarn install --no-progress --frozen-lockfile && \
    yarn cache clean

COPY ./src ./src
RUN yarn build


FROM node:14.2.0-alpine3.11 AS prod
WORKDIR /app

COPY package.json yarn.lock ./

RUN yarn install --production --no-progress --frozen-lockfile && \
    yarn cache clean

COPY --from=builder /app/dist ./dist

CMD ["yarn", "start"]
```

The first step is to try and pull both `:builder` and `:latest` images.  We append `|| true` as the images might not exist yet, and we want the build to pass if they don't!

```bash
docker pull app:builder || true
docker pull app:latest || true
```

Now that we have the application images locally, we can proceed to building the `:builder` stage.  We tag it twice: once with just `app:builder` and once with the short-commit that built it.

```bash
docker build \
    --cache-from=app:builder \
    --target builder \
    -t app:builder-$COMMIT_SHORT \
    -t app:builder \
    .
```

Now that we have built our `builder` stage, we can use this to do lots of other things which require both `dependencies` and `devDependencies`, such as running tests and linters, and we could even distribute these tasks to multiple other machines if we wanted extra parallelism:

```bash
docker run --rm -it app:builder-$COMMIT_SHORT yarn test
docker run --rm -it app:builder-$COMMIT_SHORT yarn test:integration
docker run --rm -it app:builder-$COMMIT_SHORT yarn lint
```

Once we are happy with our tests, we can now build the production container, which we do by using the `--cache-from` directive twice; once with the builder image we just created, and once with the latest version of our application.  Note the order of the `--cache-from` parameters matters; this won't work if you specify the `app:latest` before `app:builder`!

```bash
docker build \
    --cache-from=app:builder \
    --cache-from=app:latest \
    -t app:$COMMIT_SHORT \
    -t app:latest \
    .
```

Now we can publish everything.  We always publish the commit tagged version so that separate branch builds can be fetched and tested, and if the branch is `master`, we publish both the `:builder` and `:latest` tags:

```bash
docker push app:$COMMIT_SHORT

if [ "$BRANCH" == "master" ]; then
    docker push app:builder
    docker push app:latest
fi
```

The full build script looks like this:

```bash
docker pull app:builder || true
docker pull app:latest || true

docker build \
    --cache-from=app:builder \
    --target builder \
    -t app:builder-$COMMIT_SHORT \
    -t app:builder \
    .

# run these in parallel
docker run --rm -it app:builder-$COMMIT_SHORT yarn test
docker run --rm -it app:builder-$COMMIT_SHORT yarn test:integration
docker run --rm -it app:builder-$COMMIT_SHORT yarn lint

docker build \
    --cache-from=app:builder \
    --cache-from=app:latest \
    -t app:$COMMIT_SHORT \
    -t app:latest \
    .

docker push app:$COMMIT_SHORT

if [ "$BRANCH" == "master" ]; then
    docker push app:builder
    docker push app:latest
fi
```

## Effects

By publishing both our `:builder` and `:latest` tags, we can effectively share the layer caches for all build stages across all build agents.  As the layers are shared, pulling the images at the beginning of the builds is pretty fast, and the publishes at the end of the build are very, very fast.

The real benefit comes with building our monolith, which now only needs a small layer to be pulled on deployment, rather than all of the layers, which speeds up our deployments by minutes per host.
