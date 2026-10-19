package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloveclovedev/cumin-trace/templates"
)

// CTR-CHK-005: after init with the binary of a release, the workflow runs on
// every pull request, downloads the release binary of that version, checks
// its checksum, and runs cumin-trace check.
func TestWritesTheWorkflowThatRunsTheCheckOnPullRequests(t *testing.T) {
	root := t.TempDir()
	runInitWithVersion(t, root, "PC", "", "v1.2.3")

	workflow := readFile(t, root, CheckWorkflow)

	for _, want := range []string{
		"on:\n  pull_request:\n\n",
		"permissions:\n  contents: read\n",
		"VERSION: v1.2.3\n",
		`gh release download "$VERSION" --repo cloveclovedev/cumin-trace`,
		"sha256sum --check -",
		`cumin-trace-linux-amd64" check'`,
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("the workflow has no %q:\n%s", want, workflow)
		}
	}
	if strings.Contains(workflow, "{{version}}") || strings.Contains(workflow, "setup-go") {
		t.Errorf("the workflow holds a value that is not replaced, or sets up Go:\n%s", workflow)
	}
	if !strings.Contains(workflow, "GH_TOKEN: ${{ github.token }}") {
		t.Errorf("the workflow lost the expression of the token:\n%s", workflow)
	}
}

// CTR-CHK-005: the design document shows the same text of the workflow as
// the embedded template.
func TestTheDesignShowsTheTextOfTheEmbeddedWorkflow(t *testing.T) {
	embedded, err := templates.Files.ReadFile("workflows/cumin-trace-check.yml")
	if err != nil {
		t.Fatal(err)
	}
	design, err := os.ReadFile(filepath.Join("..", "..", "docs", "ja", "designs", "templates", "release.md"))
	if err != nil {
		t.Fatal(err)
	}

	_, section, found := strings.Cut(string(design), "\n## プロダクトの check の workflow\n")
	_, block, inBlock := strings.Cut(section, "\n```yaml\n")
	shown, _, closed := strings.Cut(block, "\n```\n")
	if !found || !inBlock || !closed {
		t.Fatalf("release.md has no YAML block in the section of the check workflow")
	}
	if shown+"\n" != string(embedded) {
		t.Errorf("release.md shows:\n%s\nthe embedded workflow is:\n%s", shown, embedded)
	}
}

// CTR-CHK-005: a binary without the version of a release writes no workflow,
// and says so. A workflow that exists is reported as such.
func TestSkipsTheWorkflowWithoutTheVersionOfARelease(t *testing.T) {
	const noVersion = "skipped .github/workflows/cumin-trace-check.yml: this cumin-trace has no version; use a release binary\n"
	for _, version := range []string{"", "(devel)", "v0.1.1-0.20261014120000-2c029c5abcde", "v0.1.0+dirty"} {
		root := t.TempDir()

		output := runInitWithVersion(t, root, "PC", "", version)

		if !strings.HasSuffix(output, "\n"+noVersion) {
			t.Errorf("with version %q: got output:\n%s\nwant the last line %q", version, output, noVersion)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(CheckWorkflow))); !os.IsNotExist(err) {
			t.Errorf("with version %q: got error %v for the workflow, want no file", version, err)
		}
	}

	root := t.TempDir()
	runInit(t, root, "PC", "")

	output := runInitWithVersion(t, root, "PC", "", "")

	if !strings.HasSuffix(output, "skipped .github/workflows/cumin-trace-check.yml: already exists\n") {
		t.Errorf("got output:\n%s\nwant the workflow skipped as a file that exists", output)
	}
}
