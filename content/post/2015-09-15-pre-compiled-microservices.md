---
layout: post
title: Running pre-compiled microservices in Docker with Mono
tags: design microservices docker mono
---

Last time we went through [creating a Dockerfile for a microservice][blog-docker], with the service being compiled on creation of the container image, using xbuild.

However we might not want to compile the application to create the container image, and use an existing version (e.g. one created by a build server.)

Our original Dockerfile was this:

```
FROM mono:3.10-onbuild
RUN apt-get update && apt-get install mono-4.0-service -y

CMD [ "mono-service",  "./MicroServiceDemo.exe", "--no-daemon" ]
EXPOSE 12345
```

We only need to make a few modifications to use a pre-compiled application:

```
FROM mono:3.10.0
RUN apt-get update && apt-get install mono-4.0-service -y

RUN mkdir -p /usr/src/app
COPY . /usr/src/app
WORKDIR /usr/src/app

CMD [ "mono-service",  "./MicroServiceDemo.exe", "--no-daemon" ]
EXPOSE 12345
```

Asides from changing the base image to `mono:3.10.0`, the only changes made are to add the following lines:

```
RUN mkdir -p /usr/src/app
COPY . /usr/src/app
WORKDIR /usr/src/app
```

These lines create a new directory for our application, copy the contents of the current directory (e.g. the paths specified when you type `docker build -t servicedemo .`) and make the directory our working directory.

You can now create a container with the same commands as last time:

```bash
docker build -t servicedemo .
docker run -d -p 12345:12345 --name demo servicedemo
```

There is a demo project for all of this on my github: [DockerMonoDemo][github-repo].


[blog-docker]: /2015/09/05/running-microservices-in-docker-with-mono.html
[github-repo]: https://github.com/Pondidum/DockerMonoDemo
