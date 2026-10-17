package requirement

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Read reads every Markdown file under the directory dir of the repository
// root, and finds the requirement headings of the product with the prefix
// (CTR-FMT-001, CTR-FMT-003). The path dir is relative to root, with "/" as
// the separator.
func Read(root, dir, prefix string) (Documents, error) {
	files, err := markdownFiles(root, dir)
	if err != nil {
		return Documents{}, err
	}
	form := NewIDForm(prefix)
	var docs Documents
	for _, file := range files {
		source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return Documents{}, fmt.Errorf("%s: cannot read the file", file)
		}
		readDocument(&docs, file, source, prefix, form)
	}
	return docs, nil
}

// markdownFiles returns the files below dir whose name ends with ".md", at
// any depth, as paths from root in byte order. It follows no symbolic link.
func markdownFiles(root, dir string) ([]string, error) {
	top := filepath.Join(root, filepath.FromSlash(dir))
	info, err := os.Lstat(top)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%s: no such directory", dir)
	}
	var files []string
	err = filepath.WalkDir(top, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: cannot read the directory", dir)
	}
	sort.Strings(files)
	return files, nil
}

// openHeading is a heading whose range is not at its end.
type openHeading struct {
	Heading
	// isRequirement is true for a requirement heading, also for one whose ID
	// has a wrong form.
	isRequirement bool
	// index is the place of the requirement in docs.Requirements. It counts
	// only when the ID is not empty.
	index int
}

// readDocument adds the requirement headings of one document to docs, with
// the fields of each requirement (CTR-FMT-005), and adds the references of
// the document (CTR-FMT-010). A heading inside a code block is no heading for
// the parser, so it is not read. A byte order mark at the start of the
// document is removed before the reading.
func readDocument(docs *Documents, file string, source []byte, prefix string, form IDForm) {
	source = blankFrontmatter(bytes.TrimPrefix(source, []byte("\uFEFF")))
	root := goldmark.New().Parser().Parse(text.NewReader(source))
	// open holds the headings whose range is not at its end, from the highest
	// level down. The range of a heading ends at the next heading of the same
	// or a higher level (CTR-FMT-003). A heading that is no requirement
	// heading with an ID of the right form has an empty ID.
	var open []openHeading
	// statuses holds the list items of the 状態 fields. Their text is no
	// reference: the replacement ID is read with the field (CTR-FMT-010).
	statuses := map[ast.Node]bool{}
	// seen holds the references of the document, so that the same ID on one
	// line is one reference.
	seen := map[Reference]bool{}
	// headings holds the place in docs.Requirements of each requirement
	// heading with an ID of the right form, for the reading of the text.
	headings := map[ast.Node]int{}
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := node.(type) {
		case *ast.List:
			if node.Parent() != root || node.IsOrdered() {
				return ast.WalkContinue, nil
			}
			// The fields belong to the nearest requirement heading above. A
			// heading whose ID has a wrong form takes no field.
			for i := len(open) - 1; i >= 0; i-- {
				if !open[i].isRequirement {
					continue
				}
				if open[i].ID != "" {
					readFields(&docs.Requirements[open[i].index], node, source, form, statuses)
				}
				break
			}
			return ast.WalkContinue, nil
		case *ast.ListItem:
			if statuses[node] {
				return ast.WalkSkipChildren, nil
			}
			return ast.WalkContinue, nil
		case *ast.FencedCodeBlock, *ast.CodeBlock, *ast.HTMLBlock:
			// These blocks hold their lines and no Text child, so skipping
			// their children changes nothing. The case says that they hold
			// no reference.
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			for _, ref := range form.References(source, node.Segment.Start, node.Segment.Stop) {
				reference := Reference{ID: string(source[ref[0]:ref[1]]), File: file, Line: lineAt(source, ref[0])}
				if !seen[reference] {
					seen[reference] = true
					docs.References = append(docs.References, reference)
				}
			}
			return ast.WalkContinue, nil
		}
		// Only a heading at the top level of the document counts. A heading
		// in a block quote or a list item is text that holds references.
		heading, ok := node.(*ast.Heading)
		if !ok || heading.Parent() != root {
			return ast.WalkContinue, nil
		}
		for len(open) > 0 && open[len(open)-1].Level >= heading.Level {
			open = open[:len(open)-1]
		}
		var above Heading
		if len(open) > 0 && open[len(open)-1].ID != "" {
			above = open[len(open)-1].Heading
		}
		open = append(open, openHeading{Heading: Heading{Level: heading.Level}})
		// The text of a heading that is no requirement heading holds
		// references (CTR-FMT-010).
		span, ok := heading.FirstChild().(*ast.CodeSpan)
		if !ok {
			return ast.WalkContinue, nil
		}
		id := codeSpanText(span, source)
		if !strings.HasPrefix(id, prefix+"-") {
			return ast.WalkContinue, nil
		}
		line := lineAt(source, heading.Lines().At(0).Start)
		open[len(open)-1].isRequirement = true
		if form.Valid(id) {
			open[len(open)-1].ID = id
			open[len(open)-1].index = len(docs.Requirements)
			headings[heading] = len(docs.Requirements)
			// The split of a 検証 value also splits the heading into the
			// code span and the title after it.
			_, title := ParseVerification(strings.TrimSpace(string(heading.Lines().Value(source))))
			docs.Requirements = append(docs.Requirements, Requirement{
				ID:       id,
				ParentID: ParentID(id),
				Title:    title,
				File:     file,
				Line:     line,
				Level:    heading.Level,
				Above:    above,
			})
		} else {
			docs.InvalidHeadings = append(docs.InvalidHeadings, InvalidHeading{Text: id, File: file, Line: line})
		}
		return ast.WalkSkipChildren, nil
	})
	readTexts(docs.Requirements, root, source, headings)
}

