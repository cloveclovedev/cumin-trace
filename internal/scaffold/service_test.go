package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloveclovedev/cumin-trace/internal/check"
)

// runInit runs init in the directory and returns the output.
func runInit(t *testing.T, root, prefix, area string) string {
	t.Helper()
	var stdout bytes.Buffer
	if err := Run(root, prefix, area, "", &stdout); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return stdout.String()
}

// readFile returns the content of the file at a path from the top of the
// repository, with "/" as the separator.
func readFile(t *testing.T, root, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// checkPasses runs the format check and fails the test on any error.
func checkPasses(t *testing.T, root string) {
	t.Helper()
	var stdout bytes.Buffer
	count, err := check.Run(root, &stdout)
	if err != nil || count != 0 {
		t.Errorf("check: got %d errors, error %v, and output:\n%s\nwant no error", count, err, stdout.String())
	}
}

// CTR-TPL-001, CTR-TPL-003: in an empty directory, init with a prefix and an
// area writes the settings, the entry page, and the area page with its
// diagram, with the values of the arguments. The format check passes at once.
func TestWritesTheTemplatesWithAnAreaThatPassTheCheck(t *testing.T) {
	root := t.TempDir()

	output := runInit(t, root, "PC", "TASK")

	want := "written .cumin-trace.toml\n" +
		"written docs/ja/requirements/index.md\n" +
		"written docs/ja/requirements/task/index.md\n" +
		"written docs/ja/requirements/task/overview.puml\n" +
		"written docs/ja/requirements/task/overview.svg\n"
	if output != want {
		t.Errorf("got output:\n%s\nwant:\n%s", output, want)
	}
	settings := readFile(t, root, ".cumin-trace.toml")
	if !strings.Contains(settings, "prefix = \"PC\"\n") || !strings.Contains(settings, "areas = [\"TASK\"]\n") {
		t.Errorf("settings do not hold the prefix and the area:\n%s", settings)
	}
	if entry := readFile(t, root, "docs/ja/requirements/index.md"); !strings.Contains(entry, "| `TASK` | [TASK](task/index.md) |") {
		t.Errorf("the entry page has no row of the area:\n%s", entry)
	}
	if page := readFile(t, root, "docs/ja/requirements/task/index.md"); !strings.Contains(page, "### `PC-TASK-001` ") {
		t.Errorf("the area page has no example with the ID PC-TASK-001:\n%s", page)
	}
	checkPasses(t, root)
}

// CTR-TPL-001: in an empty directory, init with a prefix only writes the
// settings with an empty list of areas and the entry page, and no area page.
// The format check passes at once.
func TestWritesTheTemplatesWithoutAnAreaThatPassTheCheck(t *testing.T) {
	root := t.TempDir()

	output := runInit(t, root, "PC", "")

	want := "written .cumin-trace.toml\n" +
		"written docs/ja/requirements/index.md\n"
	if output != want {
		t.Errorf("got output:\n%s\nwant:\n%s", output, want)
	}
	settings := readFile(t, root, ".cumin-trace.toml")
	if !strings.Contains(settings, "prefix = \"PC\"\n") || !strings.Contains(settings, "areas = []\n") {
		t.Errorf("settings do not hold the prefix and an empty list of areas:\n%s", settings)
	}
	entries, err := os.ReadDir(filepath.Join(root, "docs", "ja", "requirements"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "index.md" {
		t.Errorf("got %d entries in the requirements directory, want index.md only", len(entries))
	}
	checkPasses(t, root)
}

// CTR-TPL-003: the area page has the headings 全体像, 要件, and 未決事項, a
// place for a diagram, and one example requirement with a 検証 line.
func TestTheAreaPageHasTheHeadingsAndOneExample(t *testing.T) {
	root := t.TempDir()
	runInit(t, root, "PC", "TASK")

	page := readFile(t, root, "docs/ja/requirements/task/index.md")

	var headings []string
	for _, line := range strings.Split(page, "\n") {
		if strings.HasPrefix(line, "## ") {
			headings = append(headings, strings.TrimPrefix(line, "## "))
		}
	}
	if strings.Join(headings, ",") != "全体像,要件,未決事項" {
		t.Errorf("got level 2 headings %q, want 全体像, 要件, 未決事項", headings)
	}
	if !strings.Contains(page, "![カテゴリの全体像](overview.svg)") {
		t.Errorf("the area page shows no diagram:\n%s", page)
	}
	if strings.Count(page, "### `") != 1 || strings.Count(page, "- 検証: `test` ") != 1 {
		t.Errorf("the area page does not hold exactly one requirement with one 検証 line:\n%s", page)
	}
}

// CTR-TPL-001: a file that exists before init keeps its bytes, and init
// reports it as skipped. init writes the other files.
func TestKeepsAFileThatExists(t *testing.T) {
	root := t.TempDir()
	existing := "prefix = \"OWN\"\nareas = [\"MINE\"]\n"
	if err := os.WriteFile(filepath.Join(root, ".cumin-trace.toml"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	output := runInit(t, root, "PC", "TASK")

	if !strings.HasPrefix(output, "skipped .cumin-trace.toml: already exists\nwritten docs/ja/requirements/index.md\n") {
		t.Errorf("got output:\n%s\nwant the settings skipped and the next file written", output)
	}
	if got := readFile(t, root, ".cumin-trace.toml"); got != existing {
		t.Errorf("got settings %q, want the bytes before init %q", got, existing)
	}

	again := runInit(t, root, "PC", "TASK")

	if strings.Contains(again, "written") || strings.Count(again, "skipped") != len(Files("TASK")) {
		t.Errorf("a second run got output:\n%s\nwant every file skipped", again)
	}
}

// CTR-TPL-001: init refuses a prefix or an area that the settings cannot
// hold, and writes no file.
func TestRefusesAWrongPrefixOrArea(t *testing.T) {
	tests := []struct {
		name, prefix, area, want string
	}{
		{"a prefix in lower case", "pc", "TASK", `invalid prefix "pc"`},
		{"an empty prefix", "", "", `invalid prefix ""`},
		{"an area with a hyphen", "PC", "TA-SK", `invalid area "TA-SK"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			var stdout bytes.Buffer

			err := Run(root, test.prefix, test.area, "v0.1.0", &stdout)

			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Errorf("got error %v, want an error that holds %q", err, test.want)
			}
			if entries, _ := os.ReadDir(root); len(entries) != 0 || stdout.Len() != 0 {
				t.Errorf("got %d files and output %q, want none", len(entries), stdout.String())
			}
		})
	}
}

// CTR-CHK-005: only the version of a release counts as a version. The
// workflow names the release to download by this version.
func TestTakesOnlyTheVersionOfARelease(t *testing.T) {
	tests := map[string]string{
		"v0.1.0":                               "v0.1.0",
		"v12.3.45":                             "v12.3.45",
		"":                                     "",
		"(devel)":                              "",
		"v0.1.1-0.20261014120000-2c029c5abcde": "",
		"v0.1.0+dirty":                         "",
		"0.1.0":                                "",
	}
	for version, want := range tests {
		if got := ReleaseVersion(version); got != want {
			t.Errorf("ReleaseVersion(%q) = %q, want %q", version, got, want)
		}
	}
}
