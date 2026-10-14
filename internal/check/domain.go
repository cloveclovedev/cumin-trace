// Package check is the format check of the requirement documents
// (cumin-trace check).
package check

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cloveclovedev/cumin-trace/internal/requirement"
)

// The numbers of the errors in the table of the design note
// (docs/ja/designs/check.md). They are the third key of the order.
const (
	ruleInvalidID          = 1
	ruleDuplicateID        = 2
	ruleChildOutsideParent = 3
	ruleWrongMethod        = 4
	ruleNoVerification     = 5
	ruleWrongStatus        = 6
	ruleUnknownArea        = 9
	ruleNumberGap          = 10
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
func Find(docs requirement.Documents, form requirement.IDForm, areas []string) []Error {
	var errs []Error
	errs = append(errs, invalidIDs(docs.InvalidHeadings)...)
	errs = append(errs, duplicateIDs(docs.Requirements)...)
	errs = append(errs, childrenOutsideParent(docs.Requirements)...)
	errs = append(errs, wrongMethods(docs.Requirements)...)
	errs = append(errs, missingVerifications(docs.Requirements)...)
	errs = append(errs, wrongStatuses(docs.Requirements, form)...)
	errs = append(errs, unknownAreas(docs.Requirements, areas)...)
	errs = append(errs, numberGaps(docs.Requirements)...)
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

// duplicateIDs reports each place of an ID that two or more requirement
// headings have, and names the other places (CTR-FMT-003).
func duplicateIDs(requirements []requirement.Requirement) []Error {
	places := map[string][]requirement.Requirement{}
	for _, req := range requirements {
		places[req.ID] = append(places[req.ID], req)
	}
	var errs []Error
	for _, req := range requirements {
		if len(places[req.ID]) < 2 {
			continue
		}
		var others []string
		for _, other := range places[req.ID] {
			if other.File != req.File || other.Line != req.Line {
				others = append(others, fmt.Sprintf("%s:%d", other.File, other.Line))
			}
		}
		errs = append(errs, Error{
			File:    req.File,
			Line:    req.Line,
			Rule:    ruleDuplicateID,
			ID:      req.ID,
			Message: "duplicate ID: also at " + strings.Join(others, ", "),
		})
	}
	return errs
}

// childrenOutsideParent reports each child whose heading above is not the
// heading of its parent, or that is not one level deeper than its parent
// (CTR-FMT-004).
func childrenOutsideParent(requirements []requirement.Requirement) []Error {
	var errs []Error
	for _, req := range requirements {
		if req.ParentID == "" {
			continue
		}
		if req.Above.ID == req.ParentID && req.Level == req.Above.Level+1 {
			continue
		}
		errs = append(errs, Error{
			File:    req.File,
			Line:    req.Line,
			Rule:    ruleChildOutsideParent,
			ID:      req.ID,
			Message: fmt.Sprintf("child requirement is not under its parent: put it one level below the heading of %s", req.ParentID),
		})
	}
	return errs
}

// wrongMethods reports each 検証 line that starts with no method of the four
// (CTR-FMT-006).
func wrongMethods(requirements []requirement.Requirement) []Error {
	var errs []Error
	for _, req := range requirements {
		for _, verification := range req.Verifications {
			switch verification.Method {
			case "test", "demo", "inspection", "analysis":
				continue
			}
			errs = append(errs, Error{
				File:    req.File,
				Line:    verification.Line,
				Rule:    ruleWrongMethod,
				ID:      req.ID,
				Message: "verification must start with `test`, `demo`, `inspection`, or `analysis` in a code span",
			})
		}
	}
	return errs
}

// missingVerifications reports each requirement without children that has no
// 検証 line (CTR-FMT-007). A line with a wrong method counts as a line.
func missingVerifications(requirements []requirement.Requirement) []Error {
	parents := map[string]bool{}
	for _, req := range requirements {
		if req.ParentID != "" {
			parents[req.ParentID] = true
		}
	}
	var errs []Error
	for _, req := range requirements {
		if parents[req.ID] || len(req.Verifications) > 0 {
			continue
		}
		errs = append(errs, Error{
			File:    req.File,
			Line:    req.Line,
			Rule:    ruleNoVerification,
			ID:      req.ID,
			Message: "no verification: a requirement without children needs a 検証 line",
		})
	}
	return errs
}

// wrongStatuses reports each 状態 line that says another thing than 廃止,
// with or without a replacement ID (CTR-FMT-008).
func wrongStatuses(requirements []requirement.Requirement, form requirement.IDForm) []Error {
	var errs []Error
	for _, req := range requirements {
		for _, status := range req.Statuses {
			if _, ok := form.Deprecation(status.Value); ok {
				continue
			}
			errs = append(errs, Error{
				File:    req.File,
				Line:    status.Line,
				Rule:    ruleWrongStatus,
				ID:      req.ID,
				Message: "invalid status: want 廃止, or 廃止。<ID> に置き換え",
			})
		}
	}
	return errs
}

// numberGaps reports each gap in the numbers of an area, and in the numbers
// of the children of one parent (CTR-FMT-012). The numbers must be complete
// from 1 to the largest one. The order of the requirements does not matter.
func numberGaps(requirements []requirement.Requirement) []Error {
	// A series is the numbers that follow one base: "<prefix>-<area>-" for
	// the requirements of an area, and "<parent ID>." for the children of a
	// parent. It keeps the first requirement of each number.
	series := map[string]map[int]requirement.Requirement{}
	format := map[string]string{}
	for _, req := range requirements {
		base, form := req.ID[:len(req.ID)-3], "%s%03d"
		if req.ParentID != "" {
			base, form = req.ParentID+".", "%s%d"
		}
		// A number that is too large for an int becomes the largest int.
		number, _ := strconv.Atoi(req.ID[len(base):])
		if series[base] == nil {
			series[base] = map[int]requirement.Requirement{}
			format[base] = form
		}
		if _, ok := series[base][number]; !ok {
			series[base][number] = req
		}
	}
	var errs []Error
	for base, byNumber := range series {
		numbers := make([]int, 0, len(byNumber))
		for number := range byNumber {
			numbers = append(numbers, number)
		}
		sort.Ints(numbers)
		previous := 0
		for _, number := range numbers {
			req := byNumber[number]
			var message string
			switch {
			case number == 0:
				message = "number gap: numbers start at 001"
			case number == previous+2:
				message = fmt.Sprintf("number gap: %s is missing", fmt.Sprintf(format[base], base, number-1))
			case number > previous+2:
				message = fmt.Sprintf("number gap: %s to %s are missing",
					fmt.Sprintf(format[base], base, previous+1), fmt.Sprintf(format[base], base, number-1))
			}
			previous = number
			if message == "" {
				continue
			}
			errs = append(errs, Error{File: req.File, Line: req.Line, Rule: ruleNumberGap, ID: req.ID, Message: message})
		}
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
