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
	designShowsEmbedded(t, "## プロダクトの check の workflow", "workflows/cumin-trace-check.yml")
}

// CTR-DASH-008: the design document shows the same text of the example
// workflow of the site as the embedded template.
func TestTheDesignShowsTheTextOfTheEmbeddedSiteWorkflow(t *testing.T) {
	designShowsEmbedded(t, "## プロダクトのサイトの workflow", "workflows/cumin-trace-site.yml")
}

// designShowsEmbedded fails the test when the YAML block of the section of
// release.md differs from the embedded file.
func designShowsEmbedded(t *testing.T, heading, source string) {
	t.Helper()
	embedded, err := templates.Files.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	design, err := os.ReadFile(filepath.Join("..", "..", "docs", "ja", "designs", "templates", "release.md"))
	if err != nil {
		t.Fatal(err)
	}

	_, section, found := strings.Cut(string(design), "\n"+heading+"\n")
	_, block, inBlock := strings.Cut(section, "\n```yaml\n")
	shown, _, closed := strings.Cut(block, "\n```\n")
	if !found || !inBlock || !closed {
		t.Fatalf("release.md has no YAML block in the section %q", heading)
	}
	if shown+"\n" != string(embedded) {
		t.Errorf("release.md shows:\n%s\nthe embedded workflow is:\n%s", shown, embedded)
	}
}

// CTR-CHK-005, CTR-DASH-008: a binary without the version of a release
// writes no workflow, and says so with one line for each workflow. A
// workflow that exists is reported as such.
func TestSkipsTheWorkflowWithoutTheVersionOfARelease(t *testing.T) {
	const noVersion = "skipped .github/workflows/cumin-trace-check.yml: this cumin-trace has no version; use a release binary\n" +
		"skipped .github/workflows/cumin-trace-site.yml: this cumin-trace has no version; use a release binary\n"
	for _, version := range []string{"", "(devel)", "v0.1.1-0.20261014120000-2c029c5abcde", "v0.1.0+dirty"} {
		root := t.TempDir()

		output := runInitWithVersion(t, root, "PC", "", version)

		if !strings.HasSuffix(output, "\n"+noVersion) {
			t.Errorf("with version %q: got output:\n%s\nwant the last lines %q", version, output, noVersion)
		}
		for _, workflow := range []string{CheckWorkflow, SiteWorkflow} {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(workflow))); !os.IsNotExist(err) {
				t.Errorf("with version %q: got error %v for %s, want no file", version, err, workflow)
			}
		}
	}

	root := t.TempDir()
	runInit(t, root, "PC", "")

	output := runInitWithVersion(t, root, "PC", "", "")

	const exists = "skipped .github/workflows/cumin-trace-check.yml: already exists\n" +
		"skipped .github/workflows/cumin-trace-site.yml: already exists\n"
	if !strings.HasSuffix(output, exists) {
		t.Errorf("got output:\n%s\nwant the workflows skipped as files that exist", output)
	}
}

// CTR-DASH-008: after init with the binary of a release, the example
// workflow runs on a push to main and on a schedule, with read permissions
// only. It downloads the release binary of that version, writes the trace
// to site/trace.json, and then builds the site. It publishes nothing.
func TestWritesTheWorkflowThatRebuildsTheSiteWithTheTrace(t *testing.T) {
	root := t.TempDir()
	runInitWithVersion(t, root, "PC", "", "v1.2.3")

	workflow := readFile(t, root, SiteWorkflow)

	for _, want := range []string{
		"on:\n  push:\n    branches:\n      - main\n  schedule:\n    - cron: ",
		"permissions:\n  contents: read\n  issues: read\n  pull-requests: read\n\njobs:",
		"VERSION: v1.2.3\n",
		`gh release download "$VERSION" --repo cloveclovedev/cumin-trace`,
		"sha256sum --check -",
		"GITHUB_TOKEN: ${{ github.token }}",
		`cumin-trace-linux-amd64" trace > site/trace.json'`,
		"node-version-file: site/.node-version",
		"run: npm --prefix site ci\n",
		"run: npm --prefix site run build\n",
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("the workflow has no %q:\n%s", want, workflow)
		}
	}
	for _, unwanted := range []string{"{{version}}", ": write", "uses: actions/upload-", "uses: actions/deploy-"} {
		if strings.Contains(workflow, unwanted) {
			t.Errorf("the workflow has %q:\n%s", unwanted, workflow)
		}
	}
}

// CTR-DASH-008: when the trace fails, the build does not run. The trace
// step comes before the build steps, and no step overrides the default
// status check success() or lets the job pass with a failed step.
func TestTheSiteWorkflowBuildsOnlyAfterTheTrace(t *testing.T) {
	workflow, err := templates.Files.ReadFile("workflows/cumin-trace-site.yml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(workflow)

	trace := strings.Index(text, "- name: cumin-trace trace\n")
	for _, step := range []string{"- uses: actions/setup-node@", "- name: npm ci\n", "- name: Build the site\n"} {
		if at := strings.Index(text, step); trace < 0 || at < trace {
			t.Errorf("the step %q comes before the trace, or the trace is missing:\n%s", step, text)
		}
	}
	for _, unwanted := range []string{"if:", "continue-on-error", "|| true"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("the workflow has %q, which can run a step after a failed trace:\n%s", unwanted, text)
		}
	}
}
