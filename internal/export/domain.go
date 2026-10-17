// Package export writes every requirement of the documents as JSON
// (CTR-EXP-001). The trace builds its requirement objects with the same
// function, so that the shape has one place.
package export

import (
	"slices"

	"github.com/cloveclovedev/cumin-trace/internal/requirement"
)

// Output is the whole JSON of the export.
type Output struct {
	Requirements []Requirement `json:"requirements"`
}

// Requirement is one requirement in the JSON, with the keys in the order of
// the example of the export (CTR-EXP-001).
type Requirement struct {
	ID            string         `json:"id"`
	Parent        *string        `json:"parent"`
	Title         string         `json:"title"`
	Text          string         `json:"text"`
	Verifications []Verification `json:"verifications"`
	// Status is "active" or "deprecated".
	Status     string  `json:"status"`
	ReplacedBy *string `json:"replaced_by"`
	File       string  `json:"file"`
	// Line is the line of the heading.
	Line int `json:"line"`
}

// Verification is one 検証 line in the JSON.
type Verification struct {
	Method string `json:"method"`
	How    string `json:"how"`
}

// Build returns the requirements in the order of the areas, and in the order
// of the documents inside one area (CTR-EXP-002). A requirement of an area
// outside areas comes last; the format check reports such an area first.
func Build(requirements []requirement.Requirement, areas []string) Output {
	sorted := Sort(requirements, areas)
	output := Output{Requirements: make([]Requirement, 0, len(sorted))}
	for _, req := range sorted {
		output.Requirements = append(output.Requirements, Of(req))
	}
	return output
}

// Sort returns a copy of the requirements in the order of the areas, and in
// the order of the documents inside one area (CTR-EXP-002). The trace uses the
// same order.
func Sort(requirements []requirement.Requirement, areas []string) []requirement.Requirement {
	sorted := slices.Clone(requirements)
	rank := func(req requirement.Requirement) int {
		if i := slices.Index(areas, requirement.Area(req.ID)); i >= 0 {
			return i
		}
		return len(areas)
	}
	slices.SortStableFunc(sorted, func(a, b requirement.Requirement) int {
		return rank(a) - rank(b)
	})
	return sorted
}

// Of returns the JSON object of one requirement (CTR-EXP-001). An empty list
// is [], and a missing value is null.
func Of(req requirement.Requirement) Requirement {
	verifications := make([]Verification, 0, len(req.Verifications))
	for _, v := range req.Verifications {
		verifications = append(verifications, Verification{Method: v.Method, How: v.Description})
	}
	status := "active"
	if req.Deprecated {
		status = "deprecated"
	}
	return Requirement{
		ID:            req.ID,
		Parent:        nullable(req.ParentID),
		Title:         req.Title,
		Text:          req.Text,
		Verifications: verifications,
		Status:        status,
		ReplacedBy:    nullable(req.ReplacedBy),
		File:          req.File,
		Line:          req.Line,
	}
}

// nullable returns nil for an empty text, so that JSON writes null.
func nullable(text string) *string {
	if text == "" {
		return nil
	}
	return &text
}
