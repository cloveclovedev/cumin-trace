// Package scaffold writes the templates of the requirement documents into a
// product repository (cumin-trace init).
package scaffold

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/cloveclovedev/cumin-trace/internal/core"
)

// File is one file that init writes.
type File struct {
	// Target is the path in the repository, with "/" as the separator.
	Target string
	// Source is the path of the embedded text in the package templates.
	Source string
	// Mode is the permission of a new file.
	Mode fs.FileMode
	// NeedsVersion says that the text names the release version of the
	// binary. init writes the file only when the binary has one.
	NeedsVersion bool
}

// Files returns the files in the order that init writes them
// (docs/ja/designs/templates/files.md). With no area, the list has no area
// page and no diagram of the area (CTR-TPL-001, CTR-TPL-003).
func Files(area string) []File {
	files := []File{
		{Target: core.SettingsFile, Source: "settings.toml", Mode: 0o644},
		{Target: core.DefaultRequirementsDir + "/index.md", Source: "docs/entry.md", Mode: 0o644},
		{Target: core.DefaultRequirementsDir + "/overview.md", Source: "docs/overview.md", Mode: 0o644},
		{Target: core.DefaultRequirementsDir + "/technology.md", Source: "docs/technology.md", Mode: 0o644},
		{Target: core.DefaultRequirementsDir + "/glossary.md", Source: "docs/glossary.md", Mode: 0o644},
		{Target: core.DefaultRequirementsDir + "/backlog.md", Source: "docs/backlog.md", Mode: 0o644},
	}
	if area != "" {
		dir := core.DefaultRequirementsDir + "/" + areaDir(area)
		files = append(files,
			File{Target: dir + "/index.md", Source: "docs/area.md", Mode: 0o644},
			File{Target: dir + "/overview.puml", Source: "docs/area-overview.puml", Mode: 0o644},
			File{Target: dir + "/overview.svg", Source: "docs/area-overview.svg", Mode: 0o644},
		)
	}
	// The source code structure is outside the requirement documents (CTR-TPL-009).
	files = append(files, File{Target: "docs/ja/architecture/overview.md", Source: "docs/architecture.md", Mode: 0o644})
	// The skill holds the nine templates, written from the same embedded text
	// as the documents (CTR-TPL-005, CTR-TPL-008).
	files = append(files, File{Target: SkillDir + "/SKILL.md", Source: "skill/SKILL.md", Mode: 0o644})
	for _, name := range skillTemplates {
		files = append(files, File{Target: SkillDir + "/templates/" + name, Source: "docs/" + name, Mode: 0o644})
	}
	// The script that renders the diagrams, and its image, as in cumin-works
	// (CTR-TPL-006).
	files = append(files,
		File{Target: RenderScript, Source: RenderScript, Mode: 0o755},
		File{Target: "tools/plantuml/Dockerfile", Source: "tools/plantuml/Dockerfile", Mode: 0o644},
	)
	// The template of the Starlight site, with pinned versions and a lock file
	// (CTR-SITE-001, CTR-SITE-005).
	for _, name := range siteFiles {
		files = append(files, File{Target: "site/" + name, Source: siteSource(name), Mode: 0o644})
	}
	// The workflow that runs the format check on pull requests, with the
	// release binary of this version (CTR-CHK-005).
	files = append(files, File{Target: CheckWorkflow, Source: "workflows/cumin-trace-check.yml", Mode: 0o644, NeedsVersion: true})
	// The example workflow that rebuilds the site with the trace, with the
	// release binary of this version (CTR-DASH-008).
	files = append(files, File{Target: SiteWorkflow, Source: "workflows/cumin-trace-site.yml", Mode: 0o644, NeedsVersion: true})
	return files
}

// SkillDir is the directory of the skill for writing documents (CTR-TPL-005).
const SkillDir = ".claude/skills/cumin-trace-docs"

// RenderScript is the script that renders the diagrams to SVG (CTR-TPL-006).
const RenderScript = "scripts/render-diagrams.sh"

