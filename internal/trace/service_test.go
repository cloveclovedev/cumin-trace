package trace

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// token is the token of the tests. It is no real token.
const token = "test-token-0123456789"

// fakeIssue is one issue on the fake GitHub.
type fakeIssue struct {
	number      int
	labels      []string
	state       string
	stateReason string
	milestone   string
	body        string
	subIssues   []fakeSubIssue
}

// fakeSubIssue is one sub-issue on the fake GitHub.
type fakeSubIssue struct {
	// repository is empty for octocat/Hello-World.
	repository string
	number     int
	state      string
	milestone  string
	// pullRequest is the number of the pull request in octocat/Hello-World
	// that closed the sub-issue, or 0.
	pullRequest int
	// byCommit is true for a sub-issue that a commit closed.
	byCommit bool
}

// fakeGitHub answers the query of the requirement issues with the issues that
// have the label of the query, two on one page, and two sub-issues of each on
// one page. It answers the queries of more sub-issues and of the closers too.
// It records each request.
type fakeGitHub struct {
	t        *testing.T
	issues   []fakeIssue
	mu       sync.Mutex
	requests []fakeRequest
}

// fakeRequest is what the fake GitHub saw in one request.
type fakeRequest struct {
	method        string
	authorization string
	query         string
	variables     map[string]any
}

func (f *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		f.t.Errorf("decode the request: %v", err)
	}
	f.mu.Lock()
	f.requests = append(f.requests, fakeRequest{r.Method, r.Header.Get("Authorization"), payload.Query, payload.Variables})
	f.mu.Unlock()

	query := strings.TrimSpace(payload.Query)
	switch {
	case strings.HasPrefix(query, "query moreSubIssues"):
		f.serveMoreSubIssues(w, payload.Variables)
		return
	case strings.HasPrefix(query, "query closers"):
		f.serveClosers(w, payload.Variables)
		return
	}
	label, _ := payload.Variables["label"].(string)
	var matched []fakeIssue
	for _, issue := range f.issues {
		for _, l := range issue.labels {
			if l == label {
				matched = append(matched, issue)
				break
			}
		}
	}
	start := 0
	if after, ok := payload.Variables["after"].(string); ok {
		start, _ = strconv.Atoi(after)
	}
	end := min(start+2, len(matched))
	var nodes []map[string]any
	for _, issue := range matched[start:end] {
		node := map[string]any{
			"id": "I_" + strconv.Itoa(issue.number), "number": issue.number, "state": issue.state,
			"stateReason": nil, "body": issue.body, "milestone": nil,
			"subIssues": subIssuesPage(issue.subIssues, 0),
		}
		if issue.stateReason != "" {
			node["stateReason"] = issue.stateReason
		}
		if issue.milestone != "" {
			node["milestone"] = map[string]any{"title": issue.milestone}
		}
		nodes = append(nodes, node)
	}
	response := map[string]any{"data": map[string]any{"repository": map[string]any{"issues": map[string]any{
		"pageInfo": map[string]any{"hasNextPage": end < len(matched), "endCursor": strconv.Itoa(end)},
		"nodes":    nodes,
	}}}}
	_ = json.NewEncoder(w).Encode(response)
}

// subIssuesPage returns two sub-issues from start, as one page.
func subIssuesPage(subIssues []fakeSubIssue, start int) map[string]any {
	end := min(start+2, len(subIssues))
	nodes := []map[string]any{}
	for _, subIssue := range subIssues[start:end] {
		node := map[string]any{
			"id": subIssueID(subIssue), "number": subIssue.number, "state": subIssue.state,
			"repository": map[string]any{"nameWithOwner": repositoryOf(subIssue)}, "milestone": nil,
		}
		if subIssue.milestone != "" {
			node["milestone"] = map[string]any{"title": subIssue.milestone}
		}
		nodes = append(nodes, node)
	}
	return map[string]any{
		"pageInfo": map[string]any{"hasNextPage": end < len(subIssues), "endCursor": strconv.Itoa(end)},
		"nodes":    nodes,
	}
}

func repositoryOf(subIssue fakeSubIssue) string {
	if subIssue.repository == "" {
		return "octocat/Hello-World"
	}
	return subIssue.repository
}