// readFields adds the fields of a list to the requirement: each item whose
// first line starts with "検証:" or "状態:" (CTR-FMT-005). An item with
// another name is body text. It adds each item of a 状態 field to statuses.
func readFields(req *Requirement, list *ast.List, source []byte, form IDForm, statuses map[ast.Node]bool) {
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		block := item.FirstChild()
		switch block.(type) {
		case *ast.TextBlock, *ast.Paragraph:
		default:
			continue
		}
		if block.Lines().Len() == 0 {
			continue
		}
		first := block.Lines().At(0)
		line := lineAt(source, first.Start)
		if value, ok := strings.CutPrefix(string(first.Value(source)), "検証:"); ok {
			method, description := ParseVerification(strings.TrimSpace(value))
			req.Verifications = append(req.Verifications, Verification{Method: method, Description: description, Line: line})
		}
		if value, ok := strings.CutPrefix(string(first.Value(source)), "状態:"); ok {
			value = strings.TrimSpace(value)
			statuses[item] = true
			req.Statuses = append(req.Statuses, Status{Value: value, Line: line})
			// The replacement comes from the first line of the right form.
			if replacedBy, ok := form.Deprecation(value); ok && !req.Deprecated {
				req.Deprecated = true
				req.ReplacedBy = replacedBy
				req.ReplacedByLine = line
			}
		}
	}
}

// isField reports whether a list item of a list at the top level is a field:
// an item of a list without numbers whose first line starts with "検証:" or
// "状態:" (CTR-FMT-005).
func isField(item ast.Node, source []byte) bool {
	if list, ok := item.Parent().(*ast.List); !ok || list.IsOrdered() {
		return false
	}
	block := item.FirstChild()
	switch block.(type) {
	case *ast.TextBlock, *ast.Paragraph:
	default:
		return false
	}
	if block.Lines().Len() == 0 {
		return false
	}
	line := block.Lines().At(0)
	first := string(line.Value(source))
	return strings.HasPrefix(first, "検証:") || strings.HasPrefix(first, "状態:")
}

// thematicBreak matches a line that is a thematic break in CommonMark.
var thematicBreak = regexp.MustCompile(`\A {0,3}(?:(?:\*[ \t]*){3,}|(?:-[ \t]*){3,}|(?:_[ \t]*){3,})\r?\n?\z`)

