// The site of the requirement documents, with the default look of Starlight
// (CTR-SITE-001). It builds static files only (CTR-SITE-008).
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import { readSettings } from "./src/settings.mjs";

const settings = readSettings();

export default defineConfig({
  base: settings.base,
  integrations: [
    starlight({
      title: settings.prefix || "Requirements",
    }),
  ],
});