// CheckWorkflow is the workflow that runs the format check on pull requests
// (CTR-CHK-005).
const CheckWorkflow = ".github/workflows/cumin-trace-check.yml"

// SiteWorkflow is the example workflow that rebuilds the site with the
// trace (CTR-DASH-008).
const SiteWorkflow = ".github/workflows/cumin-trace-site.yml"

// skillTemplates names the templates of the skill, one for each kind of
// document, in the order that init writes them. The seven kinds of
// CTR-TPL-001 come first, then the entry and the detail of a design document
// (CTR-TPL-008).
var skillTemplates = []string{
	"entry.md",
	"overview.md",
	"technology.md",
	"glossary.md",
	"backlog.md",
	"area.md",
	"architecture.md",
	"design-index.md",
	"design-detail.md",
}

// siteFiles names the files of the site, in the order that init writes
// them (docs/ja/designs/site/files.md).
var siteFiles = []string{
	"package.json",
	"package-lock.json",
	".node-version",
	".gitignore",
	"astro.config.mjs",
	"src/settings.mjs",
	"src/content.config.mjs",
	"src/requirements.mjs",
	"src/requirements.css",
	"src/dashboard.mjs",
	"src/pages/dashboard.astro",
	"scripts/check-site.mjs",
}

// siteSource returns the embedded path of a file of the site. The embedded
// name has no leading ".", because go:embed skips such a file in a directory.
func siteSource(name string) string {
	return "site/" + strings.TrimPrefix(name, ".")
}

// CheckPrefix returns an error when the prefix is not a valid prefix of
// .cumin-trace.toml (CTR-FMT-011).
func CheckPrefix(prefix string) error {
	if !core.IsName(prefix) {
		return fmt.Errorf("invalid prefix %q: want upper-case letters and digits", prefix)
	}
	return nil
}

// CheckArea returns an error when the area is not a valid area of
// .cumin-trace.toml. An empty area is an error too: the caller passes an
// area only when the arguments hold one.
func CheckArea(area string) error {
	if !core.IsName(area) {
		return fmt.Errorf("invalid area %q: want upper-case letters and digits", area)
	}
	return nil
}

var releasePattern = regexp.MustCompile(`\Av[0-9]+\.[0-9]+\.[0-9]+\z`)

// ReleaseVersion returns the version of the binary when it is the version of
// a release, v<number>.<number>.<number>. It returns "" for any other value:
// empty, (devel), a pseudo-version, or a build with changes.
func ReleaseVersion(version string) string {
	if releasePattern.MatchString(version) {
		return version
	}
	return ""
}

// Values returns the replacer that puts the values of the arguments and the
// release version into the embedded text. An empty area says that the
// product has no area yet.
func Values(prefix, area, version string) *strings.Replacer {
	areas, areaRows, areaName := "[]", "", "<AREA>"
	if area != "" {
		areas = fmt.Sprintf("[%q]", area)
		areaRows = fmt.Sprintf("| `%s` | [%s](%s/index.md) | <!-- このカテゴリで決めること --> |\n", area, area, areaDir(area))
		areaName = area
	}
	return strings.NewReplacer(
		"{{prefix}}", prefix,
		"{{area}}", areaName,
		"{{area_dir}}", areaDir(area),
		"{{areas}}", areas,
		"{{area_rows}}", areaRows,
		"{{version}}", version,
	)
}

// areaDir returns the name of the directory of the area.
func areaDir(area string) string {
	return strings.ToLower(area)
}

// WrittenLine returns the line that init prints for a file that it wrote.
func WrittenLine(target string) string {
	return "written " + target
}

// SkippedLine returns the line that init prints for a file that exists.
func SkippedLine(target string) string {
	return "skipped " + target + ": already exists"
}

// NoVersionLine returns the line that init prints for a file that names the
// release version, when the binary has no release version.
func NoVersionLine(target string) string {
	return "skipped " + target + ": this cumin-trace has no version; use a release binary"
}