func subIssueID(subIssue fakeSubIssue) string {
	return "S_" + repositoryOf(subIssue) + "#" + strconv.Itoa(subIssue.number)
}

// serveMoreSubIssues answers the query of more sub-issues of one issue.
func (f *fakeGitHub) serveMoreSubIssues(w http.ResponseWriter, variables map[string]any) {
	id, _ := variables["id"].(string)
	start := 0
	if after, ok := variables["after"].(string); ok {
		start, _ = strconv.Atoi(after)
	}
	for _, issue := range f.issues {
		if "I_"+strconv.Itoa(issue.number) == id {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"node": map[string]any{
				"subIssues": subIssuesPage(issue.subIssues, start),
			}}})
			return
		}
	}
	f.t.Errorf("more sub-issues of an unknown issue %q", id)
}

// serveClosers answers the query of the closers with the last closed event of
// each sub-issue.
func (f *fakeGitHub) serveClosers(w http.ResponseWriter, variables map[string]any) {
	ids, _ := variables["ids"].([]any)
	byID := map[string]fakeSubIssue{}
	for _, issue := range f.issues {
		for _, subIssue := range issue.subIssues {
			byID[subIssueID(subIssue)] = subIssue
		}
	}
	nodes := []any{}
	for _, value := range ids {
		subIssue, ok := byID[value.(string)]
		if !ok || subIssue.state != "CLOSED" {
			f.t.Errorf("got the closer of %v; want only closed sub-issues", value)
			nodes = append(nodes, nil)
			continue
		}
		events := []map[string]any{}
		switch {
		case subIssue.pullRequest != 0:
			events = append(events, map[string]any{"closer": map[string]any{
				"number": subIssue.pullRequest, "repository": map[string]any{"nameWithOwner": "octocat/Hello-World"},
			}})
		case subIssue.byCommit:
			events = append(events, map[string]any{"closer": map[string]any{}})
		}
		nodes = append(nodes, map[string]any{"id": value, "timelineItems": map[string]any{"nodes": events}})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"nodes": nodes}})
}

// writeRepository writes the settings and one requirement document into a new
// temporary directory and returns it.
func writeRepository(t *testing.T, settings, document string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "ja", "requirements")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".cumin-trace.toml"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

const settingsOfTest = "prefix = \"PC\"\nareas = [\"TASK\"]\n"

const documentOfTest = "# Tasks\n\n" +
	"### `PC-TASK-001` One\n\n- 検証: `test` It works\n\n---\n\n" +
	"### `PC-TASK-002` Two\n\n- 検証: `test` It works\n\n---\n\n" +
	"### `PC-TASK-003` Three\n\n- 検証: `test` It works\n\n---\n\n" +
	"### `PC-TASK-004` Four\n\n- 状態: 廃止\n- 検証: `test` It works\n"

// readAtOfTest is the fixed clock of the tests.
func readAtOfTest() time.Time {
	return time.Date(2026, 10, 20, 18, 0, 0, 0, time.FixedZone("JST", 9*60*60))
}

// runTrace runs the trace against the fake GitHub and returns the exit
// values and the outputs.
func runTrace(t *testing.T, root string, github *fakeGitHub) (int, error, string, string) {
	t.Helper()
	server := httptest.NewServer(github)
	defer server.Close()
	env := Env{Token: token, Repository: "octocat/Hello-World", GraphQLURL: server.URL, Now: readAtOfTest}
	var stdout, stderr bytes.Buffer
	count, err := Run(root, env, &stdout, &stderr)
	return count, err, stdout.String(), stderr.String()
}

// decode decodes the JSON of the trace.
func decode(t *testing.T, stdout string) Output {
	t.Helper()
	var output Output
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode the JSON: %v\n%s", err, stdout)
	}
	return output
}

// issueNumbers returns the numbers of the requirement issues of each
// requirement.
func issueNumbers(output Output) map[string][]int {
	numbers := map[string][]int{}
	for _, req := range output.Requirements {
		numbers[req.ID] = []int{}
		for _, issue := range req.RequirementIssues {
			numbers[req.ID] = append(numbers[req.ID], issue.Number)
		}
	}
	return numbers
}

