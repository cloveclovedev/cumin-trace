package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloveclovedev/cumin-trace/internal/trace"
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
	root := writeRepository(t, "### `CTR-FMT-001` A good requirement\n- 検証: `test` It works\n")
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
		{"an unknown command", []string{"show"}, writeRepository(t, "")},
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

// CTR-FMT-011: the requirement documents of this repository pass the check
// with the settings of this repository. Go runs the test in the directory of
// the package, so the top of the repository is two levels up.
func TestTheDocumentsOfThisRepositoryPassTheCheck(t *testing.T) {
	root := filepath.Join("..", "..")
	var stdout, stderr bytes.Buffer

	code := run([]string{"check"}, root, &stdout, &stderr)

	if code != 0 || stdout.String() != "" || stderr.String() != "" {
		t.Errorf("got exit code %d, stdout %q, stderr %q; want 0 and no output", code, stdout.String(), stderr.String())
	}
}

// CTR-TPL-001: init then check in an empty directory both exit with 0, with
// and without an area.
func TestInitThenCheckExitWithZero(t *testing.T) {
	for name, args := range map[string][]string{
		"with an area":    {"init", "PC", "TASK"},
		"without an area": {"init", "PC"},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			var stdout, stderr bytes.Buffer

			initCode := run(args, root, &stdout, &stderr)
			checkCode := run([]string{"check"}, root, &bytes.Buffer{}, &stderr)

			if initCode != 0 || checkCode != 0 || stderr.String() != "" {
				t.Errorf("got exit codes %d and %d, stderr %q; want 0, 0, and no error", initCode, checkCode, stderr.String())
			}
		})
	}
}

// CTR-TPL-001: init exits with 2 and writes no file when the arguments are
// wrong.
func TestInitExitsWithTwoOnWrongArguments(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{"no prefix", []string{"init"}, "cumin-trace: usage: cumin-trace init <prefix> [<area>]\n"},
		{"an extra argument", []string{"init", "PC", "TASK", "MORE"}, "cumin-trace: usage: cumin-trace init <prefix> [<area>]\n"},
		{"a wrong prefix", []string{"init", "pc"}, "cumin-trace: invalid prefix \"pc\": want upper-case letters and digits\n"},
		{"an empty area", []string{"init", "PC", ""}, "cumin-trace: invalid area \"\": want upper-case letters and digits\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			var stdout, stderr bytes.Buffer

			code := run(test.args, root, &stdout, &stderr)

			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			if code != 2 || stdout.String() != "" || stderr.String() != test.wantStderr || len(entries) != 0 {
				t.Errorf("got exit code %d, stdout %q, stderr %q, %d files; want 2, stderr %q, and no file", code, stdout.String(), stderr.String(), len(entries), test.wantStderr)
			}
		})
	}
}

// setVersion sets the version that the build sets, for one test.
func setVersion(t *testing.T, v string) {
	t.Helper()
	saved := version
	version = v
	t.Cleanup(func() { version = saved })
}

// CTR-CMD-002: version prints the version that the build set, and exits with
// 0.
func TestVersionPrintsTheVersionThatTheBuildSet(t *testing.T) {
	setVersion(t, "v1.2.3")
	var stdout, stderr bytes.Buffer

	code := run([]string{"version"}, t.TempDir(), &stdout, &stderr)

	if code != 0 || stdout.String() != "v1.2.3\n" || stderr.String() != "" {
		t.Errorf("got exit code %d, stdout %q, stderr %q; want 0 and %q", code, stdout.String(), stderr.String(), "v1.2.3\n")
	}
}

// CTR-CMD-002: a build without a version still prints one line, and exits
// with 0. A test binary has no version that the build set.
func TestVersionPrintsOneLineWithoutAVersion(t *testing.T) {
	setVersion(t, "")
	var stdout, stderr bytes.Buffer

	code := run([]string{"version"}, t.TempDir(), &stdout, &stderr)

	line, rest, found := strings.Cut(stdout.String(), "\n")
	if code != 0 || line == "" || !found || rest != "" || stderr.String() != "" {
		t.Errorf("got exit code %d, stdout %q, stderr %q; want 0 and one line", code, stdout.String(), stderr.String())
	}
}

