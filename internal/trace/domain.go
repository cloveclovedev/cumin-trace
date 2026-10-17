// Package trace links the requirements to the requirement issues on GitHub
// and writes the result as JSON (CTR-TRC-001 to CTR-TRC-009).
package trace

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/cloveclovedev/cumin-trace/internal/requirement"
)

// Issue is one requirement issue.
type Issue struct {
	Number int
	// Open is true for an open issue.
	Open bool
	// NotPlanned is true for an issue that was closed as not planned or as
	// a duplicate. It is false for an open issue.
	NotPlanned bool
	// Milestone is the title of the milestone. It is empty for no milestone.
	Milestone string
	Body      string
}

// Output is the result of the trace, in the form of the JSON (CTR-TRC-008).
type Output struct {
	ReadAt       string              `json:"read_at"`
	Repository   string              `json:"repository"`
	Requirements []RequirementOutput `json:"requirements"`
}

// RequirementOutput is one requirement and its requirement issues.
type RequirementOutput struct {
	ID         string `json:"id"`
	Deprecated bool   `json:"deprecated"`
	// Progress is nil for a parent and for a deprecated requirement.
	Progress *Progress `json:"progress"`
	// Children is nil for a requirement without children and for a
	// deprecated requirement.
	Children          *ChildrenOutput `json:"children"`
	RequirementIssues []IssueOutput   `json:"requirement_issues"`
}

// Progress is how far a requirement has come (CTR-TRC-004).
type Progress string

const (
	NotFiled   Progress = "not_filed"   // CTR-TRC-004.1
	InProgress Progress = "in_progress" // CTR-TRC-004.2
	Done       Progress = "done"        // CTR-TRC-004.3
)

// ChildrenOutput counts the children of a parent that are not deprecated, by
// progress (CTR-TRC-005).
type ChildrenOutput struct {
	NotFiled   int `json:"not_filed"`
	InProgress int `json:"in_progress"`
	Done       int `json:"done"`
	Total      int `json:"total"`
}

// IssueOutput is one requirement issue of a requirement.
type IssueOutput struct {
	Number      int     `json:"number"`
	State       string  `json:"state"`
	StateReason *string `json:"state_reason"`
	Milestone   *string `json:"milestone"`
}

// Link links each requirement to the issues that list its ID in the
// "## Requirements" section (CTR-TRC-002). One issue can list many IDs, and
// many issues can list one ID. The requirements stay in the order of the
// documents, and the issues of a requirement are in the order of their
// numbers. An ID that no requirement has links to nothing. Each requirement
// gets its progress, or the counts of its children (CTR-TRC-004, CTR-TRC-005).
func Link(requirements []requirement.Requirement, issues []Issue, form requirement.IDForm, readAt, repository string) Output {
	byID := map[string]int{}
	linked := make([][]Issue, len(requirements))
	output := Output{ReadAt: readAt, Repository: repository, Requirements: make([]RequirementOutput, len(requirements))}
	for i, req := range requirements {
		byID[req.ID] = i
		output.Requirements[i] = RequirementOutput{ID: req.ID, Deprecated: req.Deprecated, RequirementIssues: []IssueOutput{}}
	}
	sorted := append([]Issue(nil), issues...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Number < sorted[j].Number })
	for _, issue := range sorted {
		for _, id := range requirement.ListedIDs([]byte(issue.Body), form) {
			i, ok := byID[id]
			if !ok {
				continue
			}
			linked[i] = append(linked[i], issue)
			output.Requirements[i].RequirementIssues = append(output.Requirements[i].RequirementIssues, issueOutput(issue))
		}
	}
	setProgress(requirements, linked, output.Requirements)
	return output
}

// ProgressOf returns the progress of a requirement without children from its
// requirement issues (CTR-TRC-004). One open issue makes it in progress. Else
// one issue closed as completed makes it done. Else it is not filed.
func ProgressOf(issues []Issue) Progress {
	progress := NotFiled
	for _, issue := range issues {
		switch {
		case issue.Open:
			return InProgress
		case !issue.NotPlanned:
			progress = Done
		}
	}
	return progress
}

// setProgress sets the progress of each requirement that is not deprecated
// and has no children. It sets the counts of the children of each parent that
// is not deprecated, and leaves out the deprecated children (CTR-TRC-005).
func setProgress(requirements []requirement.Requirement, linked [][]Issue, outputs []RequirementOutput) {
	children := map[string][]int{}
	for i, req := range requirements {
		if req.ParentID != "" {
			children[req.ParentID] = append(children[req.ParentID], i)
		}
	}
	for i, req := range requirements {
		if req.Deprecated {
			continue
		}
		if _, ok := children[req.ID]; !ok {
			progress := ProgressOf(linked[i])
			outputs[i].Progress = &progress
		}
	}
	for i, req := range requirements {
		indexes, ok := children[req.ID]
		if req.Deprecated || !ok {
			continue
		}
		counts := &ChildrenOutput{}
		for _, child := range indexes {
			if outputs[child].Progress == nil {
				continue
			}
			switch *outputs[child].Progress {
			case NotFiled:
				counts.NotFiled++
			case InProgress:
				counts.InProgress++
			case Done:
				counts.Done++
			}
			counts.Total++
		}
		outputs[i].Children = counts
	}
}

// issueOutput returns the JSON form of an issue.
func issueOutput(issue Issue) IssueOutput {
	out := IssueOutput{Number: issue.Number, State: "open"}
	if !issue.Open {
		out.State = "closed"
		reason := "completed"
		if issue.NotPlanned {
			reason = "not_planned"
		}
		out.StateReason = &reason
	}
	if issue.Milestone != "" {
		milestone := issue.Milestone
		out.Milestone = &milestone
	}
	return out
}

var (
	repositoryPattern = regexp.MustCompile(`\A[^/]+/[^/]+\z`)
	remotePatterns    = []*regexp.Regexp{
		regexp.MustCompile(`\Ahttps://(?:[^@/]*@)?github\.com/([^/]+/[^/]+?)(?:\.git)?/?\z`),
		regexp.MustCompile(`\A[^@/:]+@github\.com:([^/]+/[^/]+?)(?:\.git)?/?\z`),
		regexp.MustCompile(`\Assh://(?:[^@/]*@)?github\.com/([^/]+/[^/]+?)(?:\.git)?/?\z`),
	}
)

// ValidRepository reports whether the text has the form <owner>/<name>: one
// "/" with a non-empty text at each side.
func ValidRepository(text string) bool {
	return repositoryPattern.MatchString(text)
}

// RepositoryFromRemote returns <owner>/<name> of a github.com URL of the
// remote origin, in the https, scp, or ssh form. It returns an empty string
// for any other URL. The URL itself never goes into the result.
func RepositoryFromRemote(url string) string {
	url = strings.TrimSpace(url)
	for _, pattern := range remotePatterns {
		if match := pattern.FindStringSubmatch(url); match != nil && ValidRepository(match[1]) {
			return match[1]
		}
	}
	return ""
}

// Repository chooses the repository to read: GITHUB_REPOSITORY when it is
// set, else the URL of the remote origin.
func Repository(env, remote string) (string, error) {
	if env != "" {
		if !ValidRepository(env) {
			return "", fmt.Errorf("invalid GITHUB_REPOSITORY %q: want <owner>/<name>", env)
		}
		return env, nil
	}
	if repository := RepositoryFromRemote(remote); repository != "" {
		return repository, nil
	}
	return "", fmt.Errorf("no repository: set GITHUB_REPOSITORY to <owner>/<name>")
}
