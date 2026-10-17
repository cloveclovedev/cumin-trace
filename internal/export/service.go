package export

import (
	"encoding/json"
	"io"

	"github.com/cloveclovedev/cumin-trace/internal/check"
	"github.com/cloveclovedev/cumin-trace/internal/core"
	"github.com/cloveclovedev/cumin-trace/internal/requirement"
)

// Run reads the requirement documents of the repository at root and writes
// every requirement as JSON to stdout (CTR-EXP-001). On a format error, it
// writes the error lines to stderr and no JSON (CTR-EXP-003). The returned
// number is the number of the format errors. The returned error says that the
// export cannot run.
func Run(root string, stdout, stderr io.Writer) (int, error) {
	settings, err := core.LoadSettings(root)
	if err != nil {
		return 0, err
	}
	docs, err := requirement.Read(root, settings.RequirementsDir, settings.Prefix)
	if err != nil {
		return 0, err
	}
	form := requirement.NewIDForm(settings.Prefix)
	if errs := check.Find(docs, form, settings.Areas); len(errs) > 0 {
		if err := check.WriteErrors(stderr, errs); err != nil {
			return 0, err
		}
		return len(errs), nil
	}

	encoded, err := json.MarshalIndent(Build(docs.Requirements, settings.Areas), "", "  ")
	if err != nil {
		return 0, err
	}
	_, err = stdout.Write(append(encoded, '\n'))
	return 0, err
}
