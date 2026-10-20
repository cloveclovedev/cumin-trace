package scaffold

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/cloveclovedev/cumin-trace/templates"
)

// Run writes each missing file of the templates into the repository at root,
// and writes one line for each file to stdout (CTR-TPL-001). An empty area
// says that the product has no area yet. The version is the version of the
// binary, which is a release version or not. Without a release version, it
// skips the files that name the version (CTR-CHK-005, CTR-DASH-008). It keeps every file that exists.
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
		line, err := write(file, path, values, version)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(stdout, line); err != nil {
			return err
		}
	}
	return nil
}

// write writes one file when it is missing, and returns the line that init
// prints for the file.
func write(file File, path string, values *strings.Replacer, version string) (string, error) {
	if file.NeedsVersion && version == "" {
		// A file that exists is reported as such, before the version.
		found, err := exists(path)
		if err != nil {
			return "", err
		}
		if found {
			return SkippedLine(file.Target), nil
		}
		return NoVersionLine(file.Target), nil
	}
	text, err := templates.Files.ReadFile(file.Source)
	if err != nil {
		return "", err
	}
	written, err := create(path, []byte(values.Replace(string(text))), file.Mode)
	if err != nil {
		return "", err
	}
	if written {
		return WrittenLine(file.Target), nil
	}
	return SkippedLine(file.Target), nil
}
