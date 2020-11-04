#!/bin/bash

docker run \
    --rm \
    -it \
    -v "$PWD:/srv/jekyll" \
    -p 4000:4000 \
    -p 4040:4040 \
    jekyll/jekyll:pages \
    jekyll serve --watch --livereload --livereload-port 4040
