package trace

import "github.com/cloveclovedev/cumin-trace/internal/platform"

// fromPlatform turns the issues of GitHub into the model. A closed issue is
// not planned when GitHub says NOT_PLANNED or DUPLICATE. Any other closed
// issue, also one with no reason, is completed. A sub-issue keeps the pull
// request that closed it.
func fromPlatform(issues []platform.Issue) []Issue {
	model := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		model = append(model, Issue{
			Number:     issue.Number,
			Open:       issue.State == "OPEN",
			NotPlanned: issue.State != "OPEN" && (issue.StateReason == "NOT_PLANNED" || issue.StateReason == "DUPLICATE"),
			Milestone:  issue.Milestone,
			Body:       issue.Body,
			SubIssues:  subIssuesFromPlatform(issue.SubIssues),
		})
	}
	return model
}

// subIssuesFromPlatform turns the sub-issues of GitHub into the model.
func subIssuesFromPlatform(subIssues []platform.SubIssue) []SubIssue {
	model := make([]SubIssue, 0, len(subIssues))
	for _, subIssue := range subIssues {
		item := SubIssue{
			Repository: subIssue.Repository,
			Number:     subIssue.Number,
			Open:       subIssue.State == "OPEN",
			Milestone:  subIssue.Milestone,
		}
		if subIssue.ClosedBy != nil {
			item.ClosedBy = &PullRequest{Repository: subIssue.ClosedBy.Repository, Number: subIssue.ClosedBy.Number}
		}
		model = append(model, item)
	}
	return model
}
