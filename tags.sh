#! /bin/bash

ls _posts/*.md | xargs grep -h '^tags:.*' | sed 's/tags: //g' | sed 's/ /\n/g' | sort | uniq -c