// CTR-TRC-001: only the issues with the label become requirement issues,
// open and closed, across two pages. The label comes from the settings.
func TestOnlyIssuesWithTheLabelBecomeRequirementIssuesAcrossPages(t *testing.T) {
	body := "## Requirements\n\n- PC-TASK-001\n"
	github := &fakeGitHub{t: t, issues: []fakeIssue{
		{number: 1, labels: []string{"req"}, state: "OPEN", body: body},
		{number: 2, labels: []string{"bug"}, state: "OPEN", body: body},
		{number: 3, labels: []string{"req"}, state: "CLOSED", stateReason: "COMPLETED", body: body},
		{number: 4, labels: []string{"req", "bug"}, state: "CLOSED", stateReason: "NOT_PLANNED", body: body},
		{number: 5, state: "OPEN", body: body},
	}}
	root := writeRepository(t, settingsOfTest+"requirement_label = \"req\"\n", documentOfTest)

	_, err, stdout, _ := runTrace(t, root, github)

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := issueNumbers(decode(t, stdout))["PC-TASK-001"]; !equalInts(got, []int{1, 3, 4}) {
		t.Errorf("got the issues %v of PC-TASK-001; want [1 3 4]", got)
	}
	if len(github.requests) != 2 {
		t.Errorf("got %d requests; want 2, one for each page", len(github.requests))
	}
}

// CTR-TRC-001: without requirement_label, the label is cumin/type/requirement.
func TestTheDefaultLabelIsTheRequirementLabelOfCumin(t *testing.T) {
	github := &fakeGitHub{t: t, issues: []fakeIssue{
		{number: 1, labels: []string{"cumin/type/requirement"}, state: "OPEN", body: "## Requirements\n\n- PC-TASK-001\n"},
	}}
	root := writeRepository(t, settingsOfTest, documentOfTest)

	_, err, stdout, _ := runTrace(t, root, github)

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := issueNumbers(decode(t, stdout))["PC-TASK-001"]; !equalInts(got, []int{1}) {
		t.Errorf("got the issues %v of PC-TASK-001; want [1]", got)
	}
}

// CTR-TRC-002: an issue that lists two IDs links to both, and two issues
// that list one ID both link to it. A check box with a space after it and a
// code span are allowed. An ID in another section, in the middle of an item,
// in a code block, or right after a check box links to nothing.
func TestAnIssueLinksToEachIDThatItsRequirementsSectionLists(t *testing.T) {
	github := &fakeGitHub{t: t, issues: []fakeIssue{
		{number: 7, labels: []string{"cumin/type/requirement"}, state: "OPEN",
			body: "## Context\n\nPC-TASK-003 is the reason.\n\n## Requirements\n\n- [ ] PC-TASK-001 One\n- [x] `PC-TASK-002` Two\n- see PC-TASK-003\n- [x]PC-TASK-003\n\n```\n- PC-TASK-003\n```\n\n## Notes\n\n- PC-TASK-003\n"},
		{number: 8, labels: []string{"cumin/type/requirement"}, state: "OPEN",
			body: "## Requirements\n- PC-TASK-002\n"},
	}}
	root := writeRepository(t, settingsOfTest, documentOfTest)

	_, err, stdout, _ := runTrace(t, root, github)

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := issueNumbers(decode(t, stdout))
	want := map[string][]int{"PC-TASK-001": {7}, "PC-TASK-002": {7, 8}, "PC-TASK-003": {}, "PC-TASK-004": {}}
	for id, numbers := range want {
		if !equalInts(got[id], numbers) {
			t.Errorf("got the issues %v of %s; want %v", got[id], id, numbers)
		}
	}
}

