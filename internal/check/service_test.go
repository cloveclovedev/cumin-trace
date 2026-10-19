package check

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const settings = `prefix = "CTR"
areas = ["FMT", "CHK"]
`

// writeRepository writes the files into a new temporary directory and returns
// it. A key is a path from the top of the repository, with "/" as the separator.
func writeRepository(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// runCheck runs the check on the repository and returns the number of the
// errors and the output.
func runCheck(t *testing.T, root string) (int, string) {
	t.Helper()
	var stdout bytes.Buffer
	count, err := Run(root, &stdout)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return count, stdout.String()
}

// CTR-FMT-001: a document needs no frontmatter, and the default directory is
// docs/ja/requirements/.
func TestReadsDocumentWithoutFrontmatterInTheDefaultDirectory(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/format/index.md": "# Format\n\n" +
			"### `CTR-FMT-001` A good requirement\n- 検証: `test` It works\n" +
			"### `CTR-OTHER-001` A requirement that proves the reading\n- 検証: `test` It works\n",
		"docs/elsewhere/index.md": "### `CTR-NOWHERE-001` Outside the directory\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/format/index.md:5: CTR-OTHER-001: area OTHER is not in .cumin-trace.toml\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-001: the settings name the directory of the requirement documents.
func TestReadsTheDirectoryThatTheSettingsName(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml":              settings + "requirements_dir = \"requirements\"\n[site]\ndocs_dir = \".\"\n",
		"requirements/deep/down/page.md": "## `CTR-OTHER-001` In the named directory\n- 検証: `test` It works\n",
		"docs/ja/requirements/index.md":  "## `CTR-NOWHERE-001` In the default directory\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "requirements/deep/down/page.md:1: CTR-OTHER-001: area OTHER is not in .cumin-trace.toml\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-002, CTR-CHK-001: an ID with a wrong form is an error. The document
// holds only this error.
func TestReportsAnIDWithAWrongForm(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "# Requirements\n\n" +
			"### `CTR-FMT-001` A good requirement\n- 検証: `test` It works\n" +
			"#### `CTR-FMT-001.1` A good child\n- 検証: `test` It works\n" +
			"### `CTR-FMT-02` Two digits\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:7: CTR-FMT-02: invalid ID format: want <prefix>-<area>-<3 digits>, or that with .<child number>\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-011, CTR-CHK-001: an ID whose area is not in the settings is an
// error. The document holds only this error.
func TestReportsAnAreaThatIsNotInTheSettings(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "# Requirements\n\n" +
			"### `CTR-CHK-001` A good requirement\n- 検証: `test` It works\n" +
			"### `CTR-SITE-001` An area outside the settings\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:5: CTR-SITE-001: area SITE is not in .cumin-trace.toml\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-CHK-002: the check does not stop at the first error. Each line holds
// file, line, ID, and message.
func TestReportsBothErrorsOfADocumentWithTwoErrors(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "### `CTR-FMT-1` One digit\n\n" +
			"text\n\n" +
			"### `CTR-SITE-001` An area outside the settings\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:1: CTR-FMT-1: invalid ID format: want <prefix>-<area>-<3 digits>, or that with .<child number>\n" +
		"docs/ja/requirements/index.md:5: CTR-SITE-001: area SITE is not in .cumin-trace.toml\n"
	if count != 2 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 2 errors and output:\n%s", count, output, want)
	}
}

// CTR-FMT-003, CTR-FMT-001: only a heading that starts with an ID of the
// product in a code span is a requirement. A heading in a code block or in
// the frontmatter is not read.
func TestReadsNoHeadingInACodeBlockOrInTheFrontmatter(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "---\n" +
			"# `CTR-BAD` A comment of the frontmatter\n" +
			"---\n\n" +
			"```markdown\n" +
			"### `CTR-BAD` In a fenced code block\n" +
			"```\n\n" +
			"    ### `CTR-BAD` In an indented code block\n\n" +
			"- A list item\n\n" +
			"  ~~~\n" +
			"  ### `CTR-BAD` In a fenced code block of a list item\n" +
			"  ~~~\n\n" +
			"### `PC-TASK-1` An ID of another product\n\n" +
			"### CTR-BAD An ID outside a code span\n\n" +
			"### Grouping heading\n\n" +
			"### `CTR-BAD` After everything\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:23: CTR-BAD: invalid ID format: want <prefix>-<area>-<3 digits>, or that with .<child number>\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-CHK-002 and the principle of fixed rules: the same input gives the same
// output, in the order of file and line.
func TestGivesTheSameOutputForTheSameInput(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml":               settings,
		"docs/ja/requirements/b/index.md": "## `CTR-X-001` One\n- 検証: `test` It works\n## `CTR-FMT-1` Two\n",
		"docs/ja/requirements/a/index.md": "## `CTR-Y-001` One\n- 検証: `test` It works\n",
		"docs/ja/requirements/a-b.md":     "## `CTR-Z-001` One\n- 検証: `test` It works\n",
	})

	_, first := runCheck(t, root)
	_, second := runCheck(t, root)

	want := "docs/ja/requirements/a-b.md:1: CTR-Z-001: area Z is not in .cumin-trace.toml\n" +
		"docs/ja/requirements/a/index.md:1: CTR-Y-001: area Y is not in .cumin-trace.toml\n" +
		"docs/ja/requirements/b/index.md:1: CTR-X-001: area X is not in .cumin-trace.toml\n" +
		"docs/ja/requirements/b/index.md:3: CTR-FMT-1: invalid ID format: want <prefix>-<area>-<3 digits>, or that with .<child number>\n"
	if first != want {
		t.Errorf("got output:\n%s\nwant output:\n%s", first, want)
	}
	if second != first {
		t.Errorf("the second run gives another output:\n%s\nfirst output:\n%s", second, first)
	}
}

// CTR-CHK-004: the check passes with no token and no fake GitHub.
func TestPassesWithNoTokenAndNoGitHub(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml":             settings,
		"docs/ja/requirements/index.md": "### `CTR-FMT-001` A good requirement\n\n- 検証: `test` It works\n\n---\n",
	})

	count, output := runCheck(t, root)

	if count != 0 || output != "" {
		t.Errorf("got %d errors and output:\n%s\nwant no error and no output", count, output)
	}
}

// CTR-CHK-004: the packages of the check import no HTTP client.
func TestImportsNoHTTPClient(t *testing.T) {
	for _, dir := range []string{".", "../requirement", "../core"} {
		packages, err := parser.ParseDir(token.NewFileSet(), dir, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, pkg := range packages {
			for name, file := range pkg.Files {
				for _, spec := range file.Imports {
					if strings.Trim(spec.Path.Value, `"`) == "net/http" {
						t.Errorf("%s imports net/http", name)
					}
				}
			}
		}
	}
}

// CTR-FMT-011: the check cannot run without right settings, and says why.
func TestRefusesWrongSettings(t *testing.T) {
	document := "### `CTR-FMT-001` A good requirement\n- 検証: `test` It works\n"
	tests := []struct {
		name     string
		settings string
		want     string
	}{
		{"a syntax error", "prefix = \"CTR\"\nareas = oops\n", ".cumin-trace.toml:2:"},
		{"no prefix", "areas = [\"FMT\"]\n", "prefix is missing"},
		{"a prefix in lower case", "prefix = \"ctr\"\nareas = [\"FMT\"]\n", "prefix \"ctr\""},
		{"no areas key", "prefix = \"CTR\"\n", "areas is missing"},
		{"an area named twice", "prefix = \"CTR\"\nareas = [\"FMT\", \"FMT\"]\n", "named twice"},
		{"an unknown key", settings + "area = \"FMT\"\n", "unknown key area"},
		{"a directory above the repository", settings + "requirements_dir = \"../docs\"\n", "must not contain .."},
		{"a directory that does not exist", settings + "requirements_dir = \"docs/ja/nowhere\"\n", "nowhere: no such directory"},
		{"a requirements_dir outside the default docs_dir", settings + "requirements_dir = \"spec/requirements\"\n", "must hold requirements_dir"},
		{"an unknown key in site", settings + "[site]\nbse = \"/x/\"\n", "unknown key site.bse"},
		{"a docs_dir that does not hold requirements_dir", settings + "[site]\ndocs_dir = \"docs/en\"\n", "must hold requirements_dir"},
		{"a base without a leading /", settings + "[site]\nbase = \"x/\"\n", "must start and end with /"},
		{"a base without a trailing /", settings + "[site]\nbase = \"/x\"\n", "must start and end with /"},
		{"a base with an empty segment", settings + "[site]\nbase = \"//\"\n", "must not contain an empty"},
		{"a base with a .. segment", settings + "[site]\nbase = \"/../\"\n", "must not contain an empty"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := writeRepository(t, map[string]string{
				".cumin-trace.toml":             test.settings,
				"docs/ja/requirements/index.md": document,
			})

			_, err := Run(root, &bytes.Buffer{})

			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Errorf("got error %v, want an error that holds %q", err, test.want)
			}
		})
	}
}

