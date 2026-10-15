package scaffold

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/cloveclovedev/cumin-trace/templates"
)

// Run writes each missing file of the templates into the repository at root,
// and writes one line for each file to stdout (CTR-TPL-001). An empty area
// says that the product has no area yet. The version is the version of the
// binary, which is a release version or not. Without a release version, it
// skips the files that name the version (CTR-CHK-005). It keeps every file that exists.
// The returned error says that a file cannot be written; the files before it
// stay.
func Run(root, prefix, area, version string, stdout io.Writer) error {
	if err := CheckPrefix(prefix); err != nil {
		return err
	}
	if area != "" {
		if err := CheckArea(area); err != nil {
			return err
		}
	}
	version = ReleaseVersion(version)
	values := Values(prefix, area, version)
	for _, file := range Files(area) {
		path := filepath.Join(root, filepath.FromSlash(file.Target))
		if file.NeedsVersion && version == "" {
			// A file that exists is reported as such, before the version.
			found, err := exists(path)
			if err != nil {
				return err
			}
			line := NoVersionLine(file.Target)
			if found {
				line = SkippedLine(file.Target)
			}
			if _, err := fmt.Fprintln(stdout, line); err != nil {
				return err
			}
			continue
		}
		text, err := templates.Files.ReadFile(file.Source)
		if err != nil {
			return err
		}
		content := []byte(values.Replace(string(text)))
		written, err := create(path, content, file.Mode)
		if err != nil {
			return err
		}
		line := SkippedLine(file.Target)
		if written {
			line = WrittenLine(file.Target)
		}
		if _, err := fmt.Fprintln(stdout, line); err != nil {
			return err
		}
	}
	return nil
}
