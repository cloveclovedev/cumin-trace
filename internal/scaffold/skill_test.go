package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloveclovedev/cumin-trace/templates"
)

// CTR-TPL-005: after init, the skill directory holds SKILL.md and one
// template for each of the nine kinds of document, with or without an area.
func TestTheSkillHoldsOneTemplateForEachKind(t *testing.T) {
	for _, area := range []string{"TASK", ""} {
		t.Run("area "+area, func(t *testing.T) {
			root := t.TempDir()
			runInit(t, root, "PC", area)

			entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(SkillDir), "templates"))
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, entry := range entries {
				names = append(names, entry.Name())
			}
			want := "architecture.md,area.md,backlog.md,design-detail.md,design-index.md,entry.md,glossary.md,overview.md,technology.md"
			if got := strings.Join(names, ","); got != want {
				t.Errorf("got templates %s, want %s", got, want)
			}
			if skill := readFile(t, root, SkillDir+"/SKILL.md"); !strings.HasPrefix(skill, "---\nname: cumin-trace-docs\n") {
				t.Errorf("SKILL.md does not start with the front matter of the skill:\n%s", skill)
			}
		})
	}
}

// CTR-TPL-005: each of the seven templates of CTR-TPL-001 in the skill has
// the same bytes as the document that init writes.
func TestTheSharedTemplatesOfTheSkillMatchTheDocuments(t *testing.T) {
	root := t.TempDir()
	runInit(t, root, "PC", "TASK")

	documents := map[string]string{
		"entry.md":        "docs/ja/requirements/index.md",
		"overview.md":     "docs/ja/requirements/overview.md",
		"technology.md":   "docs/ja/requirements/technology.md",
		"glossary.md":     "docs/ja/requirements/glossary.md",
		"backlog.md":      "docs/ja/requirements/backlog.md",
		"area.md":         "docs/ja/requirements/task/index.md",
		"architecture.md": "docs/ja/architecture/overview.md",
	}
	for name, document := range documents {
		if readFile(t, root, SkillDir+"/templates/"+name) != readFile(t, root, document) {
			t.Errorf("the template %s differs from %s", name, document)
		}
	}
}

// CTR-TPL-005: SKILL.md gives each instruction of the requirement, and names
// each template with the kind of its form.
func TestTheSkillGivesEachInstruction(t *testing.T) {
	root := t.TempDir()
	runInit(t, root, "PC", "")

	skill := readFile(t, root, SkillDir+"/SKILL.md")

	for _, instruction := range []string{
		"A kind whose form cumin-trace sets: start each new document of that kind from its template.",
		"A recommended kind: when the repository already has a document of that kind, write the new document in the form of the existing one. Start from the template only when the repository has none.",
		"Read the format of the requirement documents",
		"Read the glossary of this repository",
		"Use each term of the glossary exactly as the glossary writes it.",
		"Never use a wording from the column 使わない言い方 (wordings to avoid) of the glossary.",
		"Give a new requirement the sequence number after the largest one of its area. Count the numbers of deprecated requirements too.",
		"Run `cumin-trace check` at the top of the repository.",
		"Fix every error that the check prints.",
		"Write the page to `<requirements_dir>/<area in lower case>/index.md`.",
		"Add a row for the new area to the table of areas on the entry page, `<requirements_dir>/index.md`.",
		"Replace the area of the template with the new area in every ID.",
		"Draw the diagram of the area as `overview.puml` in the same directory. Render it to `overview.svg` in that directory.",
	} {
		if !strings.Contains(skill, instruction) {
			t.Errorf("SKILL.md does not hold the instruction %q", instruction)
		}
	}
	rows := map[string]string{
		"entry.md": "Set by cumin-trace", "overview.md": "Set by cumin-trace", "area.md": "Set by cumin-trace",
		"glossary.md": "Set by cumin-trace", "backlog.md": "Set by cumin-trace",
		"technology.md": "Recommended", "architecture.md": "Recommended",
		"design-index.md": "Recommended", "design-detail.md": "Recommended",
	}
	for _, line := range strings.Split(skill, "\n") {
		for name, form := range rows {
			if strings.Contains(line, "`templates/"+name+"`") {
				if !strings.HasSuffix(line, "| "+form+" |") {
					t.Errorf("the row of %s does not say %q: %s", name, form, line)
				}
				delete(rows, name)
			}
		}
	}
	for name := range rows {
		t.Errorf("SKILL.md names no template %s", name)
	}
}

// CTR-TPL-008: the entry of a design document has the headings 要件, 全体像,
// 決めたこと, and 未決事項, and the detail has a place for exact values and a
// place for the verified APIs and libraries.
func TestTheDesignTemplatesHaveTheirPlaces(t *testing.T) {
	root := t.TempDir()
	runInit(t, root, "PC", "TASK")

	entry := readFile(t, root, SkillDir+"/templates/design-index.md")
	if got := strings.Join(headings(entry), ","); got != "要件,全体像,決めたこと,未決事項" {
		t.Errorf("got level 2 headings %q of the design entry, want 要件, 全体像, 決めたこと, 未決事項", got)
	}
	for _, place := range []string{"(../../architecture/overview.md)", "- 理由: ", "- 採らなかった案: ", "- 細部: ", "この節ごと消す"} {
		if !strings.Contains(entry, place) {
			t.Errorf("the design entry has no %q:\n%s", place, entry)
		}
	}

	detail := readFile(t, root, SkillDir+"/templates/design-detail.md")
	if got := strings.Join(headings(detail), ","); got != "決まり,確かめたこと" {
		t.Errorf("got level 2 headings %q of the design detail, want 決まり, 確かめたこと", got)
	}
	for _, place := range []string{"- 版: ", "- 公式ページ: ", "- 手元の実行: ", "- 未確認: "} {
		if !strings.Contains(detail, place) {
			t.Errorf("the design detail has no %q:\n%s", place, detail)
		}
	}
}

// CTR-TPL-005: no embedded path holds .claude, CLAUDE.md, or
// .github/workflows, so that an agent can change the text of the skill in
// this repository (docs/ja/designs/templates/files.md).
func TestNoEmbeddedPathIsAProtectedPath(t *testing.T) {
	err := fs.WalkDir(templates.Files, ".", func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		lower := strings.ToLower(path)
		if strings.Contains(lower, ".claude") || strings.HasSuffix(lower, "claude.md") || strings.Contains(lower, ".github/workflows") {
			t.Errorf("the embedded path %s is a protected path", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