// CTR-SITE-002: a [site] table that follows the rules passes the check.
func TestAcceptsASiteTableThatFollowsTheRules(t *testing.T) {
	tests := []struct {
		name string
		site string
	}{
		{"the default directory and a base", "[site]\ndocs_dir = \"docs/ja\"\nbase = \"/cumin-trace/\"\n"},
		{"the top of the repository", "[site]\ndocs_dir = \".\"\n"},
		{"the directory of the requirements", "[site]\ndocs_dir = \"docs/ja/requirements\"\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := writeRepository(t, map[string]string{
				".cumin-trace.toml":             settings + test.site,
				"docs/ja/requirements/index.md": "# Requirements\n",
			})

			count, output := runCheck(t, root)

			if count != 0 || output != "" {
				t.Errorf("got %d errors and output:\n%s\nwant no error", count, output)
			}
		})
	}
}

// CTR-FMT-011: an empty list of areas says that the product has no area yet.
// Documents without requirements pass, and a requirement of any area is an
// error.
func TestAcceptsAnEmptyListOfAreas(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml":                    "prefix = \"CTR\"\nareas = []\n",
		"docs/ja/requirements/index.md":        "# Requirements\n\nNo requirement yet.\n",
		"docs/ja/requirements/format/index.md": "### `CTR-FMT-001` A requirement\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/format/index.md:1: CTR-FMT-001: area FMT is not in .cumin-trace.toml\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-003: requirements at different heading levels are both read, and a
// horizontal rule changes nothing. The child after a horizontal rule stays
// under its parent, and the requirement after the child is under no parent.
func TestReadsRequirementsAtDifferentLevelsWithAndWithoutHorizontalRules(t *testing.T) {
	withRules := "# Requirements\n\n" +
		"## `CTR-X-001` At level 2\n\n" +
		"text\n\n" +
		"---\n\n" +
		"### `CTR-X-001.1` A child after a horizontal rule\n- 検証: `test` It works\n" +
		"---\n\n" +
		"#### `CTR-Y-001` At level 4\n- 検証: `test` It works\n" +
		"---\n\n" +
		"## `CTR-FMT-001` After the range of the first requirement\n- 検証: `test` It works\n" +
		"---\n"
	withoutRules := strings.ReplaceAll(withRules, "---\n", "\n")
	want := "docs/ja/requirements/index.md:3: CTR-X-001: area X is not in .cumin-trace.toml\n" +
		"docs/ja/requirements/index.md:9: CTR-X-001.1: area X is not in .cumin-trace.toml\n" +
		"docs/ja/requirements/index.md:13: CTR-Y-001: area Y is not in .cumin-trace.toml\n"
	for name, document := range map[string]string{"with horizontal rules": withRules, "without horizontal rules": withoutRules} {
		t.Run(name, func(t *testing.T) {
			root := writeRepository(t, map[string]string{
				".cumin-trace.toml":             settings,
				"docs/ja/requirements/index.md": document,
			})

			count, output := runCheck(t, root)

			if count != 3 || output != want {
				t.Errorf("got %d errors and output:\n%s\nwant 3 errors and output:\n%s", count, output, want)
			}
		})
	}
}

// CTR-FMT-003, CTR-CHK-001: the same ID on two pages is an error that names
// both places. The documents hold only this error.
func TestReportsTheSameIDOnTwoPages(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/check/index.md": "# Check\n\n" +
			"### `CTR-CHK-001` A good requirement\n- 検証: `test` It works\n" +
			"### `CTR-FMT-002` The same ID as on the other page\n- 検証: `test` It works\n",
		"docs/ja/requirements/format/index.md": "# Format\n\n" +
			"### `CTR-FMT-001` A good requirement\n- 検証: `test` It works\n" +
			"---\n\n" +
			"### `CTR-FMT-002` The same ID as on the other page\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/check/index.md:5: CTR-FMT-002: duplicate ID: also at docs/ja/requirements/format/index.md:7\n" +
		"docs/ja/requirements/format/index.md:7: CTR-FMT-002: duplicate ID: also at docs/ja/requirements/check/index.md:5\n"
	if count != 2 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 2 errors and output:\n%s", count, output, want)
	}
}

