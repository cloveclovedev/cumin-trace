// The hast plugin of Sätteri that gives each requirement heading an anchor
// from its ID, and turns each reference to a known ID into a link to its page
// and anchor, also across pages (CTR-SITE-003). It puts each requirement in
// one block, with the ID apart from the title (CTR-SITE-006). The places and
// the rules of a requirement and of a reference are those of the format check
// (docs/ja/designs/check/reading.md).
import { readdirSync, readFileSync } from "node:fs";
import { isAbsolute, join, relative, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { defineHastPlugin } from "satteri";

export const escape = (text) => text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

// idPattern returns the form of the IDs of the product with the prefix
// (CTR-FMT-001), with the "g" flag when global is true.
function idPattern(prefix, global) {
  const id = `${escape(prefix)}-[A-Z0-9]+-[0-9]{3}(?:\\.[1-9][0-9]*)?`;
  return global ? new RegExp(id, "g") : new RegExp(`^${id}$`);
}

// references returns the start and the end of each reference in the text
// (CTR-FMT-010). The start and the end of the text count as characters that
// are no letter, no digit, and no "-".
export function references(prefix, text) {
  const refs = [];
  for (const match of text.matchAll(idPattern(prefix, true))) {
    const from = match.index;
    const to = from + match[0].length;
    if (from > 0 && /[A-Za-z0-9-]/.test(text[from - 1])) continue;
    if (to < text.length && /[A-Za-z0-9]/.test(text[to])) continue;
    if (text[to] === "." && /[0-9]/.test(text[to + 1] ?? "")) continue;
    refs.push([from, to]);
  }
  return refs;
}

// pageURL returns the URL of the page of a document, from its path relative
// to docs_dir: the path with no ".md", and the directory for an index.md.
export function pageURL(base, path) {
  const page = path.split(sep).join("/").replace(/\.md$/, "").replace(/(^|\/)index$/, "");
  return base + (page === "" ? "" : page.toLowerCase() + "/");
}

// markdownFiles returns the Markdown files below dir, at any depth.
function markdownFiles(dir) {
  let entries;
  try {
    entries = readdirSync(dir, { withFileTypes: true, recursive: true });
  } catch {
    return [];
  }
  return entries.filter((e) => e.isFile() && e.name.endsWith(".md")).map((e) => join(e.parentPath, e.name));
}

// headingIDs returns the IDs of the requirement headings of a document: an
// ATX heading outside the frontmatter and the code blocks whose first part is
// a code span with an ID of the right form (CTR-FMT-003).
export function headingIDs(prefix, text) {
  const valid = idPattern(prefix, false);
  const lines = text.replace(/^﻿/, "").split(/\r?\n/);
  const ids = [];
  let i = 0;
  if (lines[0] === "---") {
    for (i = 1; i < lines.length && lines[i] !== "---"; i++);
    i++;
  }
  let fence = "";
  for (; i < lines.length; i++) {
    const line = lines[i];
    const open = line.match(/^ {0,3}(`{3,}|~{3,})/);
    if (fence) {
      if (open && open[1][0] === fence[0] && open[1].length >= fence.length && line.trim() === open[1]) fence = "";
      continue;
    }
    if (open) {
      fence = open[1];
      continue;
    }
    const heading = line.match(/^ {0,3}#{1,6}[ \t]+`([^`]*)`/);
    const id = heading?.[1].replace(/^ (.*) $/, "$1");
    if (id !== undefined && valid.test(id)) ids.push(id);
  }
  return ids;
}

// idTable returns the URL of the page of each requirement ID, from the
// documents under requirements_dir.
export function idTable(settings, root) {
  const docs = join(root, settings.docsDir);
  const table = new Map();
  if (!settings.prefix) return table;
  for (const file of markdownFiles(join(root, settings.requirementsDir))) {
    const url = pageURL(settings.base, relative(docs, file));
    for (const id of headingIDs(settings.prefix, readFileSync(file, "utf8"))) {
      if (!table.has(id)) table.set(id, url);
    }
  }
  return table;
}

const headingTags = ["h1", "h2", "h3", "h4", "h5", "h6"];

// requirementID returns the content of the first code span of a requirement
// heading, or undefined for any other element.
function requirementID(node, ctx, prefix) {
  if (node.type !== "element" || !headingTags.includes(node.tagName)) return undefined;
  if (ctx.parent(node)?.type !== "root") return undefined;
  const first = node.children.find((child) => child.type !== "text" || child.value.trim() !== "");
  if (first?.type !== "element" || first.tagName !== "code") return undefined;
  const content = ctx.textContent(first);
  return content.startsWith(`${prefix}-`) ? content : undefined;
}

// isStatus reports whether the element is the list item of a 状態 field: an
// item of an unordered list at the top level of the document (CTR-FMT-005).
function isStatus(node, ctx) {
  if (node.type !== "element" || node.tagName !== "li") return false;
  const list = ctx.parent(node);
  if (list?.tagName !== "ul" || ctx.parent(list)?.type !== "root") return false;
  return /^\s*状態:/.test(ctx.textContent(node));
}

// hasTitle reports whether the frontmatter of the document has a title.
export function hasTitle(text) {
  const lines = text.replace(/^\uFEFF/, "").split(/\r?\n/);
  if (lines[0] !== "---") return false;
  for (let i = 1; i < lines.length && lines[i] !== "---"; i++) {
    if (/^title[ \t]*:/.test(lines[i])) return true;
  }
  return false;
}

// headingLevel returns the level of a heading element, or 0 for any other
// node.
function headingLevel(node) {
  return node.type === "element" && headingTags.includes(node.tagName) ? Number(node.tagName[1]) : 0;
}

// blocks returns the children of the root with each requirement in a
// <section class="requirement">, from its heading to the next heading of the
// same or a higher level. A child requirement nests in its parent. A
// horizontal rule that ends a block is removed (CTR-SITE-006).
export function blocks(children, isRequirement) {
  const top = [];
  const stack = [];
  for (const child of children) {
    const level = headingLevel(child);
    if (level > 0) {
      while (stack.length > 0 && stack[stack.length - 1].level >= level) stack.pop();
    }
    const into = stack.length > 0 ? stack[stack.length - 1].section.children : top;
    if (level > 0 && isRequirement(child)) {
      const section = {
        type: "element",
        tagName: "section",
        properties: { className: ["requirement"] },
        children: [child],
      };
      into.push(section);
      stack.push({ level, section });
    } else {
      into.push(child);
    }
  }
  trimRules(top);
  return top;
}

// trimRules removes the horizontal rule at the end of each section, at any
// depth.
function trimRules(children) {
  for (const child of children) {
    if (child.type !== "element" || child.tagName !== "section") continue;
    trimRules(child.children);
    let last = child.children.length - 1;
    while (last >= 0 && child.children[last].type === "text" && child.children[last].value.trim() === "") last--;
    if (last >= 0 && child.children[last].type === "element" && child.children[last].tagName === "hr") {
      child.children.splice(last);
    }
  }
}

// inside reports whether the file is below the directory dir.
function inside(dir, file) {
  const rel = relative(dir, file);
  return rel !== "" && !rel.startsWith("..") && !isAbsolute(rel);
}

// requirementsPlugin returns the hast plugin for the settings of the site.
export function requirementsPlugin(settings, root) {
  const table = idTable(settings, root);
  const docs = join(root, settings.docsDir);
  const requirements = join(root, settings.requirementsDir);
  const valid = idPattern(settings.prefix, false);
  return defineHastPlugin({
    name: "cumin-trace-requirements",
    // The first heading that became the title of the page leaves the body,
    // so that the title shows once (CTR-SITE-002). Each requirement becomes
    // one block (CTR-SITE-006).
    after(tree, ctx) {
      if (!ctx.fileURL) return;
      const file = fileURLToPath(ctx.fileURL);
      if (!inside(docs, file)) return;
      let children = [...tree.children];
      const isRequirement = (node) =>
        settings.prefix && inside(requirements, file) && requirementID(node, ctx, settings.prefix) !== undefined;
      if (!hasTitle(readFileSync(file, "utf8"))) {
        // A requirement heading stays, with its anchor and its block.
        const first = children.findIndex((child) => headingLevel(child) > 0);
        if (first >= 0 && !isRequirement(children[first])) children.splice(first, 1);
      }
      if (settings.prefix && inside(requirements, file)) {
        children = blocks(children, isRequirement);
      }
      ctx.replaceNode(tree, { type: "root", children });
    },
    element: {
      filter: headingTags,
      visit(node, ctx) {
        if (!settings.prefix || !ctx.fileURL) return;
        if (!inside(requirements, fileURLToPath(ctx.fileURL))) return;
        const id = requirementID(node, ctx, settings.prefix);
        if (id === undefined) return;
        if (valid.test(id)) ctx.setProperty(node, "id", id);
        const code = node.children.find((child) => child.type === "element" && child.tagName === "code");
        ctx.setProperty(code, "className", ["requirement-id"]);
      },
    },
    text(node, ctx) {
      if (!settings.prefix) return;
      const refs = references(settings.prefix, node.value).filter(([from, to]) => table.has(node.value.slice(from, to)));
      if (refs.length === 0) return;
      for (let parent = ctx.parent(node); parent && parent.type !== "root"; parent = ctx.parent(parent)) {
        if (parent.type === "element" && ["a", "pre", "script", "style"].includes(parent.tagName)) return;
        if (isStatus(parent, ctx)) return;
        if (requirementID(parent, ctx, settings.prefix) !== undefined) return;
      }
      const parts = [];
      let at = 0;
      for (const [from, to] of refs) {
        if (from > at) parts.push({ type: "text", value: node.value.slice(at, from) });
        const id = node.value.slice(from, to);
        parts.push({
          type: "element",
          tagName: "a",
          properties: { href: `${table.get(id)}#${id}` },
          children: [{ type: "text", value: id }],
        });
        at = to;
      }
      if (at < node.value.length) parts.push({ type: "text", value: node.value.slice(at) });
      ctx.replaceNode(node, parts);
    },
  });
}
