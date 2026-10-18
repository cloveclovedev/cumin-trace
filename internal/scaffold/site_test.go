package scaffold

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cloveclovedev/cumin-trace/templates"
)

// CTR-SITE-001: init writes every file of the site template into site/,
// with the bytes of the embedded text, and a leading "." on the names that
// go:embed cannot hold.
func TestWritesTheSiteTemplateWithTheEmbeddedBytes(t *testing.T) {
	root := t.TempDir()
	runInit(t, root, "PC", "TASK")

	for _, name := range []string{".node-version", ".gitignore", "package.json", "package-lock.json", "astro.config.mjs", "src/settings.mjs", "src/content.config.mjs", "src/requirements.mjs", "scripts/check-site.mjs"} {
		want, err := templates.Files.ReadFile("site/" + strings.TrimPrefix(name, "."))
		if err != nil {
			t.Fatal(err)
		}
		if got := readFile(t, root, "site/"+name); got != string(want) {
			t.Errorf("site/%s does not hold the embedded bytes", name)
		}
	}
}

// CTR-SITE-001: the site pins every version: each dependency of package.json
// is an exact version, the lock file holds the same versions, and the site
// names its Node.js version.
func TestTheSitePinsEveryVersion(t *testing.T) {
	var manifest struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	var lock struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
	}
	for name, v := range map[string]any{"site/package.json": &manifest, "site/package-lock.json": &lock} {
		text, err := templates.Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(text, v); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}

	if len(manifest.Dependencies) == 0 {
		t.Fatal("package.json has no dependencies")
	}
	for name, version := range manifest.Dependencies {
		if !releasePattern.MatchString("v" + version) {
			t.Errorf("%s has the version %q, want an exact version", name, version)
		}
		if got := lock.Packages["node_modules/"+name].Version; got != version {
			t.Errorf("the lock file has %s at %q, want %q", name, got, version)
		}
	}
	if text, _ := templates.Files.ReadFile("site/node-version"); strings.TrimSpace(string(text)) != "24" {
		t.Errorf("got the Node.js version %q, want 24", text)
	}
}

// CTR-SITE-001: the site keeps the default look of Starlight. It has no
// custom CSS and no component that replaces one of Starlight.
func TestTheSiteHasNoCustomTheme(t *testing.T) {
	config, err := templates.Files.ReadFile("site/astro.config.mjs")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"customCss", "components"} {
		if strings.Contains(string(config), key) {
			t.Errorf("astro.config.mjs sets %s, want the default look", key)
		}
	}
}