// CTR-FMT-003: an ID in three places names the two other places on each line.
func TestNamesEveryOtherPlaceOfAnIDInThreePlaces(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "## `CTR-FMT-001` One\n- 検証: `test` It works\n" +
			"## `CTR-FMT-001` Two\n- 検証: `test` It works\n" +
			"## `CTR-FMT-001` Three\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:1: CTR-FMT-001: duplicate ID: also at docs/ja/requirements/index.md:3, docs/ja/requirements/index.md:5\n" +
		"docs/ja/requirements/index.md:3: CTR-FMT-001: duplicate ID: also at docs/ja/requirements/index.md:1, docs/ja/requirements/index.md:5\n" +
		"docs/ja/requirements/index.md:5: CTR-FMT-001: duplicate ID: also at docs/ja/requirements/index.md:1, docs/ja/requirements/index.md:3\n"
	if count != 3 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 3 errors and output:\n%s", count, output, want)
	}
}

// CTR-FMT-004, CTR-CHK-001: a child outside the heading of its parent is an
// error. The document holds only this error.
func TestReportsAChildOutsideTheHeadingOfItsParent(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "# Requirements\n\n" +
			"### `CTR-FMT-001` A parent\n- 検証: `test` It works\n" +
			"### `CTR-FMT-002` Another requirement\n- 検証: `test` It works\n" +
			"#### `CTR-FMT-001.1` A child under another requirement\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:7: CTR-FMT-001.1: child requirement is not under its parent: put it one level below the heading of CTR-FMT-001\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-004: a child stands under the heading of its parent, one level
// deeper. Every other place is an error.
func TestReportsAChildThatIsNotOneLevelBelowItsParent(t *testing.T) {
	const message = ": CTR-FMT-001.1: child requirement is not under its parent: put it one level below the heading of CTR-FMT-001\n"
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"one level deeper", map[string]string{
			"docs/ja/requirements/index.md": "## `CTR-FMT-001` A parent\n\ntext\n\n### `CTR-FMT-001.1` A child\n- 検証: `test` It works\n",
		}, ""},
		{"two levels deeper", map[string]string{
			"docs/ja/requirements/index.md": "## `CTR-FMT-001` A parent\n- 検証: `test` It works\n#### `CTR-FMT-001.1` A child\n- 検証: `test` It works\n",
		}, "docs/ja/requirements/index.md:3" + message},
		{"the same level as the parent", map[string]string{
			"docs/ja/requirements/index.md": "## `CTR-FMT-001` A parent\n- 検証: `test` It works\n## `CTR-FMT-001.1` A child\n- 検証: `test` It works\n",
		}, "docs/ja/requirements/index.md:3" + message},
		{"before the parent", map[string]string{
			"docs/ja/requirements/index.md": "### `CTR-FMT-001.1` A child\n- 検証: `test` It works\n## `CTR-FMT-001` A parent\n- 検証: `test` It works\n",
		}, "docs/ja/requirements/index.md:1" + message},
		{"under a heading with no ID below the parent", map[string]string{
			"docs/ja/requirements/index.md": "## `CTR-FMT-001` A parent\n- 検証: `test` It works\n### Grouping heading\n\n#### `CTR-FMT-001.1` A child\n- 検証: `test` It works\n",
		}, "docs/ja/requirements/index.md:5" + message},
		{"on another page", map[string]string{
			"docs/ja/requirements/a.md": "## `CTR-FMT-001` A parent\n- 検証: `test` It works\n",
			"docs/ja/requirements/b.md": "### `CTR-FMT-001.1` A child\n- 検証: `test` It works\n",
		}, "docs/ja/requirements/b.md:1" + message},
		{"under another requirement, before the parent", map[string]string{
			"docs/ja/requirements/index.md": "## `CTR-FMT-002` Another requirement\n- 検証: `test` It works\n### `CTR-FMT-001.1` A child\n- 検証: `test` It works\n## `CTR-FMT-001` A parent\n- 検証: `test` It works\n",
		}, "docs/ja/requirements/index.md:3" + message},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.files[".cumin-trace.toml"] = settings
			root := writeRepository(t, test.files)

			_, output := runCheck(t, root)

			if output != test.want {
				t.Errorf("got output:\n%s\nwant output:\n%s", output, test.want)
			}
		})
	}
}

