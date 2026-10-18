// The site of the requirement documents, with the default look of Starlight
// (CTR-SITE-001). It builds static files only (CTR-SITE-008). The default
// Markdown processor, Sätteri, adds the anchors and the links of the
// requirement IDs (CTR-SITE-003).
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import { satteri } from "@astrojs/markdown-satteri";
import { readSettings, root } from "./src/settings.mjs";
import { requirementsPlugin } from "./src/requirements.mjs";

const settings = readSettings();

export default defineConfig({
  base: settings.base,
  markdown: {
    processor: satteri({ hastPlugins: [requirementsPlugin(settings, root)] }),
  },
  integrations: [
    starlight({
      title: settings.prefix || "Requirements",
    }),
  ],
});
