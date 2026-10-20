#!/bin/sh
# Runs check-site on the documents of the cases of the page building
# (docs/ja/designs/site/build.md), in a copy of site/ of this repository.
# Run it after `npm --prefix site ci`. The cases prove CTR-SITE-002 (the
# title and the first heading), CTR-SITE-003 (the links to requirements),
# CTR-SITE-008 (static files only), CTR-SITE-010 (the links between
# documents), CTR-SITE-011 (the sidebar), CTR-SITE-012 (the UI language and the URLs), and CTR-SITE-013 (the copy is no git repository, so the prefix
# names the site). With the trace of trace.json, they prove CTR-DASH-001
# (the requirement tree), CTR-DASH-002 (the summary), CTR-DASH-003 (the
# requirements to file next), CTR-DASH-005 (the links), and CTR-DASH-006 (the
# read time, or only the failed read).
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
cp "$here/trace.json" "$work/site/trace.json"
prefix=$(sed -n 's/^prefix = "\(.*\)"$/\1/p' "$work/.cumin-trace.toml")
npm --prefix "$work/site" run check-site -- --site-name "$prefix"
# CTR-DASH-001: the rows of the requirement tree, written by hand from the
# rules of docs/ja/designs/dashboard/page.md: the categories in the order of
# the trace, the children one level deeper, no deprecated requirement and no
# child of one, no issue closed as not planned, the milestones in title
# order with "なし" last, and each sub-issue once with its state. A category
# takes the title of the document of its first requirement, or its ID when
# no document has it; a requirement has its ID without the prefix and the
# category, and its progress, or "parent" when it has children.
want='[["DM",0,"完了 1 / 3","Data model"],["CTR-DM-001",1,"進行中",[10,11],["v0.1","v0.2"],"1 / 3",["example/other#5 open","example/product#20 closed","example/product#21 open"],"001","in_progress"],["CTR-DM-002",1,"完了 1 / 2",[12],["v0.1"],"",[],"002","parent"],["CTR-DM-002.1",2,"未起票",[],[],"",[],"002.1","not_filed"],["CTR-DM-002.2",2,"完了",[14,15],["v0.1","なし"],"1 / 1",["example/product#31 closed"],"002.2","done"],["OLD",0,"完了 1 / 1","OLD"],["CTR-OLD-001",1,"完了",[17],["v0.1"],"1 / 1",["example/product#40 closed"],"001","done"]]'
got=$(cd "$work/site" && node --input-type=module -e '

import { readSettings, root } from "./src/settings.mjs";
import { areaTitle, readTrace, tree } from "./src/dashboard.mjs";
const rows = tree(readTrace(".."));
const settings = readSettings();
console.log(JSON.stringify(rows.map((r) => (r.kind === "category" ? [r.id, r.depth, r.progress, areaTitle(settings, root, r)] : [r.id, r.depth, r.progress, r.issues, r.milestones, r.implementation, r.subIssues.map((s) => `${s.repository}#${s.number} ${s.closed ? "closed" : "open"}`), r.short, r.state]))));
')
if [ "$got" != "$want" ]; then
  echo "the tree is $got"
  echo "want $want"
  exit 1
fi
echo "the tree follows the rules of the design, as wanted"
# CTR-DASH-002, CTR-DASH-003: the summary and the requirements to file next,
# written by hand from the rules of docs/ja/designs/dashboard/page.md: only
# the requirements with no children count, each once in each milestone of its
# requirement issues, with no issue closed as not planned and no deprecated
# requirement or child of one. The row of no milestone is last.
want='{"rows":[{"milestone":"v0.1","in_progress":1,"done":2},{"milestone":"v0.2","in_progress":1,"done":0},{"milestone":"(マイルストーンなし)","in_progress":0,"done":1}],"notFiled":1}|[{"id":"CTR-DM-002.1","title":"The first child"}]'
got=$(cd "$work/site" && node --input-type=module -e '
import { fileNext, readTrace, summary } from "./src/dashboard.mjs";
const trace = readTrace("..");
console.log(`${JSON.stringify(summary(trace))}|${JSON.stringify(fileNext(trace))}`);
')
if [ "$got" != "$want" ]; then
  echo "the summary and the requirements to file next are $got"
  echo "want $want"
  exit 1
fi
echo "the summary and the requirements to file next follow the rules of the design, as wanted"
# CTR-DASH-005: the links of the dashboard.
got=$(cd "$work/site" && node --input-type=module -e '
import { readSettings, root } from "./src/settings.mjs";
import { idTable } from "./src/requirements.mjs";
import { issueURL, requirementURL } from "./src/dashboard.mjs";
const table = idTable(readSettings(), root);
console.log([requirementURL(table, "CTR-DM-002.1"), requirementURL(table, "CTR-OLD-001"), issueURL("example/other", 5)].map(String).join(" "));
')
want='/cases/requirements/data-model_v2/#CTR-DM-002.1 undefined https://github.com/example/other/issues/5'
if [ "$got" != "$want" ]; then
  echo "the links are $got"
  echo "want $want"
  exit 1
fi
echo "the links of the dashboard go to the site and to GitHub, as wanted"
# CTR-DASH-006: the read time is in Japan time, whatever the time zone of the
# build machine, and the page keeps read_at in the datetime attribute.
got=$(cd "$work/site" && TZ=America/Los_Angeles node --input-type=module -e '
import { japanTime } from "./src/dashboard.mjs";
console.log([japanTime("2026-10-17T09:10:55Z"), japanTime("2026-01-04T20:05:03Z")].join("|"));
')
want='2026-10-17 18:10:55|2026-1-5 5:05:03'
if [ "$got" != "$want" ]; then
  echo "the read times are $got"
  echo "want $want"
  exit 1
fi
if ! grep -q '<time datetime="2026-10-17T09:10:55Z">2026-10-17 18:10:55</time>' "$work/site/dist/dashboard/index.html"; then
  echo "the dashboard does not show the read time 2026-10-17 18:10:55 with its datetime"
  exit 1
fi
echo "the dashboard shows the read time in Japan time, as wanted"
# CTR-DASH-006: a trace.json that is no JSON shows only the failed read.
printf 'not json\n' > "$work/site/trace.json"
npm --prefix "$work/site" run check-site -- --site-name "$prefix" > "$work/out.txt" 2>&1 || {
  cat "$work/out.txt"
  exit 1
}
if ! grep -q '>GitHub を読めなかったため、進捗を表示できません。</p></div>' "$work/site/dist/dashboard/index.html"; then
  echo "the dashboard does not show only the failed read with a trace.json that is no JSON"
  exit 1
fi
echo "the dashboard shows only the failed read with a trace.json that is no JSON, as wanted"
rm "$work/site/trace.json"
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
# builds the site with sidebar/.drafts/draft.md, which the entry document
# links to, and sidebar/.hidden.md.
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