// CTR-TRC-008: the JSON holds the time of the read and each requirement with
// its progress and its requirement issues: number, state, reason, and
// milestone. The same
// input gives the same JSON.
func TestTheJSONHoldsTheTimeOfTheReadAndTheMilestones(t *testing.T) {
	github := &fakeGitHub{t: t, issues: []fakeIssue{
		{number: 12, labels: []string{"cumin/type/requirement"}, state: "OPEN", milestone: "v0.1",
			body: "## Requirements\n\n- PC-TASK-001\n- PC-TASK-004\n- PC-TASK-099\n"},
		{number: 9, labels: []string{"cumin/type/requirement"}, state: "CLOSED", stateReason: "DUPLICATE",
			body: "## Requirements\n\n- PC-TASK-001\n"},
		{number: 10, labels: []string{"cumin/type/requirement"}, state: "CLOSED",
			body: "## Requirements\n\n- PC-TASK-002\n"},
	}}
	root := writeRepository(t, settingsOfTest, documentOfTest)

	count, err, stdout, stderr := runTrace(t, root, github)
	_, _, again, _ := runTrace(t, root, github)

	want := `{
  "read_at": "2026-10-20T09:00:00Z",
  "repository": "octocat/Hello-World",
  "requirements": [
    {
      "id": "PC-TASK-001",
      "deprecated": false,
      "progress": "in_progress",
      "children": null,
      "requirement_issues": [
        {
          "number": 9,
          "state": "closed",
          "state_reason": "not_planned",
          "milestone": null,
          "sub_issues": []
        },
        {
          "number": 12,
          "state": "open",
          "state_reason": null,
          "milestone": "v0.1",
          "sub_issues": []
        }
      ],
      "implementation_issues": {
        "closed": 0,
        "total": 0
      }
    },
    {
      "id": "PC-TASK-002",
      "deprecated": false,
      "progress": "done",
      "children": null,
      "requirement_issues": [
        {
          "number": 10,
          "state": "closed",
          "state_reason": "completed",
          "milestone": null,
          "sub_issues": []
        }
      ],
      "implementation_issues": {
        "closed": 0,
        "total": 0
      }
    },
    {
      "id": "PC-TASK-003",
      "deprecated": false,
      "progress": "not_filed",
      "children": null,
      "requirement_issues": [],
      "implementation_issues": {
        "closed": 0,
        "total": 0
      }
    },
    {
      "id": "PC-TASK-004",
      "deprecated": true,
      "progress": null,
      "children": null,
      "requirement_issues": [
        {
          "number": 12,
          "state": "open",
          "state_reason": null,
          "milestone": "v0.1",
          "sub_issues": []
        }
      ],
      "implementation_issues": {
        "closed": 0,
        "total": 0
      }
    }
  ]
}
`
	if count != 0 || err != nil || stderr != "" {
		t.Fatalf("got count %d, error %v, stderr %q; want 0, no error, and no stderr", count, err, stderr)
	}
	if stdout != want {
		t.Errorf("got the JSON\n%s\nwant\n%s", stdout, want)
	}
	if again != stdout {
		t.Errorf("got another JSON on the second run\n%s", again)
	}
}

// progressOf returns the progress of each requirement, or "null".
func progressOf(output Output) map[string]string {
	progress := map[string]string{}
	for _, req := range output.Requirements {
		progress[req.ID] = "null"
		if req.Progress != nil {
			progress[req.ID] = string(*req.Progress)
		}
	}
	return progress
}

// requirementIssue returns an issue with the requirement label that lists
// the IDs in its "## Requirements" section.
func requirementIssue(number int, state, stateReason string, ids ...string) fakeIssue {
	body := "## Requirements\n\n"
	for _, id := range ids {
		body += "- " + id + "\n"
	}
	return fakeIssue{number: number, labels: []string{"cumin/type/requirement"}, state: state, stateReason: stateReason, body: body}
}

// runProgress runs the trace on documentOfTest and returns the progress of
// each requirement.
func runProgress(t *testing.T, issues ...fakeIssue) map[string]string {
	t.Helper()
	root := writeRepository(t, settingsOfTest, documentOfTest)
	_, err, stdout, _ := runTrace(t, root, &fakeGitHub{t: t, issues: issues})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return progressOf(decode(t, stdout))
}

// CTR-TRC-004.1: a requirement without requirement issues, and one with only
// issues closed as not planned or as a duplicate, are not filed.
func TestARequirementWithoutAnOpenOrCompletedIssueIsNotFiled(t *testing.T) {
	got := runProgress(t,
		requirementIssue(1, "CLOSED", "NOT_PLANNED", "PC-TASK-002"),
		requirementIssue(2, "CLOSED", "DUPLICATE", "PC-TASK-002"),
	)

	if got["PC-TASK-001"] != "not_filed" || got["PC-TASK-002"] != "not_filed" {
		t.Errorf("got the progress %q of PC-TASK-001 and %q of PC-TASK-002; want not_filed for both", got["PC-TASK-001"], got["PC-TASK-002"])
	}
}