// CTR-FMT-012, CTR-CHK-001: a gap in the numbers of an area is an error. The
// document holds only this error.
func TestReportsAGapInTheNumbersOfAnArea(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "# Requirements\n\n" +
			"### `CTR-FMT-001` The first requirement\n- 検証: `test` It works\n" +
			"### `CTR-FMT-003` The third requirement\n- 検証: `test` It works\n" +
			"### `CTR-CHK-001` Another area with no gap\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:5: CTR-FMT-003: number gap: CTR-FMT-002 is missing\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-012: a gap in the numbers of the children of one parent is an error.
func TestReportsAGapInTheNumbersOfChildren(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "## `CTR-FMT-001` A parent\n- 検証: `test` It works\n" +
			"### `CTR-FMT-001.1` The first child\n- 検証: `test` It works\n" +
			"### `CTR-FMT-001.3` The third child\n- 検証: `test` It works\n" +
			"## `CTR-FMT-002` Another parent\n- 検証: `test` It works\n" +
			"### `CTR-FMT-002.1` The first child of another parent\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:5: CTR-FMT-001.3: number gap: CTR-FMT-001.2 is missing\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-012: the numbers of an area start at 001, and the numbers of
// children start at .1. Each gap gives one error that names what is missing.
func TestReportsEachGapWithTheMissingNumbers(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "## `CTR-FMT-004` Three numbers are missing before it\n- 検証: `test` It works\n" +
			"### `CTR-FMT-004.2` The first child is missing\n- 検証: `test` It works\n" +
			"## `CTR-FMT-006` One number is missing before it\n- 検証: `test` It works\n" +
			"## `CTR-CHK-000` A number below 001\n- 検証: `test` It works\n" +
			"## `CTR-CHK-001` A good requirement\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:1: CTR-FMT-004: number gap: CTR-FMT-001 to CTR-FMT-003 are missing\n" +
		"docs/ja/requirements/index.md:3: CTR-FMT-004.2: number gap: CTR-FMT-004.1 is missing\n" +
		"docs/ja/requirements/index.md:5: CTR-FMT-006: number gap: CTR-FMT-005 is missing\n" +
		"docs/ja/requirements/index.md:7: CTR-CHK-000: number gap: numbers start at 001\n"
	if count != 4 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 4 errors and output:\n%s", count, output, want)
	}
}

// CTR-FMT-012: requirements that do not follow the order of the numbers, on
// one page or across pages, give no error.
func TestAcceptsRequirementsThatDoNotFollowTheOrderOfTheNumbers(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/a.md": "## `CTR-FMT-003` The third requirement\n- 検証: `test` It works\n" +
			"### `CTR-FMT-003.2` The second child\n- 検証: `test` It works\n" +
			"### `CTR-FMT-003.1` The first child\n- 検証: `test` It works\n" +
			"## `CTR-FMT-001` The first requirement\n- 検証: `test` It works\n",
		"docs/ja/requirements/b.md": "## `CTR-FMT-002` The second requirement\n- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	if count != 0 || output != "" {
		t.Errorf("got %d errors and output:\n%s\nwant no error and no output", count, output)
	}
}