// readTexts sets the text of each requirement: the Markdown source of the
// blocks at the top level between its heading and the next heading at the top
// level. The fields and the thematic breaks are left out. The blocks are
// joined with one blank line, and the ends are trimmed. The headings map each
// requirement heading to its place in requirements.
func readTexts(requirements []Requirement, root ast.Node, source []byte, headings map[ast.Node]int) {
	lines := bytes.SplitAfter(source, []byte("\n"))
	blank := func(line int) bool {
		return len(bytes.TrimSpace(lines[line-1])) == 0
	}
	// text returns the source of the lines from and to, both included.
	text := func(from, to int) string {
		return strings.TrimRight(string(bytes.Join(lines[from-1:to], nil)), "\r\n")
	}
	// The start of each block at the top level, except the thematic breaks.
	// A block without a position in the tree, such as an empty heading,
	// starts at the first line after the block before it that is neither
	// blank nor a thematic break.
	var blocks []ast.Node
	var starts []int
	previousLast := 0
	for block := root.FirstChild(); block != nil; block = block.NextSibling() {
		if _, ok := block.(*ast.ThematicBreak); ok {
			continue
		}
		first, last := lineRange(block, source)
		if first == 0 {
			first = previousLast + 1
			for first < len(lines) && (blank(first) || thematicBreak.Match(lines[first-1])) {
				first++
			}
			last = first
		}
		previousLast = last
		blocks = append(blocks, block)
		starts = append(starts, first)
	}
	// end returns the last line of the range from the start of a node to the
	// line before next. Blank lines and thematic breaks at the end are left
	// out, but never a line that holds the content of the node.
	end := func(node ast.Node, next int) int {
		_, last := lineRange(node, source)
		line := next - 1
		for line > last && (blank(line) || thematicBreak.Match(lines[line-1])) {
			line--
		}
		return line
	}
	current := -1
	var parts []string
	finish := func() {
		if current >= 0 {
			requirements[current].Text = strings.TrimSpace(strings.Join(parts, "\n\n"))
		}
		parts = nil
	}
	for i, block := range blocks {
		next := len(lines) + 1
		if i+1 < len(blocks) {
			next = starts[i+1]
		}
		switch block := block.(type) {
		case *ast.Heading:
			finish()
			current = -1
			if index, ok := headings[block]; ok {
				current = index
			}
			continue
		case *ast.List:
			if current < 0 {
				continue
			}
			// A run of items that are no field is one part.
			from, to := 0, 0
			for item := block.FirstChild(); item != nil; item = item.NextSibling() {
				itemNext := next
				if item.NextSibling() != nil {
					itemNext, _ = lineRange(item.NextSibling(), source)
				}
				if isField(item, source) {
					if from > 0 {
						parts = append(parts, text(from, to))
					}
					from = 0
					continue
				}
				if from == 0 {
					from, _ = lineRange(item, source)
				}
				to = end(item, itemNext)
			}
			if from > 0 {
				parts = append(parts, text(from, to))
			}
			continue
		}
		if current >= 0 {
			parts = append(parts, text(starts[i], end(block, next)))
		}
	}
	finish()
}

// lineRange returns the first and the last line of the content of a node, as
// the positions in the tree give them. The opening line of a fenced code
// block counts, but not its closing line. It returns 0 and 0 for a node
// without a position, such as a thematic break.
func lineRange(node ast.Node, source []byte) (first, last int) {
	add := func(start, stop int) {
		from, to := lineAt(source, start), lineAt(source, max(start, stop-1))
		if first == 0 || from < first {
			first = from
		}
		last = max(last, to)
	}
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Type() == ast.TypeBlock && n.Lines().Len() > 0 {
			add(n.Lines().At(0).Start, n.Lines().At(n.Lines().Len()-1).Stop)
		}
		if code, ok := n.(*ast.FencedCodeBlock); ok {
			if code.Info != nil {
				add(code.Info.Segment.Start, code.Info.Segment.Stop)
			} else if code.Lines().Len() > 0 {
				// The opening line is the line before the content.
				opening := lineAt(source, code.Lines().At(0).Start) - 1
				if first == 0 || opening < first {
					first = opening
				}
			}
		}
		if t, ok := n.(*ast.Text); ok {
			add(t.Segment.Start, t.Segment.Stop)
		}
		return ast.WalkContinue, nil
	})
	return first, last
}