// CTR-TRC-004.2: a requirement with an open requirement issue is in
// progress, also with another issue closed as completed.
func TestARequirementWithAnOpenIssueIsInProgress(t *testing.T) {
	got := runProgress(t,
		requirementIssue(1, "CLOSED", "COMPLETED", "PC-TASK-001", "PC-TASK-002"),
		requirementIssue(2, "OPEN", "", "PC-TASK-001"),
		requirementIssue(3, "OPEN", "", "PC-TASK-002"),
		requirementIssue(4, "CLOSED", "NOT_PLANNED", "PC-TASK-002"),
	)

	if got["PC-TASK-001"] != "in_progress" || got["PC-TASK-002"] != "in_progress" {
		t.Errorf("got the progress %q of PC-TASK-001 and %q of PC-TASK-002; want in_progress for both", got["PC-TASK-001"], got["PC-TASK-002"])
	}
}

// CTR-TRC-004.3: a requirement whose requirement issues are all closed, at
// least one as completed, is done. A closed issue with no reason is
// completed.
func TestARequirementWithOnlyClosedIssuesAndOneCompletedIsDone(t *testing.T) {
	got := runProgress(t,
		requirementIssue(1, "CLOSED", "COMPLETED", "PC-TASK-001"),
		requirementIssue(2, "CLOSED", "COMPLETED", "PC-TASK-001"),
		requirementIssue(3, "CLOSED", "", "PC-TASK-002"),
		requirementIssue(4, "CLOSED", "NOT_PLANNED", "PC-TASK-002"),
	)

	if got["PC-TASK-001"] != "done" || got["PC-TASK-002"] != "done" {
		t.Errorf("got the progress %q of PC-TASK-001 and %q of PC-TASK-002; want done for both", got["PC-TASK-001"], got["PC-TASK-002"])
	}
}

// CTR-TRC-005: a parent has no progress of its own and counts its children
// by progress, without the deprecated children. A deprecated requirement has
// no progress and no counts, and a child of a deprecated parent still has its
// own progress.
func TestAParentCountsItsChildrenByProgress(t *testing.T) {
	document := "# Tasks\n\n" +
		"### `PC-TASK-001` Parent\n\n- 検証: `test` It works\n\n" +
		"#### `PC-TASK-001.1` Not filed\n\n- 検証: `test` It works\n\n" +
		"#### `PC-TASK-001.2` In progress\n\n- 検証: `test` It works\n\n" +
		"#### `PC-TASK-001.3` Done\n\n- 検証: `test` It works\n\n" +
		"#### `PC-TASK-001.4` Done too\n\n- 検証: `test` It works\n\n" +
		"#### `PC-TASK-001.5` Deprecated\n\n- 状態: 廃止\n- 検証: `test` It works\n\n---\n\n" +
		"### `PC-TASK-002` Deprecated parent\n\n- 状態: 廃止\n- 検証: `test` It works\n\n" +
		"#### `PC-TASK-002.1` Child of a deprecated parent\n\n- 検証: `test` It works\n"
	github := &fakeGitHub{t: t, issues: []fakeIssue{
		requirementIssue(1, "OPEN", "", "PC-TASK-001", "PC-TASK-001.2", "PC-TASK-002"),
		requirementIssue(2, "CLOSED", "COMPLETED", "PC-TASK-001.3", "PC-TASK-001.4", "PC-TASK-002.1"),
		requirementIssue(3, "OPEN", "", "PC-TASK-001.5"),
	}}
	root := writeRepository(t, settingsOfTest, document)

	count, err, stdout, stderr := runTrace(t, root, github)

	if count != 0 || err != nil || stderr != "" {
		t.Fatalf("got count %d, error %v, stderr %q; want 0, no error, and no stderr", count, err, stderr)
	}
	output := decode(t, stdout)
	got := map[string]string{}
	for _, req := range output.Requirements {
		progress, children := "null", "null"
		if req.Progress != nil {
			progress = string(*req.Progress)
		}
		if req.Children != nil {
			encoded, _ := json.Marshal(req.Children)
			children = string(encoded)
		}
		got[req.ID] = progress + " " + children
	}
	want := map[string]string{
		"PC-TASK-001":   `null {"not_filed":1,"in_progress":1,"done":2,"total":4}`,
		"PC-TASK-001.1": "not_filed null",
		"PC-TASK-001.2": "in_progress null",
		"PC-TASK-001.3": "done null",
		"PC-TASK-001.4": "done null",
		"PC-TASK-001.5": "null null",
		"PC-TASK-002":   "null null",
		"PC-TASK-002.1": "done null",
	}
	if len(got) != len(want) {
		t.Errorf("got %d requirements; want %d", len(got), len(want))
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("got %s for %s; want %s", got[id], id, w)
		}
	}
}