// CTR-FMT-005: a list item with another name than the two fields is body
// text, and gives no error. A field in a nested list, in a numbered list, or
// in a quote is body text too.
func TestTakesAListItemWithAnotherNameAsBodyText(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "### `CTR-FMT-001` A good requirement\n\n" +
			"- 備考: `nothing` A list item with another name\n" +
			"- 状態について: 有効\n" +
			"- A list item\n" +
			"  - 状態: 有効\n\n" +
			"1. 検証: `wrong` In a numbered list\n\n" +
			"> - 状態: 有効\n\n" +
			"- 検証: `test` It works\n" +
			"- 状態: 廃止\n\n" +
			"---\n",
	})

	count, output := runCheck(t, root)

	if count != 0 || output != "" {
		t.Errorf("got %d errors and output:\n%s\nwant no error and no output", count, output)
	}
}

// CTR-FMT-006, CTR-CHK-001: a verification that starts with another word than
// the four methods is an error. The document holds only this error.
func TestReportsAVerificationThatStartsWithNoMethodOfTheFour(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "# Requirements\n\n" +
			"### `CTR-FMT-001` A good requirement\n\n" +
			"- 検証: `test` It works\n" +
			"- 検証: `demo` A person sees it\n" +
			"- 検証: `inspection` A person reads it\n" +
			"- 検証: `analysis` A person counts it\n\n" +
			"---\n\n" +
			"### `CTR-FMT-002` A requirement with a wrong method\n\n" +
			"- 検証: `review` A person reads it\n\n" +
			"---\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:14: CTR-FMT-002: verification must start with `test`, `demo`, `inspection`, or `analysis` in a code span\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-006: the method stands first, in a code span, and is one of the
// four words as they are written. Every other start is an error.
func TestReportsEveryVerificationThatDoesNotStartWithAMethodInACodeSpan(t *testing.T) {
	const message = ": CTR-FMT-001: verification must start with `test`, `demo`, `inspection`, or `analysis` in a code span\n"
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "### `CTR-FMT-001` A requirement\n\n" +
			"- 検証: test The method outside a code span\n" +
			"- 検証: The method `test` after other words\n" +
			"- 検証: `Test` The method in upper case\n" +
			"- 検証:\n" +
			"- 検証:`test` No space after the colon\n" +
			"- 検証: ``demo`` Two backticks\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:3" + message +
		"docs/ja/requirements/index.md:4" + message +
		"docs/ja/requirements/index.md:5" + message +
		"docs/ja/requirements/index.md:6" + message
	if count != 4 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 4 errors and output:\n%s", count, output, want)
	}
}

