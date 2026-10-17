// Package requirement holds the model of a requirement and reads the
// requirement documents into it. The check, the export, and the trace use it.
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
	// Title is the source of the heading after the code span of the ID,
	// trimmed (CTR-EXP-001).
	Title string
	// Text is the Markdown source of the blocks at the top level between the
	// heading and the next heading, without the fields and the thematic
	// breaks. The blocks are joined with one blank line (CTR-EXP-001).
	Text string
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
	// Verifications holds the 検証 lines of the requirement, in the order of
	// the document (CTR-FMT-005).
	Verifications []Verification
	// Deprecated is true when a 状態 line of the right form says 廃止
	// (CTR-FMT-008).
	Deprecated bool
	// ReplacedBy is the replacement ID of the first 状態 line of the right
	// form. It is empty when that line names none.
	ReplacedBy string
	// ReplacedByLine is the line of the 状態 line that ReplacedBy comes
	// from. It is 0 when the requirement is not deprecated. Only the check
	// uses it.
	ReplacedByLine int
	// Statuses holds every 状態 line of the requirement. Only the check uses
	// it.
	Statuses []Status
}

// Verification is one 検証 line of a requirement.
type Verification struct {
	// Method is the content of the code span that the value starts with. It
	// is empty when the value starts with no code span (CTR-FMT-006).
	Method string
	// Description is the value after the method.
	Description string
	// Line is the line of the 検証 line. Only the check uses it.
	Line int
}

// Status is one 状態 line of a requirement, as it is written.
type Status struct {
	Value string
	Line  int
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

// Reference is one ID of the product outside a requirement heading
// (CTR-FMT-010). The same ID on one line is one reference.
type Reference struct {
	ID   string
	File string
	Line int
}

// Documents is what the reading of the requirement documents gives. The
// lists are in the order of the files, and in the order of the lines inside
// one file.
type Documents struct {
	Requirements    []Requirement
	InvalidHeadings []InvalidHeading
	References      []Reference
}

// IDForm tells whether a text is a right ID of one product (CTR-FMT-002,
// CTR-FMT-004).
type IDForm struct {
	pattern *regexp.Regexp
	// reference matches an ID anywhere in a text (CTR-FMT-010).
	reference *regexp.Regexp
	// deprecation matches the two right values of a 状態 line (CTR-FMT-008).
	deprecation *regexp.Regexp
}

// NewIDForm returns the form of the IDs of the product with the prefix.
func NewIDForm(prefix string) IDForm {
	id := regexp.QuoteMeta(prefix) + `-[A-Z0-9]+-[0-9]{3}(?:\.[1-9][0-9]*)?`
	return IDForm{
		pattern:     regexp.MustCompile(`\A` + id + `\z`),
		reference:   regexp.MustCompile(id),
		deprecation: regexp.MustCompile(`\A廃止(?:。\s*(?:` + "`(" + id + ")`|(" + id + `))\s*に置き換え)?\z`),
	}
}

// Valid reports whether the whole text is a right ID.
func (f IDForm) Valid(text string) bool {
	return f.pattern.MatchString(text)
}

// References returns the start and the end of each reference in text
// (CTR-FMT-010). The text is the part of source from start to end, so that
// the characters around a reference count also beyond the text. A match is a
// reference when the character before it is no letter, no digit, and no "-",
// when the character after it is no letter and no digit, and when no "." with
// a digit follows it.
func (f IDForm) References(source []byte, start, end int) [][2]int {
	var refs [][2]int
	for _, match := range f.reference.FindAllIndex(source[start:end], -1) {
		from, to := start+match[0], start+match[1]
		if from > 0 && (isAlphanumeric(source[from-1]) || source[from-1] == '-') {
			continue
		}
		if to < len(source) && isAlphanumeric(source[to]) {
			continue
		}
		if to+1 < len(source) && source[to] == '.' && isDigit(source[to+1]) {
			continue
		}
		refs = append(refs, [2]int{from, to})
	}
	return refs
}

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func isAlphanumeric(c byte) bool {
	return isDigit(c) || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// Deprecation reports whether the value of a 状態 line has one of the two
// right forms: "廃止", or "廃止。<ID> に置き換え" (CTR-FMT-008). It returns
// the replacement ID, which is empty for the first form. The ID stands in a
// code span or not.
func (f IDForm) Deprecation(value string) (replacedBy string, ok bool) {
	match := f.deprecation.FindStringSubmatch(value)
	if match == nil {
		return "", false
	}
	return match[1] + match[2], true
}

// ParseVerification splits the value of a 検証 line into the method and the
// description (CTR-FMT-006). The method is the content of the code span that
// the value starts with. Without that code span, the method is empty and the
// whole value is the description.
func ParseVerification(value string) (method, description string) {
	rest := strings.TrimLeft(value, "`")
	ticks := len(value) - len(rest)
	if ticks == 0 {
		return "", value
	}
	// A code span ends at the next run of backticks of the same length.
	for i := 0; i < len(rest); {
		if rest[i] != '`' {
			i++
			continue
		}
		end := i
		for end < len(rest) && rest[end] == '`' {
			end++
		}
		if end-i == ticks {
			return strings.TrimSpace(rest[:i]), strings.TrimSpace(rest[end:])
		}
		i = end
	}
	return "", value
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
