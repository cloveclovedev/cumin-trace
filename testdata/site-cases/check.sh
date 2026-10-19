#!/bin/sh
# Runs check-site on the documents of the cases of the page building
# (docs/ja/designs/site/build.md), in a copy of site/ of this repository.
# Run it after `npm --prefix site ci`. The cases prove CTR-SITE-002 (the
# title and the first heading), CTR-SITE-003 (the links to requirements),
# CTR-SITE-008 (static files only), CTR-SITE-010 (the links between
# documents), and CTR-SITE-013 (the copy is no git repository, so the prefix
# names the site).
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
prefix=$(sed -n 's/^prefix = "\(.*\)"$/\1/p' "$work/.cumin-trace.toml")
npm --prefix "$work/site" run check-site -- --site-name "$prefix"
# CTR-SITE-008: a functions/ directory in the output that holds no page of a
# document fails the check.
rm -r "$work/docs/ja/functions"
mkdir -p "$work/site/public/functions"
printf 'export function onRequest() {}\n' > "$work/site/public/functions/api.js"
if npm --prefix "$work/site" run check-site -- --site-name "$prefix" > "$work/out.txt" 2>&1; then
  echo "check-site passed with a server function in site/dist/functions"
  exit 1
fi
if ! grep -q '^site/dist/functions: the output has a server function' "$work/out.txt"; then
  cat "$work/out.txt"
  echo "check-site failed for another reason than the server function"
  exit 1
fi
echo "check-site failed with a server function in site/dist/functions, as wanted"