// CTR-FMT-007, CTR-CHK-001: a requirement without children and without a
// verification is an error. A parent without one is no error. The document
// holds only this error.
func TestReportsARequirementWithoutChildrenAndWithoutVerification(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "# Requirements\n\n" +
			"### `CTR-FMT-001` A parent without a verification\n\n" +
			"- 備考: A list item with another name\n\n" +
			"#### `CTR-FMT-001.1` A child with a verification\n\n" +
			"- 検証: `test` It works\n\n" +
			"---\n\n" +
			"### `CTR-FMT-002` A requirement without children\n\n" +
			"It has body text only.\n\n" +
			"- 備考: A list item with another name\n\n" +
			"---\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:13: CTR-FMT-002: no verification: a requirement without children needs a 検証 line\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-007: a verification belongs to the nearest requirement heading
// above it. The verification of a child does not count for a requirement
// without children, and a child without one is an error also when its parent
// has one. A deprecated requirement needs a verification too.
func TestCountsAVerificationOnlyForTheRequirementThatHoldsIt(t *testing.T) {
	const message = ": no verification: a requirement without children needs a 検証 line\n"
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "## `CTR-FMT-001` A parent with a verification\n\n" +
			"- 検証: `test` It works\n\n" +
			"### `CTR-FMT-001.1` A child without a verification\n\n" +
			"#### Grouping heading\n\n" +
			"- 検証: `test` Under a heading with no ID, for the child above\n\n" +
			"### `CTR-FMT-001.2` A child without a verification\n\n" +
			"## `CTR-FMT-002` A deprecated requirement without a verification\n\n" +
			"- 状態: 廃止\n\n" +
			"## `CTR-FMT-003` A requirement with a wrong method only\n\n" +
			"- 検証: `review` A person reads it\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:11: CTR-FMT-001.2" + message +
		"docs/ja/requirements/index.md:13: CTR-FMT-002" + message +
		"docs/ja/requirements/index.md:19: CTR-FMT-003: verification must start with `test`, `demo`, `inspection`, or `analysis` in a code span\n"
	if count != 3 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 3 errors and output:\n%s", count, output, want)
	}
}

