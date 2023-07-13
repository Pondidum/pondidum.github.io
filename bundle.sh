#!/bin/sh

set -eu

rg --files-with-matches --fixed-strings "(/images/" ./content/post | sort | while read -r file; do
  echo "==> $file"

  dir_name=$(basename "$file" ".md")
  dir_path="./content/post/${dir_name}"

  echo "    dir path: $dir_path"
  mkdir -p "$dir_path"

  echo "    inline references"

  rg --only-matching "\(/images/.*\)" --no-line-number "$file" | tr -d "()" | uniq | while read -r image_path; do
    echo "    - $image_path"

    image_name=$(basename "$image_path")

    echo "    - $image_name"

    mv "./static$image_path" "$dir_path" || true
    sed -i "s,($image_path),($image_name),g" "$file"

  done


  echo "    move file"
  mv "$file" "$dir_path/index.md"

done