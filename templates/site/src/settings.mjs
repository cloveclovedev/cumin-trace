// Reads .cumin-trace.toml at the top of the repository, the parent of site/,
// and fills in the defaults (CTR-SITE-002). The format check of cumin-trace
// checks the values; the site does not.
import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { parse } from "smol-toml";

export const root = join(dirname(fileURLToPath(import.meta.url)), "..", "..");

export function readSettings() {
  const file = join(root, ".cumin-trace.toml");
  const data = existsSync(file) ? parse(readFileSync(file, "utf8")) : {};
  const site = data.site ?? {};
  return {
    prefix: data.prefix ?? "",
    requirementsDir: data.requirements_dir ?? "docs/ja/requirements",
    docsDir: site.docs_dir ?? "docs/ja",
    base: site.base ?? "/",
  };
}
