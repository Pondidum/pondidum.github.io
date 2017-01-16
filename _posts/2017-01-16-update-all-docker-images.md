---
layout: post
title: Update all Docker images
tags: docker bash git
---

My work's wifi is *much* faster than my 4G connection, so periodically I want to update all my docker images on my personal laptop while at work.

As I want to just set it going and then forget about it, I use the following one liner to do a `docker pull` against each image on my local machine:

```shell
docker images | grep -v REPOSITORY | awk '{print $1}'| xargs -L1 docker pull
```

Now if only I could get git bash to do TTY properly so I get the pretty download indicators too :(