// CTR-TRC-007: the token goes only into the Authorization header. Every
// request is a POST of a query, and no query is a mutation. No output and no
// error holds the token, also when GitHub fails.
func TestTheTokenStaysInTheHeaderAndTheClientSendsOnlyQueries(t *testing.T) {
	github := &fakeGitHub{t: t, issues: []fakeIssue{
		{number: 1, labels: []string{"cumin/type/requirement"}, state: "OPEN", body: "## Requirements\n\n- PC-TASK-001\n"},
		{number: 2, labels: []string{"cumin/type/requirement"}, state: "OPEN", body: "## Requirements\n\n- PC-TASK-001\n"},
		{number: 3, labels: []string{"cumin/type/requirement"}, state: "OPEN", body: "## Requirements\n\n- PC-TASK-001\n"},
	}}
	root := writeRepository(t, settingsOfTest, documentOfTest)

	_, err, stdout, stderr := runTrace(t, root, github)

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.Contains(stdout, token) || strings.Contains(stderr, token) {
		t.Errorf("the output holds the token")
	}
	for _, request := range github.requests {
		query := strings.TrimSpace(request.query)
		if request.method != http.MethodPost || !strings.HasPrefix(query, "query ") || strings.Contains(query, "mutation") {
			t.Errorf("got a %s request with the query %q; want a POST of a query", request.method, query)
		}
		if request.authorization != "bearer "+token {
			t.Errorf("got the Authorization header %q; want the token", request.authorization)
		}
		for _, value := range request.variables {
			if s, ok := value.(string); ok && strings.Contains(s, token) {
				t.Errorf("a variable holds the token")
			}
		}
	}

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer failing.Close()
	var out, errOut bytes.Buffer
	_, err = Run(root, Env{Token: token, Repository: "octocat/Hello-World", GraphQLURL: failing.URL, Now: readAtOfTest}, &out, &errOut)
	if err == nil || err.Error() != "GitHub: HTTP 401" || out.Len() != 0 {
		t.Errorf("got error %v and stdout %q; want GitHub: HTTP 401 and no JSON", err, out.String())
	}
}

// The repository comes from GITHUB_REPOSITORY, else from a github.com URL of
// the remote origin. The design decides this rule (docs/ja/designs/trace/command.md).
func TestTheRepositoryComesFromTheEnvironmentOrTheRemote(t *testing.T) {
	tests := []struct {
		env, remote, want, wantErr string
	}{
		{"octocat/Hello-World", "git@github.com:other/repo.git", "octocat/Hello-World", ""},
		{"", "https://github.com/octocat/Hello-World.git", "octocat/Hello-World", ""},
		{"", "git@github.com:octocat/Hello-World.git", "octocat/Hello-World", ""},
		{"", "ssh://git@github.com/octocat/Hello-World.git", "octocat/Hello-World", ""},
		{"", "https://gitlab.com/octocat/Hello-World.git", "", "no repository: set GITHUB_REPOSITORY to <owner>/<name>"},
		{"", "", "", "no repository: set GITHUB_REPOSITORY to <owner>/<name>"},
		{"octocat", "https://github.com/octocat/Hello-World.git", "", `invalid GITHUB_REPOSITORY "octocat": want <owner>/<name>`},
	}
	for _, test := range tests {
		got, err := Repository(test.env, test.remote)
		gotErr := ""
		if err != nil {
			gotErr = err.Error()
		}
		if got != test.want || gotErr != test.wantErr {
			t.Errorf("Repository(%q, %q) = %q, %q; want %q, %q", test.env, test.remote, got, gotErr, test.want, test.wantErr)
		}
	}
}

