// Builds the site and checks the HTML of the output. It prints one line for
// each problem and ends with the exit code 1 when one exists.
import { spawnSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { readSettings, root } from "../src/settings.mjs";
import { escape, hasTitle, headingIDs, linkTarget, pageURL } from "../src/requirements.mjs";

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

const docs = join(root, settings.docsDir);
const id = `${settings.prefix}-[A-Z0-9]+-[0-9]{3}(?:\\.[1-9][0-9]*)?`;

// CTR-SITE-002: a page whose title comes from its first heading shows the
// title once, in one <h1>.
for (const entry of existsSync(docs) ? readdirSync(docs, { recursive: true, withFileTypes: true }) : []) {
  if (!entry.isFile() || !entry.name.endsWith(".md")) continue;
  const file = join(entry.parentPath, entry.name);
  if (hasTitle(readFileSync(file, "utf8"))) continue;
  const html = page(pageURL(settings.base, relative(docs, file)));
  // A requirement heading of level 1 is no title.
  const count = html?.match(new RegExp(`<h1(?! id="${id}")[\\s>]`, "g"))?.length ?? 0;
  if (html !== undefined && count !== 1) problems.push(`${relative(root, file)}: the page has ${count} <h1>, want 1`);
}

// CTR-SITE-003: each requirement heading has an anchor whose name is its ID.
// CTR-SITE-006: each requirement heading starts its own block, and the ID is
// an element apart from the title.
const pages = new Map();
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
    if (!new RegExp(`<h[1-6][^>]* id="${escape(id)}"`).test(html)) {
      problems.push(`${relative(root, file)}: the heading of ${id} has no anchor ${id}`);
    }
    if (!new RegExp(`<section class="requirement"><h[1-6][^>]* id="${escape(id)}"[^>]*><code class="requirement-id">${escape(id)}</code>`).test(html)) {
      problems.push(`${relative(root, file)}: ${id} is not a block that starts with its heading and its ID apart`);
    }
  }
  for (const match of html.matchAll(/<section class="requirement">(?!<h[1-6])/g)) {
    problems.push(`${relative(root, file)}: a block of a requirement starts with no heading, at ${match.index}`);
  }
}

// CTR-SITE-003: each link to a requirement ID goes to a page under base that
// has the anchor, and no link stands inside a code block.
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

// sourceLinks returns the destination and the text of each inline link of a
// document that is no image, outside the code blocks and the code spans.
function sourceLinks(text) {
  const found = [];
  let fence = "";
  for (const line of text.split(/\r?\n/)) {
    const open = line.match(/^ {0,3}(`{3,}|~{3,})/);
    if (fence) {
      if (open && open[1][0] === fence[0] && open[1].length >= fence.length && line.trim() === open[1]) fence = "";
      continue;
    }
    if (open) {
      fence = open[1];
      continue;
    }
    const plain = line.replace(/`[^`]*`/g, (span) => span.replace(/[[\]()]/g, " "));
    for (const match of plain.matchAll(/(?<!!)\[([^\]]*)\]\(<?([^)\s>]+)>?(?:\s+"[^"]*")?\)/g)) {
      found.push({ text: line.slice(match.index + 1, match.index + 1 + match[1].length), href: match[2] });
    }
  }
  return found;
}

// decoded returns the URL with no percent-encoding, or the URL itself when it
// holds a malformed one.
function decoded(url) {
  try {
    return decodeURI(url);
  } catch {
    return url;
  }
}

// plainText returns the text of an HTML fragment with no tags, no entities,
// and no white space.
function plainText(html) {
  const entities = { amp: "&", lt: "<", gt: ">", quot: '"', apos: "'", nbsp: " " };
  return html
    .replace(/<[^>]*>/g, "")
    .replace(/&(#x[0-9a-f]+|#[0-9]+|[a-z]+);/gi, (all, name) => {
      if (name[0] === "#") return String.fromCodePoint(name[1] === "x" || name[1] === "X" ? parseInt(name.slice(2), 16) : parseInt(name.slice(1), 10));
      return entities[name] ?? all;
    })
    .replace(/\s+/g, "");
}

// CTR-SITE-010: each relative link of a document goes to the page of the
// document and its heading, shows a diagram source in a closed collapsible
// block, or keeps only its text. A link to a missing page or heading is a
// broken link.
let documents = 0;
for (const entry of existsSync(docs) ? readdirSync(docs, { recursive: true, withFileTypes: true }) : []) {
  if (!entry.isFile() || !entry.name.endsWith(".md")) continue;
  const file = join(entry.parentPath, entry.name);
  const url = pageURL(settings.base, relative(docs, file));
  const html = page(url);
  const name = relative(root, file);
  if (html === undefined) {
    problems.push(`${name}: no page at ${url}`);
    continue;
  }
  documents++;
  const content = html.slice(html.indexOf("sl-markdown-content"));
  if (/<a [^>]*href="[^"]*\.puml(#[^"]*)?"/.test(content)) problems.push(`${name}: the page ${url} has a link to a .puml file`);
  const hrefs = new Set([...content.matchAll(/<a [^>]*href="([^"]*)"/g)].map((m) => decoded(m[1])));
  const blocks = [...content.matchAll(/<details>([\s\S]*?)<\/details>/g)].map((m) => m[1]);
  for (const link of sourceLinks(readFileSync(file, "utf8"))) {
    const target = linkTarget(settings, root, file, link.href);
    if (target.kind === "page") {
      const [path, hash] = target.url.split("#");
      const html = page(path);
      if (!hrefs.has(decoded(target.url))) {
        problems.push(`${name}: the link to ${link.href} does not go to ${target.url}`);
      } else if (html === undefined) {
        problems.push(`${name}: the link to ${link.href} is broken: no page at ${path}`);
      } else if (hash !== undefined && !html.includes(` id="${decoded(hash)}"`)) {
        problems.push(`${name}: the link to ${link.href} is broken: ${path} has no heading #${hash}`);
      }
    } else if (target.kind === "puml") {
      const source = plainText(readFileSync(target.file, "utf8").replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c]));
      if (!blocks.some((block) => plainText(block).includes(source))) {
        problems.push(`${name}: the link to ${link.href} has no closed collapsible block with the content of the file`);
      }
    } else if (target.kind === "text" && hrefs.has(decoded(link.href))) {
      problems.push(`${name}: the link to ${link.href} is still a link, want only its text`);
    }
  }
}

for (const problem of problems) console.log(problem);
if (problems.length > 0) process.exit(1);
console.log(`check-site: ${pages.size} pages of requirements and ${documents} pages of documents checked`);
