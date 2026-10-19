#!/bin/sh
# Runs check-site on the documents of the cases of the page building
# (docs/ja/designs/site/build.md), in a copy of site/ of this repository.
# Run it after `npm --prefix site ci`. The cases prove CTR-SITE-002 (the
# title and the first heading), CTR-SITE-003 (the links to requirements),
# CTR-SITE-008 (static files only), CTR-SITE-010 (the links between
# documents), CTR-SITE-011 (the sidebar), CTR-SITE-012 (the UI language and the URLs), and CTR-SITE-013 (the copy is no git repository, so the prefix
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
# CTR-SITE-011: the sidebar follows the links of the entry document, and the
# pages that the walk does not place go to the end of their deepest group, in
# path order.
want='[{"slug":"index"},{"slug":"sidebar/zeta"},{"label":"Beta","items":[{"slug":"sidebar/beta"},{"slug":"sidebar/beta/one"},{"slug":"sidebar/beta/unlinked"}]},{"label":"Alpha","items":[{"slug":"sidebar/alpha"},{"slug":"sidebar/alpha/one"},{"slug":"sidebar/alpha/three"},{"slug":"sidebar/alpha/two"}]}]'
got=$(cd "$work/site" && node --input-type=module -e '
import { readSettings, root } from "./src/settings.mjs";
import { sidebar } from "./src/requirements.mjs";
console.log(JSON.stringify(sidebar(readSettings(), root).slice(0, 4)));
')
if [ "$got" != "$want" ]; then
  echo "the sidebar starts with $got"
  echo "want $want"
  exit 1
fi
echo "the sidebar follows the order of the cases, as wanted"
# CTR-SITE-011: the sidebar lists no document under a directory or with a name
# that starts with ".", as the docs collection loads none. check-site above
# builds the site with sidebar/.drafts/draft.md and sidebar/.hidden.md.
all=$(cd "$work/site" && node --input-type=module -e '
import { readSettings, root } from "./src/settings.mjs";
import { sidebar } from "./src/requirements.mjs";
console.log(JSON.stringify(sidebar(readSettings(), root)));
')
case "$all" in
*draft* | *hidden*)
  echo "the sidebar lists a document of a dot path: $all"
  exit 1
  ;;
esac
echo "the sidebar lists no document of a dot path, as wanted"
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
