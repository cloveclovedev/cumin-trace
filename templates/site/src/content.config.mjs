// The docs collection reads the documents under docs_dir where they are, and
// writes no copy (CTR-SITE-002, CTR-SITE-009).
import { readFileSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { defineCollection } from "astro:content";
import { glob } from "astro/loaders";
import { docsSchema } from "@astrojs/starlight/schema";
import { firstHeading } from "./requirements.mjs";
import { readSettings, root } from "./settings.mjs";

const settings = readSettings();
const base = join(root, settings.docsDir);
// A file path of the loader is relative to the project, site/, or absolute.
const project = fileURLToPath(new URL("..", import.meta.url));

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
