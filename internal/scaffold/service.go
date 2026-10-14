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
// binary, which is a release version or not. It keeps every file that exists.
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
	values := Values(prefix, area, ReleaseVersion(version))
	for _, file := range Files(area) {
		text, err := templates.Files.ReadFile(file.Source)
		if err != nil {
			return err
		}
		content := []byte(values.Replace(string(text)))
		written, err := create(filepath.Join(root, filepath.FromSlash(file.Target)), content, file.Mode)
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