// CTR-TRC-007: a token in the URL of the remote origin stays out of the
// repository.
func TestATokenInTheRemoteURLStaysOutOfTheRepository(t *testing.T) {
	got, err := Repository("", "https://user:"+token+"@github.com/octocat/Hello-World")

	if got != "octocat/Hello-World" || err != nil {
		t.Errorf("got %q, %v; want octocat/Hello-World and no error", got, err)
	}
}

// CTR-TRC-001: a cursor that does not move stops the reading of the pages
// with an error, instead of asking for the same page again.
func TestACursorThatDoesNotMoveStopsTheReading(t *testing.T) {
	tests := []struct {
		name     string
		cursor   string
		requests int
	}{
		{"an empty cursor", "", 1},
		{"the same cursor", "same", 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				_, _ = w.Write([]byte(`{"data":{"repository":{"issues":{"pageInfo":{"hasNextPage":true,"endCursor":"` + test.cursor + `"},"nodes":[]}}}}`))
			}))
			defer server.Close()
			root := writeRepository(t, settingsOfTest, documentOfTest)
			var stdout, stderr bytes.Buffer

			_, err := Run(root, Env{Token: token, Repository: "octocat/Hello-World", GraphQLURL: server.URL, Now: readAtOfTest}, &stdout, &stderr)

			if err == nil || err.Error() != "GitHub: invalid page cursor" || stdout.Len() != 0 || requests != test.requests {
				t.Errorf("got error %v, stdout %q, %d requests; want GitHub: invalid page cursor, no JSON, and %d requests", err, stdout.String(), requests, test.requests)
			}
		})
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// CTR-TRC-009: a document with one format error stops the trace before any
// GitHub call. The trace writes no JSON and writes the error line to stderr.
func TestAFormatErrorStopsTheTraceBeforeGitHub(t *testing.T) {
	github := &fakeGitHub{t: t, issues: []fakeIssue{
		{number: 1, labels: []string{"cumin/type/requirement"}, state: "OPEN", body: "## Requirements\n\n- PC-TASK-001\n"},
	}}
	root := writeRepository(t, settingsOfTest, documentOfTest+"\n---\n\n### `PC-TASK-5` Wrong\n\n- 検証: `test` It works\n")

	count, err, stdout, stderr := runTrace(t, root, github)

	line := strings.Count(documentOfTest, "\n") + 4
	wantStderr := "docs/ja/requirements/index.md:" + strconv.Itoa(line) + ": PC-TASK-5: invalid ID format: want <prefix>-<area>-<3 digits>, or that with .<child number>\n"
	if count != 1 || err != nil || stdout != "" || stderr != wantStderr || len(github.requests) != 0 {
		t.Errorf("got %d, %v, stdout %q, stderr %q, %d requests; want 1 error, no JSON, the error line, and no request", count, err, stdout, stderr, len(github.requests))
	}
}

// CTR-TRC-003: the trace reads every sub-issue of a requirement issue, across
// two pages, with its repository, state, and milestone. A closed sub-issue
// holds the pull request that closed it, and one that a commit closed holds
// none. A requirement issue without sub-issues has none.
func TestTheTraceReadsTheSubIssuesAndThePullRequestsThatClosedThem(t *testing.T) {
	issue := requirementIssue(7, "OPEN", "", "PC-TASK-001")
	issue.milestone = "v0.1"
	issue.subIssues = []fakeSubIssue{
		{number: 21, state: "OPEN", milestone: "v0.2"},
		{number: 20, state: "CLOSED", milestone: "v0.1", pullRequest: 25},
		{repository: "octocat/Other", number: 3, state: "CLOSED", byCommit: true},
	}
	github := &fakeGitHub{t: t, issues: []fakeIssue{issue, requirementIssue(8, "OPEN", "", "PC-TASK-002")}}
	root := writeRepository(t, settingsOfTest, documentOfTest)

	_, err, stdout, _ := runTrace(t, root, github)

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	output := decode(t, stdout)
	got, _ := json.Marshal(output.Requirements[0].RequirementIssues[0].SubIssues)
	want := `[{"repository":"octocat/Hello-World","number":20,"state":"closed","milestone":"v0.1","closed_by":{"repository":"octocat/Hello-World","number":25}},` +
		`{"repository":"octocat/Hello-World","number":21,"state":"open","milestone":"v0.2","closed_by":null},` +
		`{"repository":"octocat/Other","number":3,"state":"closed","milestone":null,"closed_by":null}]`
	if string(got) != want {
		t.Errorf("got the sub-issues\n%s\nwant\n%s", got, want)
	}
	if got := output.Requirements[1].RequirementIssues[0].SubIssues; got == nil || len(got) != 0 {
		t.Errorf("got the sub-issues %v of the issue 8; want []", got)
	}
	var queries []string
	for _, request := range github.requests {
		name, _, _ := strings.Cut(strings.Fields(request.query)[1], "(")
		queries = append(queries, name)
	}
	if strings.Join(queries, " ") != "requirementIssues moreSubIssues closers" {
		t.Errorf("got the queries %v; want requirementIssues, moreSubIssues, and closers", queries)
	}
}

