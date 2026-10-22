package trace

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// gitOfTest runs git in dir with a fixed identity and the date for both the
// author and the committer. It reads no configuration of the machine.
func gitOfTest(t *testing.T, dir, date string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// commitOfTest commits every file of the repository at root at the date.
func commitOfTest(t *testing.T, root, date, message string) {
	t.Helper()
	gitOfTest(t, root, date, "add", "--all")
	gitOfTest(t, root, date, "commit", "--quiet", "--allow-empty", "--message", message)
}

// writeDocument writes one requirement document of the test repository.
func writeDocument(t *testing.T, root, name, document string) {
	t.Helper()
	path := filepath.Join(root, "docs", "ja", "requirements", name)
	if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
}

// lastChangesOfTest reads the time of the last change of each ID of the test
// repository, in UTC and RFC 3339.
func lastChangesOfTest(t *testing.T, root string) map[string]string {
	t.Helper()
	changes, err := lastChangesOf(root, "docs/ja/requirements", "PC")
	if err != nil {
		t.Fatalf("lastChangesOf: %v", err)
	}
	got := map[string]string{}
	for id, when := range changes {
		got[id] = when.UTC().Format(time.RFC3339)
	}
	return got
}

const twoRequirementsOfTest = "# Tasks\n\n" +
	"### `PC-TASK-001` One\n\nThe first text.\n\n- 検証: `test` It works\n\n---\n\n" +
	"### `PC-TASK-002` Two\n\nThe second text.\n\n- 検証: `test` It works\n"

// CTR-TRC-010: a change of the text of a requirement gives it the time of
// that commit. The other requirement keeps the time of its first commit.
func TestAChangeOfTheTextGivesTheTimeOfThatCommit(t *testing.T) {
	root := writeRepository(t, settingsOfTest, twoRequirementsOfTest)
	writeDocument(t, root, "index.md", strings.Replace(twoRequirementsOfTest, "The first text.", "The first text, changed.", 1))
	commitOfTest(t, root, "2026-10-05T12:30:00+09:00", "docs: change the first requirement")

	got := lastChangesOfTest(t, root)

	want := map[string]string{"PC-TASK-001": "2026-10-05T03:30:00Z", "PC-TASK-002": "2026-10-01T09:00:00Z"}
	if len(got) != len(want) || got["PC-TASK-001"] != want["PC-TASK-001"] || got["PC-TASK-002"] != want["PC-TASK-002"] {
		t.Errorf("got %v, want %v", got, want)
	}
}

// CTR-TRC-010: a change of the title or of a verification is a change too.
func TestAChangeOfTheTitleOrOfAVerificationIsAChange(t *testing.T) {
	root := writeRepository(t, settingsOfTest, twoRequirementsOfTest)
	changed := strings.Replace(twoRequirementsOfTest, "`PC-TASK-001` One", "`PC-TASK-001` First", 1)
	changed = strings.Replace(changed, "The second text.\n\n- 検証: `test` It works", "The second text.\n\n- 検証: `demo` It works", 1)
	writeDocument(t, root, "index.md", changed)
	commitOfTest(t, root, "2026-10-06T00:00:00Z", "docs: change the title and a verification")

	got := lastChangesOfTest(t, root)

	if got["PC-TASK-001"] != "2026-10-06T00:00:00Z" || got["PC-TASK-002"] != "2026-10-06T00:00:00Z" {
		t.Errorf("got %v, want the time of the second commit for both", got)
	}
}

// CTR-TRC-010: a commit that only moves a requirement to another file and
// line, and a commit whose message has the line Requirement-Unchanged with
// the ID, change nothing for that ID.
func TestAMoveAndARequirementUnchangedLineChangeNothing(t *testing.T) {
	root := writeRepository(t, settingsOfTest, twoRequirementsOfTest)
	first, second, _ := strings.Cut(twoRequirementsOfTest, "\n---\n\n")
	writeDocument(t, root, "index.md", first+"\n")
	writeDocument(t, root, "other.md", "# More tasks\n\nAn introduction.\n\n"+second)
	commitOfTest(t, root, "2026-10-02T00:00:00Z", "docs: move the second requirement")
	writeDocument(t, root, "index.md", strings.Replace(first, "The first text.", "The first text, with a typo fixed.", 1)+"\n")
	commitOfTest(t, root, "2026-10-03T00:00:00Z", "docs: fix a typo\n\nRequirement-Unchanged: PC-TASK-001\n")

	got := lastChangesOfTest(t, root)

	if got["PC-TASK-001"] != "2026-10-01T09:00:00Z" || got["PC-TASK-002"] != "2026-10-01T09:00:00Z" {
		t.Errorf("got %v, want the time of the first commit for both", got)
	}
}

// CTR-TRC-010: the line Requirement-Unchanged covers only the ID that it
// names.
func TestARequirementUnchangedLineCoversOnlyItsID(t *testing.T) {
	root := writeRepository(t, settingsOfTest, twoRequirementsOfTest)
	changed := strings.Replace(twoRequirementsOfTest, "The first text.", "The first text, changed.", 1)
	writeDocument(t, root, "index.md", strings.Replace(changed, "The second text.", "The second text, changed.", 1))
	commitOfTest(t, root, "2026-10-04T00:00:00Z", "docs: change both\n\nRequirement-Unchanged: PC-TASK-002\n")

	got := lastChangesOfTest(t, root)

	if got["PC-TASK-001"] != "2026-10-04T00:00:00Z" || got["PC-TASK-002"] != "2026-10-01T09:00:00Z" {
		t.Errorf("got %v, want the second commit for PC-TASK-001 and the first for PC-TASK-002", got)
	}
}

// CTR-TRC-010: in a shallow clone, the trace ends with an error before any
// GitHub call, and writes no JSON.
func TestAShallowCloneStopsTheTraceBeforeGitHub(t *testing.T) {
	source := writeRepository(t, settingsOfTest, documentOfTest)
	writeDocument(t, source, "index.md", documentOfTest+"\n")
	commitOfTest(t, source, "2026-10-02T00:00:00Z", "docs: add a line")
	root := filepath.Join(t.TempDir(), "clone")
	gitOfTest(t, source, "", "clone", "--quiet", "--depth", "1", "file://"+filepath.ToSlash(source), root)
	github := &fakeGitHub{t: t}

	count, err, stdout, _ := runTrace(t, root, github)

	if count != 0 || err == nil || !strings.Contains(err.Error(), "shallow git repository") || stdout != "" || len(github.requests) != 0 {
		t.Errorf("got %d, %v, stdout %q, %d requests; want the error of a shallow repository, no JSON, and no request", count, err, stdout, len(github.requests))
	}
}
