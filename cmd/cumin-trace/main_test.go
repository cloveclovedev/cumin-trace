package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// writeRepository writes the settings and one requirement document into a new
// temporary directory and returns it.
func writeRepository(t *testing.T, document string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "ja", "requirements")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := "prefix = \"CTR\"\nareas = [\"FMT\"]\n"
	if err := os.WriteFile(filepath.Join(root, ".cumin-trace.toml"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// CTR-CHK-003: the exit code is 0 when the documents hold no error.
func TestExitsWithZeroWhenNoErrorExists(t *testing.T) {
	root := writeRepository(t, "### `CTR-FMT-001` A good requirement\n")
	var stdout, stderr bytes.Buffer

	code := run([]string{"check"}, root, &stdout, &stderr)

	if code != 0 || stdout.String() != "" || stderr.String() != "" {
		t.Errorf("got exit code %d, stdout %q, stderr %q; want 0 and no output", code, stdout.String(), stderr.String())
	}
}

// CTR-CHK-003: the exit code is not 0 when the documents hold an error.
func TestExitsWithOneWhenAnErrorExists(t *testing.T) {
	root := writeRepository(t, "### `CTR-FMT-1` A wrong ID\n")
	var stdout, stderr bytes.Buffer

	code := run([]string{"check"}, root, &stdout, &stderr)

	wantStdout := "docs/ja/requirements/index.md:1: CTR-FMT-1: invalid ID format: want <prefix>-<area>-<3 digits>, or that with .<child number>\n"
	if code != 1 || stdout.String() != wantStdout || stderr.String() != "cumin-trace: 1 error\n" {
		t.Errorf("got exit code %d, stdout %q, stderr %q; want 1, the error line, and the count", code, stdout.String(), stderr.String())
	}
}

// CTR-CHK-003: the exit code is 2 when the check cannot run.
func TestExitsWithTwoWhenTheCheckCannotRun(t *testing.T) {
	tests := []struct {
		name string
		args []string
		root string
	}{
		{"no settings file", []string{"check"}, t.TempDir()},
		{"no command", nil, writeRepository(t, "")},
		{"an unknown command", []string{"trace"}, writeRepository(t, "")},
		{"an extra argument", []string{"check", "docs"}, writeRepository(t, "")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := run(test.args, test.root, &stdout, &stderr)

			if code != 2 || stdout.String() != "" || stderr.String() == "" {
				t.Errorf("got exit code %d, stdout %q, stderr %q; want 2, and a message on stderr only", code, stdout.String(), stderr.String())
			}
		})
	}
}
