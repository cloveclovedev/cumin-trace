package check

import (
	"fmt"
	"io"

	"github.com/cloveclovedev/cumin-trace/internal/core"
	"github.com/cloveclovedev/cumin-trace/internal/requirement"
)

// Run checks the requirement documents of the repository at root, and writes
// one line for each error to stdout. It returns the number of the errors. The
// returned error says that the check cannot run. The only input is the
// settings file and the requirement documents (CTR-CHK-004).
func Run(root string, stdout io.Writer) (int, error) {
	settings, err := core.LoadSettings(root)
	if err != nil {
		return 0, err
	}
	docs, err := requirement.Read(root, settings.RequirementsDir, settings.Prefix)
	if err != nil {
		return 0, err
	}
	errs := Find(docs, requirement.NewIDForm(settings.Prefix), settings.Areas)
	if err := WriteErrors(stdout, errs); err != nil {
		return 0, err
	}
	return len(errs), nil
}

// WriteErrors writes one line for each error to w. The check and the trace
// both use it, so that they write the same lines (CTR-TRC-009).
func WriteErrors(w io.Writer, errs []Error) error {
	for _, e := range errs {
		if _, err := fmt.Fprintln(w, e); err != nil {
			return err
		}
	}
	return nil
}
