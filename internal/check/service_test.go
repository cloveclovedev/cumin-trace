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
			"### `CTR-FMT-001` A good requirement\n\n" +
			"### `CTR-OTHER-001` A requirement that proves the reading\n",
		"docs/elsewhere/index.md": "### `CTR-NOWHERE-001` Outside the directory\n",
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
		".cumin-trace.toml":              settings + "requirements_dir = \"requirements\"\n",
		"requirements/deep/down/page.md": "## `CTR-OTHER-001` In the named directory\n",
		"docs/ja/requirements/index.md":  "## `CTR-NOWHERE-001` In the default directory\n",
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
			"### `CTR-FMT-001` A good requirement\n\n" +
			"#### `CTR-FMT-001.1` A good child\n\n" +
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
			"### `CTR-CHK-001` A good requirement\n\n" +
			"### `CTR-SITE-001` An area outside the settings\n",
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
			"### `CTR-SITE-001` An area outside the settings\n",
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
		"docs/ja/requirements/b/index.md": "## `CTR-X-001` One\n\n## `CTR-FMT-1` Two\n",
		"docs/ja/requirements/a/index.md": "## `CTR-Y-001` One\n",
		"docs/ja/requirements/a-b.md":     "## `CTR-Z-001` One\n",
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
	document := "### `CTR-FMT-001` A good requirement\n"
	tests := []struct {
		name     string
		settings string
		want     string
	}{
		{"a syntax error", "prefix = \"CTR\"\nareas = oops\n", ".cumin-trace.toml:2:"},
		{"no prefix", "areas = [\"FMT\"]\n", "prefix is missing"},
		{"a prefix in lower case", "prefix = \"ctr\"\nareas = [\"FMT\"]\n", "prefix \"ctr\""},
		{"no area", "prefix = \"CTR\"\nareas = []\n", "at least one area"},
		{"an area named twice", "prefix = \"CTR\"\nareas = [\"FMT\", \"FMT\"]\n", "named twice"},
		{"an unknown key", settings + "area = \"FMT\"\n", "unknown key area"},
		{"a directory above the repository", settings + "requirements_dir = \"../docs\"\n", "must not contain .."},
		{"a directory that does not exist", settings + "requirements_dir = \"nowhere\"\n", "nowhere: no such directory"},
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
