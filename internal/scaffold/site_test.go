package scaffold

import (
	"encoding/json"
	"regexp"
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

	for _, name := range siteFiles {
		want, err := templates.Files.ReadFile(siteSource(name))
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

// CTR-SITE-001: the site keeps the default look of Starlight. It replaces no
// component, and its one style sheet, for the blocks of CTR-SITE-006 and the
// width and the corners of CTR-SITE-014, takes every color from a variable of
// Starlight (--sl-color-*): no hex, rgb, hsl, or named color such as white or
// red.
func TestTheSiteKeepsTheLookOfStarlightAndTakesItsColorsFromItsVariables(t *testing.T) {
	config, err := templates.Files.ReadFile("site/astro.config.mjs")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(config), "components") {
		t.Error("astro.config.mjs sets components, want the default look")
	}
	if !strings.Contains(string(config), `customCss: ["./src/requirements.css"]`) {
		t.Error("astro.config.mjs does not set requirements.css as the one style sheet")
	}
	css, err := templates.Files.ReadFile("site/src/requirements.css")
	if err != nil {
		t.Fatal(err)
	}
	if colors := badColors(string(css)); len(colors) > 0 {
		t.Errorf("requirements.css has the colors %v, want the variables of Starlight", colors)
	}
}

// CTR-SITE-001: the style sheet test fails on each kind of color that is not
// a variable of Starlight, and passes on the variables of Starlight.
func TestTheColorCheckFindsEveryColorThatIsNotAVariableOfStarlight(t *testing.T) {
	for _, css := range []string{
		"a { color: #fff; }",
		"a { background: rgb(0 0 0); }",
		"a { color: hsl(0 0% 0%); }",
		"a { color: white; }",
		"a { border: 1px solid red; }",
		"a { outline: 2px dashed var(--my-color); }",
	} {
		if len(badColors(css)) == 0 {
			t.Errorf("found no color in %q", css)
		}
	}
	css := "a { border: 1px solid var(--sl-color-gray-5); border-radius: 0.5rem; color: var(--sl-color-text); margin: 0 auto; }"
	if colors := badColors(css); len(colors) > 0 {
		t.Errorf("found the colors %v in %q, want none", colors, css)
	}
}

var (
	cssComment     = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssDeclaration = regexp.MustCompile(`([a-z-]+)\s*:\s*([^;{}]+)`)
	colorCode      = regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b|rgba?\(|hsla?\(`)
	colorProperty  = regexp.MustCompile(`color|background|border|outline|shadow|fill|stroke|decoration|caret|accent`)
	starlightColor = regexp.MustCompile(`var\(--sl-color-[a-z0-9-]+\)`)
	// The words of a value that a property of color can hold beside a
	// color: lengths, numbers, and the styles of a line.
	notAColor = regexp.MustCompile(`^(-?[0-9.]+[a-z%]*|none|hidden|solid|dashed|dotted|double|groove|ridge|inset|outset)$`)
)

// badColors returns each word of a property of color in css that is not a
// variable of Starlight, a length, or the style of a line, and each hex, rgb,
// or hsl color anywhere in css.
func badColors(css string) []string {
	css = cssComment.ReplaceAllString(css, "")
	bad := colorCode.FindAllString(css, -1)
	for _, d := range cssDeclaration.FindAllStringSubmatch(css, -1) {
		if !colorProperty.MatchString(d[1]) {
			continue
		}
		for _, word := range strings.Fields(starlightColor.ReplaceAllString(d[2], " ")) {
			if !notAColor.MatchString(word) {
				bad = append(bad, d[1]+": "+word)
			}
		}
	}
	return bad
}
