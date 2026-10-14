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
		"written docs/ja/requirements/overview.md\n" +
		"written docs/ja/requirements/technology.md\n" +
		"written docs/ja/requirements/glossary.md\n" +
		"written docs/ja/requirements/backlog.md\n" +
		"written docs/ja/requirements/task/index.md\n" +
		"written docs/ja/requirements/task/overview.puml\n" +
		"written docs/ja/requirements/task/overview.svg\n" +
		"written docs/ja/architecture/overview.md\n"
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
// settings with an empty list of areas and the other pages, and no area page.
// The format check passes at once.
func TestWritesTheTemplatesWithoutAnAreaThatPassTheCheck(t *testing.T) {
	root := t.TempDir()

	output := runInit(t, root, "PC", "")

	want := "written .cumin-trace.toml\n" +
		"written docs/ja/requirements/index.md\n" +
		"written docs/ja/requirements/overview.md\n" +
		"written docs/ja/requirements/technology.md\n" +
		"written docs/ja/requirements/glossary.md\n" +
		"written docs/ja/requirements/backlog.md\n" +
		"written docs/ja/architecture/overview.md\n"
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
	for _, entry := range entries {
		if entry.IsDir() {
			t.Errorf("got the directory %s in the requirements directory, want no area page", entry.Name())
		}
	}
	checkPasses(t, root)
}

// headings returns the level 2 headings of the page, in order.
func headings(page string) []string {
	var found []string
	for _, line := range strings.Split(page, "\n") {
		if strings.HasPrefix(line, "## ") {
			found = append(found, strings.TrimPrefix(line, "## "))
		}
	}
	return found
}

// CTR-TPL-002, CTR-TPL-004, CTR-TPL-007, CTR-TPL-009: each page has exactly
// the headings and the table columns of its requirement.
func TestEachPageHasTheHeadingsAndColumnsOfItsRequirement(t *testing.T) {
	root := t.TempDir()
	runInit(t, root, "PC", "TASK")

	tests := []struct {
		page, headings, header string
	}{
		{"docs/ja/requirements/overview.md", "解決したい課題,目的,指標,登場人物,範囲外,フェーズ", ""},
		{"docs/ja/requirements/technology.md", "構成,使う技術,ライブラリの方針,決めていないこと", "| 要素 | 技術 | 選んだ理由 |"},
		{"docs/ja/requirements/glossary.md", "", "| 語 | 意味 | 使わない言い方 | 英語 |"},
		{"docs/ja/architecture/overview.md", "", "| パッケージ | 役割 | 設計 |"},
	}
	for _, test := range tests {
		t.Run(test.page, func(t *testing.T) {
			page := readFile(t, root, test.page)

			if got := strings.Join(headings(page), ","); got != test.headings {
				t.Errorf("got level 2 headings %q, want %q", got, test.headings)
			}
			var tableHeaders []string
			for _, line := range strings.Split(page, "\n") {
				if strings.HasPrefix(line, "| ") {
					tableHeaders = append(tableHeaders, line)
				}
			}
			want := []string{}
			if test.header != "" {
				want = []string{test.header}
			}
			if strings.Join(tableHeaders, "\n") != strings.Join(want, "\n") {
				t.Errorf("got table lines %q, want %q", tableHeaders, want)
			}
		})
	}
}

// CTR-TPL-004, CTR-TPL-007: the technology page and the glossary hold no
// requirement ID.
func TestTheTechnologyPageAndTheGlossaryHoldNoID(t *testing.T) {
	root := t.TempDir()
	runInit(t, root, "PC", "TASK")

	for _, name := range []string{"docs/ja/requirements/technology.md", "docs/ja/requirements/glossary.md"} {
		if page := readFile(t, root, name); strings.Contains(page, "PC-") || strings.Contains(page, "{{") {
			t.Errorf("%s holds an ID or a value to fill:\n%s", name, page)
		}
	}
}

// CTR-TPL-001: the entry page links to the overview, the technology page,
// the glossary, the backlog, and the source code structure.
func TestTheEntryPageLinksToThePages(t *testing.T) {
	root := t.TempDir()
	runInit(t, root, "PC", "")

	entry := readFile(t, root, "docs/ja/requirements/index.md")

	for _, link := range []string{"(overview.md)", "(technology.md)", "(glossary.md)", "(backlog.md)", "(../architecture/overview.md)"} {
		if !strings.Contains(entry, link) {
			t.Errorf("the entry page has no link %s:\n%s", link, entry)
		}
	}
}

// CTR-TPL-003: the area page has the headings 全体像, 要件, and 未決事項, a
// place for a diagram, and one example requirement with a 検証 line.
func TestTheAreaPageHasTheHeadingsAndOneExample(t *testing.T) {
	root := t.TempDir()
	runInit(t, root, "PC", "TASK")

	page := readFile(t, root, "docs/ja/requirements/task/index.md")

	if got := headings(page); strings.Join(got, ",") != "全体像,要件,未決事項" {
		t.Errorf("got level 2 headings %q, want 全体像, 要件, 未決事項", got)
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
