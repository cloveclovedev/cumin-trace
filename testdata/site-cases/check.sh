#!/bin/sh
# Runs check-site on the documents of the cases of the page building
# (docs/ja/designs/site/build.md), in a copy of site/ of this repository.
# Run it after `npm --prefix site ci`. The cases prove CTR-SITE-002 (the
# title and the first heading), CTR-SITE-003 (the links to requirements),
# CTR-SITE-008 (static files only), CTR-SITE-010 (the links between
# documents), CTR-SITE-011 (the sidebar), CTR-SITE-012 (the UI language and the URLs), and CTR-SITE-013 (the copy is no git repository, so the prefix
# names the site). With the trace of trace.json, they prove CTR-DASH-001
# (the requirements by category), CTR-DASH-005 (the links), CTR-DASH-006 (the
# read time, or only the failed read), and CTR-DASH-009 (the discrepancies).
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
# category, and its progress: the count of its children for a requirement
# with children, whose state is its progress, or "parent" with no progress
# (an older trace). A not filed requirement with changed_after_done true has
# the text "変更後の未起票" and the mark of a change; a trace with no
# changed_after_done, as the other requirements of the fixture, marks none.
want='[["DM",0,"完了 1 / 3","Data model"],["CTR-DM-001",1,"進行中",[10,11],["v0.1","v0.2"],"1 / 3",["example/other#5 open","example/product#20 closed","example/product#21 open"],"001","in_progress",false],["CTR-DM-002",1,"完了 1 / 2",[12],["v0.1"],"",[],"002","in_progress",false],["CTR-DM-002.1",2,"未起票",[],[],"",[],"002.1","not_filed",false],["CTR-DM-002.2",2,"完了",[14,15],["v0.1","なし"],"1 / 1",["example/product#31 closed"],"002.2","done",false],["OLD",0,"完了 1 / 2","OLD"],["CTR-OLD-001",1,"完了",[17],["v0.1"],"1 / 1",["example/product#40 closed"],"001","done",false],["CTR-OLD-002",1,"変更後の未起票",[19],["v0.1"],"",[],"002","not_filed",true]]'
got=$(cd "$work/site" && node --input-type=module -e '

import { readSettings, root } from "./src/settings.mjs";
import { areaTitle, readTrace, tree } from "./src/dashboard.mjs";
const rows = tree(readTrace(".."));
const settings = readSettings();
console.log(JSON.stringify(rows.map((r) => (r.kind === "category" ? [r.id, r.depth, r.progress, areaTitle(settings, root, r)] : [r.id, r.depth, r.progress, r.issues, r.milestones, r.implementation, r.subIssues.map((s) => `${s.repository}#${s.number} ${s.closed ? "closed" : "open"}`), r.short, r.state, r.changed]))));
')
if [ "$got" != "$want" ]; then
  echo "the tree is $got"
  echo "want $want"
  exit 1
fi
echo "the tree follows the rules of the design, as wanted"
# CTR-DASH-001: the tile and the line of the requirement that changed after
# it was done carry the mark, and no other tile or line does.
html="$work/site/dist/dashboard/index.html"
for class in dashboard-tile dashboard-line; do
  marked=$(grep -o "class=\"$class\"[^>]*data-changed=\"true\"" "$html" | sed -n 's/.*data-id="\([^"]*\)".*/\1/p' | tr '\n' ' ')
  if [ "$marked" != "CTR-OLD-002 " ]; then
    echo "the $class elements with the mark of a change are: $marked"
    echo "want CTR-OLD-002"
    exit 1
  fi
done
echo "the map and the list mark the requirement that changed after it was done, as wanted"
# CTR-DASH-001: a requirement with children in a trace from an older
# cumin-trace has no progress, and keeps the neutral "parent" look.
got=$(cd "$work/site" && node --input-type=module -e '
import { readTrace, tree } from "./src/dashboard.mjs";
const trace = readTrace("..");
trace.requirements.find((r) => r.id === "CTR-DM-002").progress = null;
const row = tree(trace).find((r) => r.id === "CTR-DM-002");
console.log(JSON.stringify([row.state, row.progress]));
')
want='["parent","完了 1 / 2"]'
if [ "$got" != "$want" ]; then
  echo "the row of a parent with no progress is $got"
  echo "want $want"
  exit 1
fi
echo "a parent with no progress keeps the parent look, as wanted"
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
# CTR-DASH-009: the discrepancies in the order of the trace, with the kind
# texts of docs/ja/designs/dashboard/page.md, written by hand.
got=$(cd "$work/site" && node --input-type=module -e '
import { discrepancies, readTrace } from "./src/dashboard.mjs";
console.log(JSON.stringify(discrepancies(readTrace(".."))));
')
want='[{"kind":"open_issue_for_deprecated","label":"廃止した要件の開いた要求Issue","issue":16,"id":"CTR-DM-003"},{"kind":"unknown_id","label":"要件文書にないID","issue":18,"id":"CTR-DM-099"}]'
if [ "$got" != "$want" ]; then
  echo "the discrepancies are $got"
  echo "want $want"
  exit 1
fi
if ! grep -q 'class="dashboard-discrepancy"' "$work/site/dist/dashboard/index.html"; then
  echo "the dashboard shows no discrepancy with a trace that has two"
  exit 1
fi
echo "the dashboard lists the discrepancies of the trace, as wanted"
# CTR-DASH-009: a trace with no discrepancy shows no discrepancy section.
node -e '
const fs = require("node:fs");
const trace = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
trace.discrepancies = [];
fs.writeFileSync(process.argv[1], JSON.stringify(trace));
' "$work/site/trace.json"
npm --prefix "$work/site" run check-site -- --site-name "$prefix" > "$work/out.txt" 2>&1 || {
  cat "$work/out.txt"
  exit 1
}
if grep -q 'dashboard-discrepancy\|>不整合</h2>' "$work/site/dist/dashboard/index.html"; then
  echo "the dashboard shows a discrepancy section with a trace that has none"
  exit 1
fi
echo "the dashboard shows no discrepancy section with a trace that has none, as wanted"
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
# CTR-SITE-011, CTR-SITE-015: the sidebar follows the links of the entry
# document, and the pages that no index.md places go to the end of their
# deepest group, in path order. A placed page or group shows the text of its
# link. The "##" heading of the entry document makes a group of the top
# level. The entry document is not listed, also when it links to itself. The
# entry document places the page of gamma before the index.md of gamma does,
# so the group of gamma holds only its index.md and shows as that page.
want='[{"slug":"sidebar/zeta","label":"Zeta"},{"label":"Beta","items":[{"slug":"sidebar/beta"},{"slug":"sidebar/beta/one","label":"One"},{"slug":"sidebar/beta/unlinked"}]},{"label":"Alpha by its link","items":[{"slug":"sidebar/alpha"},{"slug":"sidebar/alpha/one","label":"One"},{"slug":"sidebar/alpha/three"},{"slug":"sidebar/alpha/two"}]},{"label":"Sections","items":[{"slug":"sidebar/gamma","label":"Gamma by its link"},{"slug":"sidebar/gamma/one","label":"One of gamma"}]}]'
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
