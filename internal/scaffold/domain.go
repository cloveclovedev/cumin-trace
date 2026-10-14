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
}

// Files returns the files in the order that init writes them
// (docs/ja/designs/templates/files.md). With no area, the list has no area
// page and no diagram of the area (CTR-TPL-001, CTR-TPL-003).
func Files(area string) []File {
	files := []File{
		{Target: core.SettingsFile, Source: "settings.toml", Mode: 0o644},
		{Target: core.DefaultRequirementsDir + "/index.md", Source: "docs/entry.md", Mode: 0o644},
	}
	if area != "" {
		dir := core.DefaultRequirementsDir + "/" + areaDir(area)
		files = append(files,
			File{Target: dir + "/index.md", Source: "docs/area.md", Mode: 0o644},
			File{Target: dir + "/overview.puml", Source: "docs/area-overview.puml", Mode: 0o644},
			File{Target: dir + "/overview.svg", Source: "docs/area-overview.svg", Mode: 0o644},
		)
	}
	return files
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
