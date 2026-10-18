// Builds the site and checks the HTML of the output. It prints one line for
// each problem and ends with the exit code 1 when one exists.
import { spawnSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { readSettings, root } from "../src/settings.mjs";
import { headingIDs, pageURL } from "../src/requirements.mjs";

const site = join(dirname(fileURLToPath(import.meta.url)), "..");
const dist = join(site, "dist");
const settings = readSettings();
const problems = [];

const build = spawnSync(join(site, "node_modules", ".bin", "astro"), ["build"], { cwd: site, stdio: "inherit" });
if (build.error) console.log(`astro build: ${build.error.message}`);
if (build.status !== 0) process.exit(build.status ?? 1);

// page returns the HTML of the output page of a URL under base, or undefined.
function page(url) {
  const file = join(dist, url.slice(settings.base.length), "index.html");
  return existsSync(file) ? readFileSync(file, "utf8") : undefined;
}

// CTR-SITE-003: each requirement heading has an anchor whose name is its ID.
const pages = new Map();
const docs = join(root, settings.docsDir);
const requirements = join(root, settings.requirementsDir);
for (const entry of existsSync(requirements) ? readdirSync(requirements, { recursive: true, withFileTypes: true }) : []) {
  if (!entry.isFile() || !entry.name.endsWith(".md")) continue;
  const file = join(entry.parentPath, entry.name);
  const url = pageURL(settings.base, relative(docs, file));
  const html = page(url);
  if (html === undefined) {
    problems.push(`${relative(root, file)}: no page at ${url}`);
    continue;
  }
  pages.set(url, html);
  for (const id of headingIDs(settings.prefix, readFileSync(file, "utf8"))) {
    if (!new RegExp(`<h[1-6][^>]* id="${id}"`).test(html)) {
      problems.push(`${relative(root, file)}: the heading of ${id} has no anchor ${id}`);
    }
  }
}

// CTR-SITE-003: each link to a requirement ID goes to a page under base that
// has the anchor, and no link stands inside a code block.
const id = `${settings.prefix}-[A-Z0-9]+-[0-9]{3}(?:\\.[1-9][0-9]*)?`;
for (const [url, html] of pages) {
  for (const match of html.matchAll(new RegExp(`<a href="([^"#]+)#(${id})"`, "g"))) {
    const [, target, anchor] = match;
    const html = target.startsWith(settings.base) ? page(target) : undefined;
    if (html === undefined) {
      problems.push(`${url}: the link to ${anchor} goes to ${target}, which is no page under ${settings.base}`);
    } else if (!html.includes(` id="${anchor}"`)) {
      problems.push(`${url}: the link to ${anchor} goes to ${target}, which has no anchor ${anchor}`);
    }
  }
  for (const block of html.matchAll(/<pre[\s\S]*?<\/pre>/g)) {
    if (block[0].includes("<a ")) problems.push(`${url}: a code block holds a link`);
  }
}

for (const problem of problems) console.log(problem);
if (problems.length > 0) process.exit(1);
console.log(`check-site: ${pages.size} pages of requirements checked`);
