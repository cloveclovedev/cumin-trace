package export

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// writeRepository writes the settings and the requirement documents into a
// new temporary directory and returns it. The keys of files are paths under
// the directory of the requirement documents.
func writeRepository(t *testing.T, areas string, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	settings := "prefix = \"CTR\"\nareas = " + areas + "\n"
	if err := os.WriteFile(filepath.Join(root, ".cumin-trace.toml"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		path := filepath.Join(root, "docs", "ja", "requirements", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const parentChildDeprecated = "# Trace\n" +
	"\n" +
	"### `CTR-TRC-001` Progress\n" +
	"\n" +
	"Each requirement has a progress.\n" +
	"\n" +
	"#### `CTR-TRC-001.1` Not started\n" +
	"\n" +
	"A requirement with no issue is not started.\n" +
	"\n" +
	"- 検証: `test` A requirement with no issue is not started\n" +
	"\n" +
	"---\n" +
	"\n" +
	"### `CTR-TRC-002` Old progress\n" +
	"\n" +
	"- 検証: `inspection` Read it\n" +
	"- 状態: 廃止。`CTR-TRC-001` に置き換え\n" +
	"\n" +
	"---\n"

// CTR-EXP-001: a parent, a child, and a deprecated requirement give the keys
// of the example in the same order, [] for an empty list, and null for a
// missing value.
func TestExportWritesEveryRequirementWithTheKeysOfTheExample(t *testing.T) {
	root := writeRepository(t, `["TRC"]`, map[string]string{"trace/index.md": parentChildDeprecated})
	var stdout, stderr bytes.Buffer

	count, err := Run(root, &stdout, &stderr)

	want := `{
  "requirements": [
    {
      "id": "CTR-TRC-001",
      "parent": null,
      "title": "Progress",
      "text": "Each requirement has a progress.",
      "verifications": [],
      "status": "active",
      "replaced_by": null,
      "file": "docs/ja/requirements/trace/index.md",
      "line": 3
    },
    {
      "id": "CTR-TRC-001.1",
      "parent": "CTR-TRC-001",
      "title": "Not started",
      "text": "A requirement with no issue is not started.",
      "verifications": [
        {
          "method": "test",
          "how": "A requirement with no issue is not started"
        }
      ],
      "status": "active",
      "replaced_by": null,
      "file": "docs/ja/requirements/trace/index.md",
      "line": 7
    },
    {
      "id": "CTR-TRC-002",
      "parent": null,
      "title": "Old progress",
      "text": "",
      "verifications": [
        {
          "method": "inspection",
          "how": "Read it"
        }
      ],
      "status": "deprecated",
      "replaced_by": "CTR-TRC-001",
      "file": "docs/ja/requirements/trace/index.md",
      "line": 15
    }
  ]
}
`
	if err != nil || count != 0 || stderr.String() != "" {
		t.Fatalf("got count %d, error %v, stderr %q; want no error", count, err, stderr.String())
	}
	if stdout.String() != want {
		t.Errorf("got JSON\n%s\nwant\n%s", stdout.String(), want)
	}
}

// CTR-EXP-002: two runs give the same bytes, and the requirements follow the
// order of the areas in the settings, also when the file of the first area
// sorts later.
func TestExportGivesTheSameBytesInTheOrderOfTheAreas(t *testing.T) {
	root := writeRepository(t, `["TRC", "CHK"]`, map[string]string{
		"check/index.md": "### `CTR-CHK-001` Check\n\n- 検証: `test` It checks\n",
		"trace/index.md": "### `CTR-TRC-001` Trace\n\n- 検証: `test` It traces\n",
	})
	var first, second, stderr bytes.Buffer

	if _, err := Run(root, &first, &stderr); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(root, &second, &stderr); err != nil {
		t.Fatal(err)
	}

	if first.String() != second.String() {
		t.Errorf("got two different outputs:\n%s\n%s", first.String(), second.String())
	}
	trace := bytes.Index(first.Bytes(), []byte(`"CTR-TRC-001"`))
	check := bytes.Index(first.Bytes(), []byte(`"CTR-CHK-001"`))
	if trace < 0 || check < 0 || trace > check {
		t.Errorf("got JSON\n%s\nwant CTR-TRC-001 before CTR-CHK-001", first.String())
	}
}

// CTR-EXP-003: with one format error, the export writes no JSON, writes the
// error line, and returns one error.
func TestExportWithAFormatErrorWritesNoJSON(t *testing.T) {
	root := writeRepository(t, `["TRC"]`, map[string]string{"trace/index.md": "### `CTR-TRC-1` A wrong ID\n"})
	var stdout, stderr bytes.Buffer

	count, err := Run(root, &stdout, &stderr)

	if err != nil || count != 1 || stdout.String() != "" || stderr.Len() == 0 {
		t.Errorf("got count %d, error %v, stdout %q, stderr %q; want 1, no JSON, and the error line", count, err, stdout.String(), stderr.String())
	}
}
