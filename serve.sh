#!/bin/bash

docker run \
    --rm \
    -it \
    -v "$PWD:/srv/jekyll" \
    -p 4000:4000 \
    jekyll/jekyll:pages \
    jekyll serve --watch --incremental