// CTR-TRC-003: the trace asks for the closer of each closed sub-issue once,
// and asks for no closer when no sub-issue is closed.
func TestTheTraceAsksForTheCloserOfEachClosedSubIssueOnce(t *testing.T) {
	shared := fakeSubIssue{number: 20, state: "CLOSED", pullRequest: 25}
	first := requirementIssue(1, "OPEN", "", "PC-TASK-001")
	first.subIssues = []fakeSubIssue{shared, {number: 21, state: "OPEN"}}
	second := requirementIssue(2, "OPEN", "", "PC-TASK-002")
	second.subIssues = []fakeSubIssue{shared}
	github := &fakeGitHub{t: t, issues: []fakeIssue{first, second}}
	root := writeRepository(t, settingsOfTest, documentOfTest)

	if _, err, _, _ := runTrace(t, root, github); err != nil {
		t.Fatalf("Run: %v", err)
	}
	last := github.requests[len(github.requests)-1]
	if ids, _ := last.variables["ids"].([]any); len(github.requests) != 2 || len(ids) != 1 {
		t.Errorf("got %d requests and the last with the ids %v; want 2 requests and one id", len(github.requests), last.variables["ids"])
	}

	open := requirementIssue(1, "OPEN", "", "PC-TASK-001")
	open.subIssues = []fakeSubIssue{{number: 21, state: "OPEN"}}
	github = &fakeGitHub{t: t, issues: []fakeIssue{open}}
	if _, err, _, _ := runTrace(t, root, github); err != nil || len(github.requests) != 1 {
		t.Errorf("got error %v and %d requests; want no error and 1 request", err, len(github.requests))
	}
}

// CTR-TRC-008: each requirement counts the closed and all implementation
// issues of its requirement issues. A sub-issue of two requirement issues
// counts once, and a sub-issue of an issue closed as not planned counts too.
// A requirement without sub-issues counts 0 of 0.
func TestEachRequirementCountsItsClosedAndAllImplementationIssues(t *testing.T) {
	first := requirementIssue(1, "OPEN", "", "PC-TASK-001")
	first.subIssues = []fakeSubIssue{
		{number: 20, state: "CLOSED", pullRequest: 25},
		{number: 21, state: "OPEN"},
	}
	second := requirementIssue(2, "CLOSED", "NOT_PLANNED", "PC-TASK-001")
	second.subIssues = []fakeSubIssue{
		{number: 20, state: "CLOSED", pullRequest: 25},
		{number: 22, state: "CLOSED"},
	}
	github := &fakeGitHub{t: t, issues: []fakeIssue{first, second, requirementIssue(3, "OPEN", "", "PC-TASK-002")}}
	root := writeRepository(t, settingsOfTest, documentOfTest)

	_, err, stdout, _ := runTrace(t, root, github)

	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := map[string]ImplementationOutput{}
	for _, req := range decode(t, stdout).Requirements {
		got[req.ID] = req.ImplementationIssues
	}
	want := map[string]ImplementationOutput{
		"PC-TASK-001": {Closed: 2, Total: 3},
		"PC-TASK-002": {Closed: 0, Total: 0},
		"PC-TASK-003": {Closed: 0, Total: 0},
		"PC-TASK-004": {Closed: 0, Total: 0},
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("got %+v for %s; want %+v", got[id], id, w)
		}
	}
	for _, request := range github.requests {
		if query := strings.TrimSpace(request.query); !strings.HasPrefix(query, "query ") || strings.Contains(query, "mutation") {
			t.Errorf("got the query %q; want only queries", query)
		}
	}
}
