package requirement

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
// the fields of each requirement (CTR-FMT-005). A heading inside a code block
// is no heading for the parser, so it is not read.
func readDocument(docs *Documents, file string, source []byte, prefix string, form IDForm) {
	source = blankFrontmatter(source)
	root := goldmark.New().Parser().Parse(text.NewReader(source))
	// open holds the headings whose range is not at its end, from the highest
	// level down. The range of a heading ends at the next heading of the same
	// or a higher level (CTR-FMT-003). A heading that is no requirement
	// heading with an ID of the right form has an empty ID.
	var open []openHeading
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if list, ok := node.(*ast.List); ok && entering && list.Parent() == root && !list.IsOrdered() {
			// The fields belong to the nearest requirement heading above. A
			// heading whose ID has a wrong form takes no field.
			for i := len(open) - 1; i >= 0; i-- {
				if !open[i].isRequirement {
					continue
				}
				if open[i].ID != "" {
					readFields(&docs.Requirements[open[i].index], list, source, form)
				}
				break
			}
			return ast.WalkContinue, nil
		}
		heading, ok := node.(*ast.Heading)
		if !ok || !entering {
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
		span, ok := heading.FirstChild().(*ast.CodeSpan)
		if !ok {
			return ast.WalkSkipChildren, nil
		}
		id := codeSpanText(span, source)
		if !strings.HasPrefix(id, prefix+"-") {
			return ast.WalkSkipChildren, nil
		}
		line := lineAt(source, heading.Lines().At(0).Start)
		open[len(open)-1].isRequirement = true
		if form.Valid(id) {
			open[len(open)-1].ID = id
			open[len(open)-1].index = len(docs.Requirements)
			docs.Requirements = append(docs.Requirements, Requirement{
				ID:       id,
				ParentID: ParentID(id),
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
}

// readFields adds the fields of a list to the requirement: each item whose
// first line starts with "検証:" or "状態:" (CTR-FMT-005). An item with
// another name is body text.
func readFields(req *Requirement, list *ast.List, source []byte, form IDForm) {
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
			req.Statuses = append(req.Statuses, Status{Value: value, Line: line})
			// The replacement comes from the first line of the right form.
			if replacedBy, ok := form.Deprecation(value); ok && !req.Deprecated {
				req.Deprecated = true
				req.ReplacedBy = replacedBy
			}
		}
	}
}

// codeSpanText returns the content of a code span.
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
