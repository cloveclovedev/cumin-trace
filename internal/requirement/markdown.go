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

// readDocument adds the requirement headings of one document to docs. A
// heading inside a code block is no heading for the parser, so it is not read.
func readDocument(docs *Documents, file string, source []byte, prefix string, form IDForm) {
	source = blankFrontmatter(source)
	root := goldmark.New().Parser().Parse(text.NewReader(source))
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		heading, ok := node.(*ast.Heading)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}
		span, ok := heading.FirstChild().(*ast.CodeSpan)
		if !ok {
			return ast.WalkSkipChildren, nil
		}
		id := codeSpanText(span, source)
		if !strings.HasPrefix(id, prefix+"-") {
			return ast.WalkSkipChildren, nil
		}
		line := lineAt(source, heading.Lines().At(0).Start)
		if form.Valid(id) {
			docs.Requirements = append(docs.Requirements, Requirement{ID: id, File: file, Line: line})
		} else {
			docs.InvalidHeadings = append(docs.InvalidHeadings, InvalidHeading{Text: id, File: file, Line: line})
		}
		return ast.WalkSkipChildren, nil
	})
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
