package requirement

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// readDocumentOfTest writes one requirement document into a new temporary
// directory, reads it, and returns the requirements.
func readDocumentOfTest(t *testing.T, document string) []Requirement {
	t.Helper()
	return readDocumentsOfTest(t, document).Requirements
}

// readDocumentsOfTest writes one requirement document into a new temporary
// directory, reads it, and returns all that the reading gives.
func readDocumentsOfTest(t *testing.T, document string) Documents {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "requirements"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "requirements", "index.md"), []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	docs, err := Read(root, "requirements", "CTR")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return docs
}

// CTR-FMT-005: both fields are read, for the requirement that holds them. A
// list item with another name is body text.
func TestReadsBothFieldsAndTakesAnotherListItemAsBodyText(t *testing.T) {
	requirements := readDocumentOfTest(t, "## `CTR-FMT-001` A parent\n\n"+
		"- 状態: 廃止\n\n"+
		"### `CTR-FMT-001.1` A child\n\n"+
		"- 備考: `test` A list item with another name\n"+
		"- 検証: `demo` A person sees it\n"+
		"  on two lines\n\n"+
		"1. 検証: `test` In a numbered list\n\n"+
		"- A list item\n"+
		"  - 検証: `test` In a nested list\n\n"+
		"> - 状態: 廃止\n\n"+
		"## `CTR-FMT-2` A wrong ID\n\n"+
		"- 検証: `test` Under a heading with a wrong ID\n")

	if len(requirements) != 2 {
		t.Fatalf("got %d requirements, want 2", len(requirements))
	}
	parent, child := requirements[0], requirements[1]
	if !parent.Deprecated || len(parent.Verifications) != 0 {
		t.Errorf("got the parent deprecated %v with verifications %v, want deprecated with no verification", parent.Deprecated, parent.Verifications)
	}
	want := []Verification{{Method: "demo", Description: "A person sees it", Line: 8}}
	if child.Deprecated || !reflect.DeepEqual(child.Verifications, want) {
		t.Errorf("got the child deprecated %v with verifications %v, want not deprecated with %v", child.Deprecated, child.Verifications, want)
	}
}

// CTR-FMT-006: a requirement with two verifications gives both, each with its
// method and its text.
func TestReadsTwoVerificationsOfOneRequirement(t *testing.T) {
	requirements := readDocumentOfTest(t, "### `CTR-FMT-001` A requirement\n\n"+
		"Body text.\n\n"+
		"- 検証: `test` The second attempt is refused\n"+
		"- 検証: `demo` The button is `disabled` on the page\n\n"+
		"---\n")

	want := []Verification{
		{Method: "test", Description: "The second attempt is refused", Line: 5},
		{Method: "demo", Description: "The button is `disabled` on the page", Line: 6},
	}
	if len(requirements) != 1 || !reflect.DeepEqual(requirements[0].Verifications, want) {
		t.Errorf("got %+v, want one requirement with the verifications %v", requirements, want)
	}
}

// CTR-FMT-006: a verification that starts with no code span has no method,
// and its whole value is the text.
func TestReadsAVerificationWithoutACodeSpanAsTextOnly(t *testing.T) {
	requirements := readDocumentOfTest(t, "### `CTR-FMT-001` A requirement\n\n"+
		"- 検証: test It works\n"+
		"- 検証: `review` A person reads it\n")

	want := []Verification{
		{Method: "", Description: "test It works", Line: 3},
		{Method: "review", Description: "A person reads it", Line: 4},
	}
	if len(requirements) != 1 || !reflect.DeepEqual(requirements[0].Verifications, want) {
		t.Errorf("got %+v, want one requirement with the verifications %v", requirements, want)
	}
}