// CTR-CMD-003: with no command, an unknown command, or a wrong number of
// arguments, the usage line is on stderr, no file is written, and the exit
// code is not 0.
func TestWrongArgumentsPrintTheUsageAndWriteNoFile(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no command", nil},
		{"an unknown command", []string{"show"}},
		{"an extra argument to check", []string{"check", "docs"}},
		{"an extra argument to export", []string{"export", "x"}},
		{"an extra argument to trace", []string{"trace", "now"}},
		{"an extra argument to version", []string{"version", "now"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			var stdout, stderr bytes.Buffer

			code := run(test.args, root, &stdout, &stderr)

			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			wantStderr := usage + "\n"
			if code == 0 || stdout.String() != "" || stderr.String() != wantStderr || len(entries) != 0 {
				t.Errorf("got exit code %d, stdout %q, stderr %q, %d files; want not 0, stderr %q, and no file", code, stdout.String(), stderr.String(), len(entries), wantStderr)
			}
		})
	}
}

// CTR-TRC-007: without a token, the trace calls no GitHub, writes no JSON,
// and exits with 2.
func TestTraceWithoutATokenExitsWithTwo(t *testing.T) {
	root := writeRepository(t, "### `CTR-FMT-001` A good requirement\n- 検証: `test` It works\n")
	var stdout, stderr bytes.Buffer

	code := runTrace(root, trace.Env{Repository: "octocat/Hello-World", Now: time.Now}, &stdout, &stderr)

	if code != 2 || stdout.String() != "" || stderr.String() != "cumin-trace: GITHUB_TOKEN is not set\n" {
		t.Errorf("got exit code %d, stdout %q, stderr %q; want 2 and the message of the token", code, stdout.String(), stderr.String())
	}
}

// CTR-TRC-009: on a format error, the trace prints the same error line as
// the check, calls no GitHub, writes no JSON, and exits with 1.
func TestTraceWithAFormatErrorExitsWithOne(t *testing.T) {
	root := writeRepository(t, "### `CTR-FMT-1` A wrong ID\n")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
	defer server.Close()
	var checkStdout, checkStderr, stdout, stderr bytes.Buffer
	if code := run([]string{"check"}, root, &checkStdout, &checkStderr); code != 1 || checkStdout.Len() == 0 {
		t.Fatalf("check: got exit code %d, stdout %q; want 1 and an error line", code, checkStdout.String())
	}

	code := runTrace(root, trace.Env{Token: "test-token", Repository: "octocat/Hello-World", GraphQLURL: server.URL, Now: time.Now}, &stdout, &stderr)

	wantStderr := checkStdout.String() + "cumin-trace: 1 error\n"
	if code != 1 || stdout.String() != "" || stderr.String() != wantStderr || requests.Load() != 0 {
		t.Errorf("got exit code %d, stdout %q, stderr %q, %d requests; want 1, no JSON, stderr %q, and no request", code, stdout.String(), stderr.String(), requests.Load(), wantStderr)
	}
}

// CTR-EXP-003: on a format error, the export prints the same error line as
// the check, writes no JSON, and exits with 1.
func TestExportWithAFormatErrorExitsWithOne(t *testing.T) {
	root := writeRepository(t, "### `CTR-FMT-1` A wrong ID\n")
	var checkStdout, checkStderr, stdout, stderr bytes.Buffer
	if code := run([]string{"check"}, root, &checkStdout, &checkStderr); code != 1 || checkStdout.Len() == 0 {
		t.Fatalf("check: got exit code %d, stdout %q; want 1 and an error line", code, checkStdout.String())
	}

	code := run([]string{"export"}, root, &stdout, &stderr)

	wantStderr := checkStdout.String() + "cumin-trace: 1 error\n"
	if code != 1 || stdout.String() != "" || stderr.String() != wantStderr {
		t.Errorf("got exit code %d, stdout %q, stderr %q; want 1, no JSON, and stderr %q", code, stdout.String(), stderr.String(), wantStderr)
	}
}

// CTR-EXP-001: the documents of this repository export with the exit code 0.
func TestTheDocumentsOfThisRepositoryExport(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"export"}, filepath.Join("..", ".."), &stdout, &stderr)

	if code != 0 || !strings.HasPrefix(stdout.String(), "{\n  \"requirements\": [") || stderr.String() != "" {
		t.Errorf("got exit code %d, stderr %q; want 0, the JSON, and no stderr", code, stderr.String())
	}
}