// CTR-FMT-008, CTR-CHK-001: a state that says another thing than deprecated
// is an error. The document holds only this error.
func TestReportsAStateThatSaysAnotherThingThanDeprecated(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "# Requirements\n\n" +
			"### `CTR-FMT-001` A deprecated requirement\n\n" +
			"- 検証: `test` It works\n" +
			"- 状態: 廃止\n\n" +
			"---\n\n" +
			"### `CTR-FMT-002` A replaced requirement\n\n" +
			"- 検証: `test` It works\n" +
			"- 状態: 廃止。`CTR-FMT-003` に置き換え\n\n" +
			"---\n\n" +
			"### `CTR-FMT-003` A requirement with a wrong state\n\n" +
			"- 検証: `test` It works\n" +
			"- 状態: 下書き\n\n" +
			"---\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:20: CTR-FMT-003: invalid status: want 廃止, or 廃止。<ID> に置き換え\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-008: the state is 廃止, or 廃止 with a replacement ID of the right
// form. Every other text is an error.
func TestReportsEveryStateThatIsNotOneOfTheTwoForms(t *testing.T) {
	const message = ": CTR-FMT-001: invalid status: want 廃止, or 廃止。<ID> に置き換え\n"
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "### `CTR-FMT-001` A requirement\n\n" +
			"- 検証: `test` It works\n" +
			"- 状態: 廃止。CTR-FMT-002に置き換え\n" +
			"- 状態: 廃止。 `CTR-FMT-002.1`  に置き換え\n" +
			"- 状態: 廃止予定\n" +
			"- 状態: 廃止。\n" +
			"- 状態: 廃止。CTR-FMT-02 に置き換え\n" +
			"- 状態: 廃止。PC-TASK-001 に置き換え\n" +
			"- 状態: 廃止。`CTR-FMT-002 に置き換え\n" +
			"- 状態:\n\n" +
			"### `CTR-FMT-002` The replacement\n\n" +
			"- 検証: `test` It works\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:6" + message +
		"docs/ja/requirements/index.md:7" + message +
		"docs/ja/requirements/index.md:8" + message +
		"docs/ja/requirements/index.md:9" + message +
		"docs/ja/requirements/index.md:10" + message +
		"docs/ja/requirements/index.md:11" + message
	if count != 6 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 6 errors and output:\n%s", count, output, want)
	}
}