// CTR-FMT-008: the deprecated state and the replacement ID are read. The ID
// stands in a code span or not. Another text gives no deprecated state.
func TestReadsTheDeprecatedStateAndTheReplacementID(t *testing.T) {
	requirements := readDocumentOfTest(t, "### `CTR-FMT-001` Deprecated\n\n"+
		"- 状態: 廃止\n\n"+
		"### `CTR-FMT-002` Replaced, with the ID in a code span\n\n"+
		"- 状態: 廃止。`CTR-FMT-004` に置き換え\n\n"+
		"### `CTR-FMT-003` Replaced, with the ID outside a code span\n\n"+
		"- 状態: 廃止。CTR-FMT-004.1 に置き換え\n\n"+
		"### `CTR-FMT-004` A wrong state\n\n"+
		"- 状態: 下書き\n\n"+
		"### `CTR-FMT-005` No state\n\n"+
		"### `CTR-FMT-006` Two states\n\n"+
		"- 状態: 下書き\n"+
		"- 状態: 廃止。CTR-FMT-005 に置き換え\n"+
		"- 状態: 廃止\n")

	type state struct {
		Deprecated bool
		ReplacedBy string
		Statuses   []Status
	}
	want := []state{
		{true, "", []Status{{"廃止", 3}}},
		{true, "CTR-FMT-004", []Status{{"廃止。`CTR-FMT-004` に置き換え", 7}}},
		{true, "CTR-FMT-004.1", []Status{{"廃止。CTR-FMT-004.1 に置き換え", 11}}},
		{false, "", []Status{{"下書き", 15}}},
		{false, "", nil},
		{true, "CTR-FMT-005", []Status{{"下書き", 21}, {"廃止。CTR-FMT-005 に置き換え", 22}, {"廃止", 23}}},
	}
	var got []state
	for _, req := range requirements {
		got = append(got, state{req.Deprecated, req.ReplacedBy, req.Statuses})
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

// CTR-FMT-010: an ID of the product outside a requirement heading is a
// reference, in body text, in a table, and in a list item, with and without a
// code span. An ID in a code block, an ID of another product, and a text that
// is no right ID are no reference.
func TestReadsReferencesInTextTablesAndListsButNotInCodeBlocks(t *testing.T) {
	docs := readDocumentsOfTest(t, "## `CTR-FMT-001` A requirement that names CTR-FMT-901\n\n"+
		"Body text names CTR-FMT-002 and `CTR-FMT-003`.\n\n"+
		"| Error | Requirement |\n"+
		"|---|---|\n"+
		"| One | CTR-FMT-004 |\n"+
		"| Two | `CTR-FMT-005.1` |\n\n"+
		"- A list item names CTR-FMT-006\n"+
		"- 検証: `test` It works like `CTR-FMT-007`\n"+
		"- 状態: 廃止。CTR-FMT-902 に置き換え\n\n"+
		"```text\n"+
		"CTR-FMT-903 in a fenced code block\n"+
		"```\n\n"+
		"    CTR-FMT-904 in an indented code block\n\n"+
		"<div>CTR-FMT-905 in an HTML block</div>\n\n"+
		"A [link](https://example.com/CTR-FMT-906) and PC-FMT-907 of another product.\n\n"+
		"Not an ID: CTR-FMT, CTR-FMT-01, CTR-FMT-0012, XCTR-FMT-908, CTR-FMT-003.1.2, CTR-fmt-909.\n\n"+
		"Twice on one line: CTR-FMT-008, CTR-FMT-008. Then CTR-FMT-009.\n\n"+
		"## A grouping heading that names CTR-FMT-010\n")

	var got []string
	for _, ref := range docs.References {
		got = append(got, fmt.Sprintf("%d %s", ref.Line, ref.ID))
	}
	want := []string{
		"3 CTR-FMT-002", "3 CTR-FMT-003",
		"7 CTR-FMT-004", "8 CTR-FMT-005.1",
		"10 CTR-FMT-006", "11 CTR-FMT-007",
		"26 CTR-FMT-008", "26 CTR-FMT-009",
		"28 CTR-FMT-010",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got the references %q\nwant %q", got, want)
	}
}

// CTR-FMT-003: a code span with a space at each end is read as GitHub shows
// it. One space is removed at each end, so the first heading is a requirement
// heading. The second heading keeps a space before its ID, so it is no
// requirement heading.
func TestReadsAHeadingWhoseCodeSpanHasASpaceAtEachEnd(t *testing.T) {
	docs := readDocumentsOfTest(t, "## ` CTR-FMT-001 ` A space at each end\n\n"+
		"- 検証: `test` It works\n\n"+
		"## `  CTR-FMT-002  ` Two spaces at each end\n")

	var ids []string
	for _, req := range docs.Requirements {
		ids = append(ids, req.ID)
	}
	if !reflect.DeepEqual(ids, []string{"CTR-FMT-001"}) || len(docs.Requirements[0].Verifications) != 1 {
		t.Errorf("got the requirements %+v, want only CTR-FMT-001 with its verification", docs.Requirements)
	}
	if len(docs.InvalidHeadings) != 0 {
		t.Errorf("got the invalid headings %+v, want none", docs.InvalidHeadings)
	}
}

// CTR-FMT-001: a document that starts with a UTF-8 byte order mark has its
// frontmatter, and the frontmatter is not read.
func TestReadsTheFrontmatterOfADocumentWithAByteOrderMark(t *testing.T) {
	docs := readDocumentsOfTest(t, "\xef\xbb\xbf---\ntitle: Names CTR-FMT-901\n---\n\n"+
		"## `CTR-FMT-001` A requirement\n")

	if len(docs.References) != 0 {
		t.Errorf("got the references %+v, want none", docs.References)
	}
	if len(docs.Requirements) != 1 || docs.Requirements[0].Line != 5 {
		t.Errorf("got the requirements %+v, want CTR-FMT-001 on line 5", docs.Requirements)
	}
}

// CTR-FMT-003: only a heading at the top level of the document counts. A
// heading inside a block quote ends no range and is no requirement heading.
func TestTakesAHeadingInABlockQuoteAsNoHeading(t *testing.T) {
	docs := readDocumentsOfTest(t, "## `CTR-FMT-001` A parent\n\n"+
		"> ## A quoted heading\n\n"+
		"> ### `CTR-FMT-002` A quoted requirement heading\n\n"+
		"### `CTR-FMT-001.1` A child\n")

	if len(docs.Requirements) != 2 {
		t.Fatalf("got the requirements %+v, want CTR-FMT-001 and CTR-FMT-001.1", docs.Requirements)
	}
	child := docs.Requirements[1]
	if child.ID != "CTR-FMT-001.1" || child.Above != (Heading{ID: "CTR-FMT-001", Level: 2}) {
		t.Errorf("got the child %+v, want CTR-FMT-001.1 with CTR-FMT-001 above", child)
	}
	want := []Reference{{ID: "CTR-FMT-002", File: "requirements/index.md", Line: 5}}
	if !reflect.DeepEqual(docs.References, want) {
		t.Errorf("got the references %+v, want %+v", docs.References, want)
	}
}

// CTR-EXP-001: the title and the text of each requirement are read. The text
// of a parent stops at the heading of its first child. The fields and the
// thematic breaks are left out, and a list item with another name stays.
func TestReadsTheTitleAndTheTextOfEachRequirement(t *testing.T) {
	requirements := readDocumentOfTest(t, "## `CTR-FMT-001`  A parent  \n\n"+
		"The first paragraph.\n\n"+
		"| A | B |\n"+
		"|---|---|\n"+
		"| 1 | 2 |\n\n"+
		"### `CTR-FMT-001.1` A child ###\n\n"+
		"Body text\n"+
		"on two lines.\n\n\n"+
		"- 備考: A list item with another name\n"+
		"- 検証: `test` It works\n"+
		"  on two lines\n"+
		"- Another item\n"+
		"  - A nested item\n\n"+
		"```text\n"+
		"Code\n\n"+
		"- 検証: in a code block\n"+
		"```\n\n"+
		"---\n\n"+
		"## `CTR-FMT-002` Deprecated\n\n"+
		"- 状態: 廃止。\n"+
		"  `CTR-FMT-001` に置き換え\n\n"+
		"It is no longer used.\n\n"+
		"***\n\n"+
		"## A grouping heading\n\n"+
		"Text of no requirement.\n")

	type titleText struct{ ID, Title, Text string }
	want := []titleText{
		{"CTR-FMT-001", "A parent", "The first paragraph.\n\n| A | B |\n|---|---|\n| 1 | 2 |"},
		{"CTR-FMT-001.1", "A child", "Body text\non two lines.\n\n" +
			"- 備考: A list item with another name\n\n" +
			"- Another item\n  - A nested item\n\n" +
			"```text\nCode\n\n- 検証: in a code block\n```"},
		{"CTR-FMT-002", "Deprecated", "It is no longer used."},
	}
	var got []titleText
	for _, req := range requirements {
		got = append(got, titleText{req.ID, req.Title, req.Text})
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// CTR-EXP-001: the title and the text of the requirements in the example of
// the export equal what the reading gives from the documents of this
// repository.
func TestReadsTheTitleAndTheTextOfTheExampleOfTheExport(t *testing.T) {
	page, err := os.ReadFile(filepath.Join("..", "..", "docs", "ja", "requirements", "export", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, block, _ := bytes.Cut(page, []byte("```json\n"))
	block, _, _ = bytes.Cut(block, []byte("```"))
	var example struct {
		Requirements []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Text  string `json:"text"`
		} `json:"requirements"`
	}
	if err := json.Unmarshal(block, &example); err != nil || len(example.Requirements) == 0 {
		t.Fatalf("cannot read the example of the export: %v", err)
	}
	docs, err := Read(filepath.Join("..", ".."), "docs/ja/requirements", "CTR")
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Requirement{}
	for _, req := range docs.Requirements {
		byID[req.ID] = req
	}
	for _, want := range example.Requirements {
		got := byID[want.ID]
		if got.Title != want.Title || got.Text != want.Text {
			t.Errorf("got %s with the title %q and the text %q\nwant the title %q and the text %q", want.ID, got.Title, got.Text, want.Title, want.Text)
		}
	}
}
