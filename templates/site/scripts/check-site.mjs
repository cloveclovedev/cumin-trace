// Builds the site and checks the HTML of the output. It prints one line for
// each problem and ends with the exit code 1 when one exists.
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { existsSync, readdirSync, readFileSync, rmSync, statSync } from "node:fs";
import { basename, dirname, extname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { readSettings, root } from "../src/settings.mjs";
import { atxHeadings, body as markdownBody, diagramScript, escape, firstHeading, frontmatter, hasTitle, headingIDs, linkTarget, pageURL } from "../src/requirements.mjs";

const site = join(dirname(fileURLToPath(import.meta.url)), "..");
const dist = join(site, "dist");
const settings = readSettings();
const problems = [];
const docs = join(root, settings.docsDir);

// digest returns the SHA-256 of a file, or undefined when no file is there.
function digest(file) {
  return existsSync(file) && statSync(file).isFile() ? createHash("sha256").update(readFileSync(file)).digest("hex") : undefined;
}

// digests returns the SHA-256 of each file under docs_dir, by its path.
function digests() {
  const found = new Map();
  for (const entry of existsSync(docs) ? readdirSync(docs, { recursive: true, withFileTypes: true }) : []) {
    if (!entry.isFile()) continue;
    const file = join(entry.parentPath, entry.name);
    found.set(relative(root, file), digest(file));
  }
  return found;
}

// Astro reuses the rendered documents of its cache, so the check removes the
// cache and the old output, and builds from the documents of now.
for (const dir of [dist, join(site, ".astro"), join(site, "node_modules", ".astro")]) rmSync(dir, { recursive: true, force: true });
const before = digests();
const build = spawnSync(join(site, "node_modules", ".bin", "astro"), ["build"], { cwd: site, stdio: "inherit" });
if (build.error) console.log(`astro build: ${build.error.message}`);
else if (build.status !== 0) console.log(`astro build: ended with ${build.status === null ? `the signal ${build.signal}` : `the exit code ${build.status}`}`);
if (build.error || build.status !== 0) process.exit(build.status || 1);

// CTR-SITE-002: the build leaves each file under docs_dir as it was.
const after = digests();
for (const name of new Set([...before.keys(), ...after.keys()])) {
  if (before.get(name) !== after.get(name)) problems.push(`${name}: the build changed the file`);
}

// pageFile returns the path of the output page of a URL under base.
function pageFile(url) {
  return join(dist, url.slice(settings.base.length), "index.html");
}

// page returns the HTML of the output page of a URL under base, or undefined.
function page(url) {
  const file = pageFile(url);
  return existsSync(file) ? readFileSync(file, "utf8") : undefined;
}

const id = `${settings.prefix}-[A-Z0-9]+-[0-9]{3}(?:\\.[1-9][0-9]*)?`;

// CTR-SITE-008: the site has no server functions, only static files. A
// directory of documents named "server" or "functions" puts their pages in
// the output directory of that name, so such a directory may hold the pages
// of documents, and no other file.
const config = readFileSync(join(site, "astro.config.mjs"), "utf8");
if (/\badapter\s*:/.test(config)) problems.push("site/astro.config.mjs: the config has an adapter, want static files only");
if (/\boutput\s*:\s*["'`]server["'`]/.test(config)) problems.push('site/astro.config.mjs: the config has output "server", want static files only');
if (existsSync(join(dist, "_worker.js"))) problems.push("site/dist/_worker.js: the output has a server function, want static files only");
const documentPages = new Set();
for (const entry of existsSync(docs) ? readdirSync(docs, { recursive: true, withFileTypes: true }) : []) {
  if (entry.isFile() && entry.name.endsWith(".md")) documentPages.add(pageFile(pageURL(settings.base, relative(docs, join(entry.parentPath, entry.name)))));
}
for (const name of ["_functions", "functions", "server"]) {
  const dir = join(dist, name);
  if (!existsSync(dir)) continue;
  const files = statSync(dir).isDirectory() ? readdirSync(dir, { recursive: true, withFileTypes: true }).filter((e) => !e.isDirectory()).map((e) => join(e.parentPath, e.name)) : [dir];
  const others = files.filter((file) => !documentPages.has(file));
  if (files.length === 0 || others.length > 0) {
    problems.push(`site/dist/${name}: the output has a server function, want static files only: ${others.map((file) => relative(dist, file)).join(", ") || "an empty directory"}`);
  }
}

// documentTitle returns the title of a document: the title of its
// frontmatter, or its first heading.
function documentTitle(text, file) {
  if (!hasTitle(text)) return firstHeading(markdownBody(text), file).title;
  const line = frontmatter(text).match(/^["']?title["']?[ \t]*:[ \t]*(.*)$/m)[1];
  return line.replace(/^(["'])(.*)\1[ \t]*$/, "$2").trim();
}

// marks returns the text with no white space and no marks that Markdown can
// remove: "*", "_", "`", "~", and "\\".
function marks(text) {
  return text.replace(/[\s*_`~\\]/g, "");
}

// headingText returns the text of the Markdown of a heading as it reads in
// the HTML, for marks(): a link or an image keeps only its text.
function headingText(markdown) {
  return marks(markdown.replace(/!?\[([^\]]*)\]\([^)]*\)/g, "$1"));
}

// sourceHeadings returns the text of each heading of a document, for
// marks(): the ATX headings, the setext headings, and the HTML headings. The
// page shows all three kinds as <h1> to <h6>, so the check counts them alike.
function sourceHeadings(text) {
  const found = atxHeadings(markdownBody(text)).map((heading) => headingText(heading.text));
  const lines = bodyLines(text);
  // A setext heading is a paragraph that a line of "=" or "-" ends. A list
  // item, a quote, an ATX heading, or a table row is no such paragraph.
  let paragraph = [];
  for (const line of lines) {
    if (/^ {0,3}(=+|-+)[ \t]*$/.test(line) && paragraph.length > 0) {
      found.push(headingText(paragraph.join(" ")));
      paragraph = [];
    } else if (line.trim() === "" || /^ {0,3}(#{1,6}([ \t]|$)|>|[-+*][ \t]|[0-9]{1,9}[.)][ \t]|\|)/.test(line)) {
      paragraph = [];
    } else if (paragraph.length > 0 || !/^ {4}/.test(line)) {
      paragraph.push(line);
    }
  }
  const html = lines.join("\n").replace(/`[^`]*`/g, "");
  for (const match of html.matchAll(/<h([1-6])(?:\s[^>]*)?>([\s\S]*?)<\/h\1\s*>/gi)) found.push(marks(plainText(match[2])));
  return found;
}

// CTR-SITE-002: the <title> of each page starts with the title of its
// document. A page whose title comes from its first heading shows the title
// once, in one <h1>, and its body does not repeat the heading. A page
// with a title in its frontmatter keeps its first heading in the body.
for (const entry of existsSync(docs) ? readdirSync(docs, { recursive: true, withFileTypes: true }) : []) {
  if (!entry.isFile() || !entry.name.endsWith(".md")) continue;
  const file = join(entry.parentPath, entry.name);
  const text = readFileSync(file, "utf8");
  const html = page(pageURL(settings.base, relative(docs, file)));
  if (html === undefined) continue;
  // The <title> starts with the title of the document, then the site name.
  // A <title> holds text and entities, and no tags.
  const want = documentTitle(text, file);
  const pageTitle = html.match(/<title>([\s\S]*?)<\/title>/)?.[1];
  if (want === "" || !rawText(pageTitle ?? "").startsWith(want.replace(/\s+/g, ""))) {
    problems.push(`${relative(root, file)}: the <title> of the page does not start with the title ${JSON.stringify(want)}`);
  }
  const content = body(html);
  const headings = [...content.matchAll(/<h[1-6][^>]*>([\s\S]*?)<\/h[1-6]>/g)].map((m) => marks(plainText(m[1])));
  const { title, line } = firstHeading(markdownBody(text), file);
  const sources = sourceHeadings(text).filter((heading) => heading === headingText(title)).length;
  const shown = headings.filter((heading) => heading === headingText(title)).length;
  if (hasTitle(text)) {
    if (line > 0 && shown < sources) problems.push(`${relative(root, file)}: the body has ${shown} headings "${title}", want ${sources}`);
    continue;
  }
  // A requirement heading of level 1 is no title, and it stays in the body.
  const count = html.match(new RegExp(`<h1(?! id="${id}")[\\s>]`, "g"))?.length ?? 0;
  if (count !== 1) problems.push(`${relative(root, file)}: the page has ${count} <h1>, want 1`);
  if (line > 0 && headingIDs(settings.prefix, atxHeadings(markdownBody(text))[0].source).length === 0 && shown > sources - 1) {
    problems.push(`${relative(root, file)}: the body repeats the title "${title}"`);
  }
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

// body returns the Markdown content of a page with no HTML comments: from the
// content element to the pagination links of Starlight.
function body(html) {
  // The class also stands in the script of the head; the body is the div.
  const start = html.indexOf('<div class="sl-markdown-content"');
  if (start === -1) return "";
  const end = html.indexOf('class="pagination-links', start);
  return html.slice(start, end === -1 ? undefined : end).replace(/<!--[\s\S]*?-->/g, "");
}

// CTR-SITE-003: each link to a requirement ID on a page of a document goes to
// a page under base that has the anchor, and no link stands inside a code
// block.
for (const entry of existsSync(docs) ? readdirSync(docs, { recursive: true, withFileTypes: true }) : []) {
  if (!entry.isFile() || !entry.name.endsWith(".md")) continue;
  const url = pageURL(settings.base, relative(docs, join(entry.parentPath, entry.name)));
  const html = body(page(url) ?? "");
  for (const match of html.matchAll(new RegExp(`<a href="([^"#]+)#(${id})"`, "g"))) {
    const [, target, anchor] = match;
    const html = target.startsWith(settings.base) ? page(target) : undefined;
    if (html === undefined) {
      problems.push(`${url}: the link to ${anchor} goes to ${target}, which is no page under ${settings.base}`);
    } else if (!html.includes(` id="${anchor}"`)) {
      problems.push(`${url}: the link to ${anchor} goes to ${target}, which has no anchor ${anchor}`);
    }
  }
  for (const block of html.matchAll(/<pre[\s>][\s\S]*?<\/pre>/g)) {
    if (block[0].includes("<a ")) problems.push(`${url}: a code block holds a link`);
  }
}

// bodyLines returns the lines of a document outside the frontmatter, the code
// blocks, and the HTML comments. The page shows none of them.
function bodyLines(text) {
  const lines = markdownBody(text).split(/\r?\n/);
  const found = [];
  let fence = "";
  let comment = false;
  for (let i = 0; i < lines.length; i++) {
    let line = lines[i];
    if (comment) {
      const end = line.indexOf("-->");
      if (end === -1) continue;
      line = " ".repeat(end + 3) + line.slice(end + 3);
      comment = false;
    } else {
      const open = line.match(/^ {0,3}(`{3,}|~{3,})/);
      if (fence) {
        if (open && open[1][0] === fence[0] && open[1].length >= fence.length && line.trim() === open[1]) fence = "";
        continue;
      }
      if (open) {
        fence = open[1];
        continue;
      }
    }
    // A comment starts outside the code spans, and ends at the first "-->".
    // Only a comment at the start of a line (an HTML block) goes on to the
    // next lines; an unclosed "<!--" in the middle of a line is text.
    for (;;) {
      const start = line.replace(/`[^`]*`/g, (span) => " ".repeat(span.length)).indexOf("<!--");
      if (start === -1) break;
      const end = line.indexOf("-->", start + 4);
      if (end === -1) {
        if (/^ {0,3}$/.test(line.slice(0, start))) {
          line = "";
          comment = true;
        }
        break;
      }
      line = line.slice(0, start) + " ".repeat(end + 3 - start) + line.slice(end + 3);
    }
    found.push(line);
  }
  return found;
}

// sourceLinks returns the destination and the text of each inline link of a
// document that is no image, outside the frontmatter, the code blocks, and
// the code spans.
function sourceLinks(text) {
  const found = [];
  for (const line of bodyLines(text)) {
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
  return rawText(html.replace(/<[^>]*>/g, ""));
}

// rawText returns the text of an element that holds no tags, such as
// <title>, with no entities and no white space.
function rawText(html) {
  const entities = { amp: "&", lt: "<", gt: ">", quot: '"', apos: "'", nbsp: " " };
  return html
    .replace(/&(#x[0-9a-f]+|#[0-9]+|[a-z]+);/gi, (all, name) => {
      if (name[0] === "#") return String.fromCodePoint(name[1] === "x" || name[1] === "X" ? parseInt(name.slice(2), 16) : parseInt(name.slice(1), 10));
      return entities[name] ?? all;
    })
    .replace(/\s+/g, "");
}

// imagePaths returns the relative paths of the images of a document, outside
// the frontmatter, the code blocks, the code spans, and the HTML comments.
function imagePaths(text) {
  const paths = [];
  for (const line of bodyLines(text)) {
    for (const match of line.replace(/`[^`]*`/g, "").matchAll(/!\[[^\]]*\]\(<?([^)\s>]+)/g)) {
      if (!URL.canParse(match[1]) && !match[1].startsWith("/")) paths.push(match[1]);
    }
  }
  return paths;
}

// CTR-SITE-010: each relative link of a document goes to the page of the
// document and its heading, shows a diagram source in a closed collapsible
// block, or keeps only its text. A link to a missing page or heading is a
// broken link. CTR-SITE-004: each image that a document links to stands in
// its page, and the file of the image is in the output. CTR-SITE-007: each
// page has the script that opens a diagram on a click.
let documents = 0;
let images = 0;
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
  const content = body(html);
  const hrefs = new Set([...content.matchAll(/<a [^>]*href="([^"]*)"/g)].map((m) => decoded(m[1])));
  if ([...hrefs].some((href) => !URL.canParse(href) && !href.startsWith("/") && /\.puml(#.*)?$/.test(href))) {
    problems.push(`${name}: the page ${url} has a relative link to a .puml file`);
  }
  const blocks = [...content.matchAll(/<details>([\s\S]*?)<\/details>/g)].map((m) => m[1]);
  if (!html.includes(`<script>${diagramScript}</script>`)) {
    problems.push(`${name}: the page ${url} has no script that opens a diagram`);
  }
  const sources = [...content.matchAll(/<img [^>]*src="([^"]+)"/g)].map((m) => m[1]);
  for (const path of imagePaths(readFileSync(file, "utf8"))) {
    // Astro writes the file as <name>.<hash><extension>, and a raster image
    // as <name>.<hash>.webp. Images with the same bytes share one file under
    // the name of one of them. Each image takes its own <img>.
    const extension = extname(decoded(path));
    const extensions = extension.toLowerCase() === ".svg" ? escape(extension) : `(?:${escape(extension)}|\\.webp)`;
    const output = new RegExp(`^${escape(basename(decoded(path), extension))}\\.[A-Za-z0-9_-]+${extensions}$`);
    const bytes = digest(resolve(dirname(file), decoded(path)));
    let index = sources.findIndex((src) => output.test(decoded(basename(src))));
    if (index === -1 && bytes !== undefined) {
      index = sources.findIndex((src) => src.startsWith(settings.base) && digest(join(dist, decoded(src).slice(settings.base.length))) === bytes);
    }
    const source = index === -1 ? undefined : sources.splice(index, 1)[0];
    if (source === undefined) {
      problems.push(`${name}: the page ${url} shows no image ${path}`);
    } else if (!source.startsWith(settings.base) || !existsSync(join(dist, decoded(source).slice(settings.base.length)))) {
      problems.push(`${name}: the image ${path} goes to ${source}, which is no file under ${settings.base}`);
    } else {
      images++;
    }
  }
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
console.log(`check-site: ${pages.size} pages of requirements, ${documents} pages of documents, and ${images} images checked`);