// CTR-FMT-010, CTR-CHK-001: a reference to an ID that does not exist is an
// error. The document holds only this error.
func TestReportsAReferenceToAnIDThatDoesNotExist(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "# Requirements\n\n" +
			"### `CTR-FMT-001` A requirement\n\n" +
			"It works like `CTR-FMT-002`.\n\n" +
			"- 検証: `test` It works\n\n" +
			"---\n\n" +
			"### `CTR-FMT-002` A requirement that refers to a missing one\n\n" +
			"It works with CTR-FMT-003.\n\n" +
			"- 検証: `test` It works\n\n" +
			"---\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:13: CTR-FMT-003: reference to an ID that does not exist\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-010, CTR-CHK-001: a reference to a deprecated requirement is an
// error. The replacement ID of a 状態 line may name a deprecated requirement.
// The document holds only this error.
func TestReportsAReferenceToADeprecatedRequirement(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "# Requirements\n\n" +
			"### `CTR-FMT-001` A deprecated requirement\n\n" +
			"- 検証: `test` It works\n" +
			"- 状態: 廃止\n\n" +
			"---\n\n" +
			"### `CTR-FMT-002` A requirement replaced by a deprecated one\n\n" +
			"- 検証: `test` It works\n" +
			"- 状態: 廃止。`CTR-FMT-001` に置き換え\n\n" +
			"---\n\n" +
			"### `CTR-FMT-003` A requirement that refers to a deprecated one\n\n" +
			"It works like `CTR-FMT-002`.\n\n" +
			"- 検証: `test` It works\n\n" +
			"---\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:19: CTR-FMT-002: reference to a deprecated requirement\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-010: a replacement ID that names no requirement is an error at the
// 状態 line. Another 状態 line holds no reference.
func TestReportsAReplacementIDThatDoesNotExist(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/index.md": "### `CTR-FMT-001` A replaced requirement\n\n" +
			"- 検証: `test` It works\n" +
			"- 状態: 廃止。CTR-FMT-002 に置き換え\n" +
			"- 状態: 廃止。CTR-FMT-003 に置き換え\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:4: CTR-FMT-002: reference to an ID that does not exist\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-FMT-010: references are read in every Markdown file of the directory,
// also in a file with no requirement. The output names the file and the line
// of the reference.
func TestReadsReferencesInAFileWithNoRequirement(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml": settings,
		"docs/ja/requirements/format/index.md": "### `CTR-FMT-001` A requirement\n\n" +
			"- 検証: `test` It works\n",
		"docs/ja/requirements/index.md": "# Requirements\n\n" +
			"The format is in CTR-FMT-001, and the check in CTR-CHK-001.\n",
	})

	count, output := runCheck(t, root)

	want := "docs/ja/requirements/index.md:3: CTR-CHK-001: reference to an ID that does not exist\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%s\nwant 1 error and output:\n%s", count, output, want)
	}
}

// CTR-CHK-002: an ID with control characters is printed with a Go escape for
// each of them, so that the error stays on one line and holds no terminal code.
func TestPrintsTheControlCharactersOfAnIDAsEscapes(t *testing.T) {
	root := writeRepository(t, map[string]string{
		".cumin-trace.toml":             settings,
		"docs/ja/requirements/index.md": "## `CTR-\x1b[31mFMT\x07-001` A heading with control characters\n",
	})

	count, output := runCheck(t, root)

	want := `docs/ja/requirements/index.md:1: CTR-\x1b[31mFMT\a-001: invalid ID format: want <prefix>-<area>-<3 digits>, or that with .<child number>` + "\n"
	if count != 1 || output != want {
		t.Errorf("got %d errors and output:\n%q\nwant 1 error and output:\n%q", count, output, want)
	}
}
