package trace

import "github.com/cloveclovedev/cumin-trace/internal/platform"

// fromPlatform turns the issues of GitHub into the model. A closed issue is
// not planned when GitHub says NOT_PLANNED or DUPLICATE. Any other closed
// issue, also one with no reason, is completed.
func fromPlatform(issues []platform.Issue) []Issue {
	model := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		model = append(model, Issue{
			Number:     issue.Number,
			Open:       issue.State == "OPEN",
			NotPlanned: issue.State != "OPEN" && (issue.StateReason == "NOT_PLANNED" || issue.StateReason == "DUPLICATE"),
			Milestone:  issue.Milestone,
			Body:       issue.Body,
		})
	}
	return model
}