// codeSpanText returns the content of a code span, as GitHub shows it. The
// parser has already removed one space at each end when the content starts
// and ends with a space, as CommonMark does.
func codeSpanText(span *ast.CodeSpan, source []byte) string {
	var content strings.Builder
	for child := span.FirstChild(); child != nil; child = child.NextSibling() {
		if segment, ok := child.(*ast.Text); ok {
			content.Write(segment.Segment.Value(source))
		}
	}
	return content.String()
}

// lineAt returns the line of the byte at offset. The first line is 1.
func lineAt(source []byte, offset int) int {
	return 1 + bytes.Count(source[:offset], []byte("\n"))
}

// blankFrontmatter replaces the frontmatter with spaces and keeps its line
// breaks, so that the parser does not read it and the line numbers stay. The
// frontmatter starts at a first line that is only "---", and ends at the next
// line that is only "---" or "...". A document without the closing line has
// no frontmatter.
func blankFrontmatter(source []byte) []byte {
	lines := bytes.SplitAfter(source, []byte("\n"))
	if strings.TrimRight(string(lines[0]), "\r\n") != "---" {
		return source
	}
	end := len(lines[0])
	for _, line := range lines[1:] {
		end += len(line)
		closing := strings.TrimRight(string(line), "\r\n")
		if closing != "---" && closing != "..." {
			continue
		}
		blanked := bytes.Clone(source)
		for i := 0; i < end; i++ {
			if blanked[i] != '\n' {
				blanked[i] = ' '
			}
		}
		return blanked
	}
	return source
}

// ListedIDs returns the IDs that the items of the "## Requirements" sections
// of an issue body start with, each ID once, in the order of the body
// (CTR-TRC-002). An ID in another section, in the middle of an item, or in a
// code block is not listed.
func ListedIDs(body []byte, form IDForm) []string {
	root := goldmark.New().Parser().Parse(text.NewReader(body))
	var ids []string
	seen := map[string]bool{}
	inSection := false
	for block := root.FirstChild(); block != nil; block = block.NextSibling() {
		// Only a heading at the top level starts or ends a section.
		if heading, ok := block.(*ast.Heading); ok {
			if heading.Level <= 2 {
				inSection = heading.Level == 2 && strings.TrimSpace(string(heading.Lines().Value(body))) == "Requirements"
			}
			continue
		}
		if _, ok := block.(*ast.List); !ok || !inSection {
			continue
		}
		_ = ast.Walk(block, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			if _, ok := node.(*ast.ListItem); !ok {
				return ast.WalkContinue, nil
			}
			if id := leadingID(node, body, form); id != "" && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
			return ast.WalkContinue, nil
		})
	}
	return ids
}

// leadingID returns the ID that a list item starts with, or an empty string.
// The item must start with a paragraph. A check box and a code span around
// the ID are allowed.
func leadingID(item ast.Node, source []byte, form IDForm) string {
	block := item.FirstChild()
	switch block.(type) {
	case *ast.TextBlock, *ast.Paragraph:
	default:
		return ""
	}
	if block.Lines().Len() == 0 {
		return ""
	}
	first := block.Lines().At(0)
	start, end := first.Start, first.Stop
	skipSpaces := func() {
		for start < end && (source[start] == ' ' || source[start] == '\t') {
			start++
		}
	}
	skipSpaces()
	// A check box counts only with a space or a tab after it.
	if end-start >= 4 && source[start] == '[' && source[start+2] == ']' &&
		(source[start+1] == ' ' || source[start+1] == 'x' || source[start+1] == 'X') &&
		(source[start+3] == ' ' || source[start+3] == '\t') {
		start += 3
		skipSpaces()
	}
	if start < end && source[start] == '`' {
		start++
	}
	refs := form.References(source, start, end)
	if len(refs) == 0 || refs[0][0] != start {
		return ""
	}
	return string(source[refs[0][0]:refs[0][1]])
}
