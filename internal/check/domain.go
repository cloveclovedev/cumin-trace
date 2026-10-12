// Package check is the format check of the requirement documents
// (cumin-trace check).
package check

import (
	"fmt"
	"sort"

	"github.com/cloveclovedev/cumin-trace/internal/requirement"
)

// The numbers of the errors in the table of the design note
// (docs/ja/designs/check.md). They are the third key of the order.
const (
	ruleInvalidID   = 1
	ruleUnknownArea = 9
)

// Error is one format error (CTR-CHK-002).
type Error struct {
	File    string
	Line    int
	Rule    int
	ID      string
	Message string
}

// String returns the line that the check prints for the error.
func (e Error) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", e.File, e.Line, e.ID, e.Message)
}

// Find applies every rule to the documents and returns the errors in the
// order of the output.
func Find(docs requirement.Documents, areas []string) []Error {
	var errs []Error
	errs = append(errs, invalidIDs(docs.InvalidHeadings)...)
	errs = append(errs, unknownAreas(docs.Requirements, areas)...)
	sortErrors(errs)
	return errs
}

// invalidIDs reports each requirement heading whose ID has a wrong form
// (CTR-FMT-002).
func invalidIDs(headings []requirement.InvalidHeading) []Error {
	var errs []Error
	for _, heading := range headings {
		errs = append(errs, Error{
			File:    heading.File,
			Line:    heading.Line,
			Rule:    ruleInvalidID,
			ID:      heading.Text,
			Message: "invalid ID format: want <prefix>-<area>-<3 digits>, or that with .<child number>",
		})
	}
	return errs
}

// unknownAreas reports each requirement whose area is not in the settings
// (CTR-FMT-011).
func unknownAreas(requirements []requirement.Requirement, areas []string) []Error {
	allowed := map[string]bool{}
	for _, area := range areas {
		allowed[area] = true
	}
	var errs []Error
	for _, req := range requirements {
		area := requirement.Area(req.ID)
		if allowed[area] {
			continue
		}
		errs = append(errs, Error{
			File:    req.File,
			Line:    req.Line,
			Rule:    ruleUnknownArea,
			ID:      req.ID,
			Message: fmt.Sprintf("area %s is not in .cumin-trace.toml", area),
		})
	}
	return errs
}

// sortErrors orders the errors by file, line, rule, ID, and message, so that
// the same input always gives the same output.
func sortErrors(errs []Error) {
	sort.Slice(errs, func(i, j int) bool {
		a, b := errs[i], errs[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.Message < b.Message
	})
}
