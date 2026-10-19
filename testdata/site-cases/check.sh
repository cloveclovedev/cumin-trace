#!/bin/sh
# Runs check-site on the documents of the cases of the page building
# (docs/ja/designs/site/build.md), in a copy of site/ of this repository.
# Run it after `npm --prefix site ci`. The cases prove CTR-SITE-002 (the
# title and the first heading), CTR-SITE-003 (the links to requirements), and
# CTR-SITE-010 (the links between documents).
set -eu
here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cp -R "$here/.cumin-trace.toml" "$here/docs" "$work/"
# A Go module allows no "?" in a file name, so the script writes this case.
printf '# What\n\nThe file name holds a question mark.\n' > "$work/docs/ja/links/what?.md"
mkdir "$work/site"
for name in astro.config.mjs package.json node_modules scripts src; do
  cp -R "$here/../../site/$name" "$work/site/"
done
npm --prefix "$work/site" run check-site
