// The docs collection reads the documents under docs_dir where they are, and
// writes no copy (CTR-SITE-002, CTR-SITE-009).
import { readFileSync } from "node:fs";
import { basename, extname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { defineCollection } from "astro:content";
import { glob } from "astro/loaders";
import { docsSchema } from "@astrojs/starlight/schema";
import { readSettings, root } from "./settings.mjs";

const settings = readSettings();
const base = join(root, settings.docsDir);
// A file path of the loader is relative to the project, site/, or absolute.
const project = fileURLToPath(new URL("..", import.meta.url));

// firstHeading returns the text of the first ATX heading outside the
// frontmatter and the code blocks, or the file name with no extension.
export function firstHeading(text, filePath) {
  const lines = text.split(/\r?\n/);
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
    const heading = line.match(/^ {0,3}#{1,6}(?:[ \t]+(.*?))?(?:[ \t]+#+)?[ \t]*$/);
    if (heading) {
      return (heading[1] ?? "").replace(/`/g, "").trim();
    }
  }
  return basename(filePath, extname(filePath));
}

// titled wraps glob(): when the frontmatter has no title, the first heading
// of the document becomes the title, before the schema checks the data.
function titled(loader) {
  return {
    ...loader,
    load(context) {
      return loader.load({
        ...context,
        parseData(props) {
          if (props.data?.title === undefined && props.filePath) {
            const text = readFileSync(resolve(project, props.filePath), "utf8");
            props = { ...props, data: { ...props.data, title: firstHeading(text, props.filePath) } };
          }
          return context.parseData(props);
        },
      });
    },
  };
}

export const collections = {
  docs: defineCollection({
    loader: titled(glob({ base, pattern: "**/*.md" })),
    schema: docsSchema(),
  }),
};
