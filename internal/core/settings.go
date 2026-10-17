// Package core holds what every feature shares and what knows no provider.
package core

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// SettingsFile is the name of the settings file at the top of a repository.
const SettingsFile = ".cumin-trace.toml"

// DefaultRequirementsDir is the directory of the requirement documents when
// the settings do not name one (CTR-FMT-001).
const DefaultRequirementsDir = "docs/ja/requirements"

// DefaultRequirementLabel is the label of a requirement issue when the
// settings do not name one (CTR-TRC-001).
const DefaultRequirementLabel = "cumin/type/requirement"

// Settings is the content of the settings file (CTR-FMT-011).
type Settings struct {
	// Prefix is the prefix of the product, the first part of every ID.
	Prefix string
	// Areas are the allowed areas, in the order of the file.
	Areas []string
	// RequirementsDir is the directory of the requirement documents,
	// relative to the top of the repository, with "/" as the separator.
	RequirementsDir string
	// RequirementLabel is the label that marks a requirement issue. The
	// format check does not check it; the trace does (CTR-TRC-001).
	RequirementLabel string
}

var namePattern = regexp.MustCompile(`\A[A-Z0-9]+\z`)

// IsName says whether the text is a valid prefix or area: one or more upper
// case letters and digits (CTR-FMT-011).
func IsName(text string) bool {
	return namePattern.MatchString(text)
}

// LoadSettings reads the settings file at the top of the repository root and
// checks its values. An unknown key is an error, so that a misspelled key
// does not go unseen.
func LoadSettings(root string) (Settings, error) {
	var file struct {
		Prefix           string   `toml:"prefix"`
		Areas            []string `toml:"areas"`
		RequirementsDir  string   `toml:"requirements_dir"`
		RequirementLabel string   `toml:"requirement_label"`
	}
	meta, err := toml.DecodeFile(filepath.Join(root, SettingsFile), &file)
	if err != nil {
		var parseErr toml.ParseError
		if errors.As(err, &parseErr) {
			return Settings{}, fmt.Errorf("%s:%d: %s", SettingsFile, parseErr.Position.Line, parseErr.Message)
		}
		if errors.Is(err, os.ErrNotExist) {
			return Settings{}, fmt.Errorf("%s: no such file", SettingsFile)
		}
		return Settings{}, fmt.Errorf("%s: %s", SettingsFile, strings.TrimPrefix(err.Error(), "toml: "))
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return Settings{}, fmt.Errorf("%s: unknown key %s", SettingsFile, undecoded[0])
	}

	if !meta.IsDefined("prefix") {
		return Settings{}, fmt.Errorf("%s: prefix is missing", SettingsFile)
	}
	if !IsName(file.Prefix) {
		return Settings{}, fmt.Errorf("%s: prefix %q must be upper case letters and digits", SettingsFile, file.Prefix)
	}

	// An empty list says that the product has no area yet. A missing key
	// is an error, so that it is not taken for an empty list.
	if !meta.IsDefined("areas") {
		return Settings{}, fmt.Errorf("%s: areas is missing", SettingsFile)
	}
	seen := map[string]bool{}
	for _, area := range file.Areas {
		if !IsName(area) {
			return Settings{}, fmt.Errorf("%s: area %q must be upper case letters and digits", SettingsFile, area)
		}
		if seen[area] {
			return Settings{}, fmt.Errorf("%s: area %q is named twice", SettingsFile, area)
		}
		seen[area] = true
	}

	dir := DefaultRequirementsDir
	if meta.IsDefined("requirements_dir") {
		dir = file.RequirementsDir
		if dir == "" || path.IsAbs(dir) || filepath.IsAbs(dir) || strings.Contains(dir, `\`) {
			return Settings{}, fmt.Errorf("%s: requirements_dir %q must be a relative path with / as the separator", SettingsFile, dir)
		}
		for _, element := range strings.Split(dir, "/") {
			if element == ".." {
				return Settings{}, fmt.Errorf("%s: requirements_dir %q must not contain ..", SettingsFile, dir)
			}
		}
		dir = path.Clean(dir)
	}

	label := DefaultRequirementLabel
	if meta.IsDefined("requirement_label") {
		label = file.RequirementLabel
	}

	return Settings{Prefix: file.Prefix, Areas: file.Areas, RequirementsDir: dir, RequirementLabel: label}, nil
}
