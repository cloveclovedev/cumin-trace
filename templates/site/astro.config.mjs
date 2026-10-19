// The site of the requirement documents, with the default look of Starlight
// (CTR-SITE-001). It builds static files only (CTR-SITE-008). The default
// Markdown processor, Sätteri, adds the anchors and the links of the
// requirement IDs (CTR-SITE-003) and the block of each requirement
// (CTR-SITE-006). The one style sheet draws the blocks. A first plugin
// rewrites the relative links between documents (CTR-SITE-010). Astro shows
// each SVG next to a document from its relative path (CTR-SITE-004), and a
// script in the head opens a diagram on a click (CTR-SITE-007). The name at
// the top left comes from siteName() (CTR-SITE-013).
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import { satteri } from "@astrojs/markdown-satteri";
import { readSettings, root, siteName } from "./src/settings.mjs";
import { diagramScript, linksPlugin, requirementsPlugin } from "./src/requirements.mjs";

const settings = readSettings();

export default defineConfig({
  base: settings.base,
  markdown: {
    processor: satteri({ hastPlugins: [linksPlugin(settings, root), requirementsPlugin(settings, root)] }),
  },
  integrations: [
    starlight({
      title: siteName(settings, root),
      customCss: ["./src/requirements.css"],
      head: [{ tag: "script", content: diagramScript }],
    }),
  ],
});
