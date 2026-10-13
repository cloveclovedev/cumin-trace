// Package requirement holds the model of a requirement and reads the
// requirement documents into it. The check uses it today; the export and the
// trace use it later.
package requirement

import (
	"regexp"
	"strings"
)

// Requirement is one requirement heading with an ID of the right form.
type Requirement struct {
	// ID is the content of the code span of the heading.
	ID string
	// ParentID is the ID of the parent of a child. It is empty for a
	// requirement that is no child.
	ParentID string
	// File is the path from the top of the repository, with "/" as the separator.
	File string
	// Line is the line of the heading. The first line of a file is 1.
	Line int
	// Level is the level of the heading.
	Level int
	// Above is the heading above, when that heading is a requirement heading
	// with an ID of the right form. Otherwise it is empty. Only the check
	// uses it.
	Above Heading
}

// Heading is a requirement heading with an ID of the right form, as another
// requirement sees it above itself.
type Heading struct {
	ID    string
	Level int
}

// InvalidHeading is one requirement heading whose ID has a wrong form.
type InvalidHeading struct {
	// Text is the content of the code span of the heading.
	Text string
	File string
	Line int
}

// Documents is what the reading of the requirement documents gives. Both
// lists are in the order of the files, and in the order of the lines inside
// one file.
type Documents struct {
	Requirements    []Requirement
	InvalidHeadings []InvalidHeading
}

// IDForm tells whether a text is a right ID of one product (CTR-FMT-002,
// CTR-FMT-004).
type IDForm struct {
	pattern *regexp.Regexp
}

// NewIDForm returns the form of the IDs of the product with the prefix.
func NewIDForm(prefix string) IDForm {
	return IDForm{pattern: regexp.MustCompile(`\A` + regexp.QuoteMeta(prefix) + `-[A-Z0-9]+-[0-9]{3}(\.[1-9][0-9]*)?\z`)}
}

// Valid reports whether the whole text is a right ID.
func (f IDForm) Valid(text string) bool {
	return f.pattern.MatchString(text)
}

// Area returns the area of a right ID: the part between the prefix and the
// number.
func Area(id string) string {
	parts := strings.Split(id, "-")
	if len(parts) < 3 {
		return ""
	}
	return parts[1]
}

// ParentID returns the ID of the parent of a right ID of a child: the ID
// without ".<child number>" (CTR-FMT-004). It returns "" for an ID of a
// requirement that is no child.
func ParentID(id string) string {
	parent, _, found := strings.Cut(id, ".")
	if !found {
		return ""
	}
	return parent
}
