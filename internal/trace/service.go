package trace

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/cloveclovedev/cumin-trace/internal/check"
	"github.com/cloveclovedev/cumin-trace/internal/core"
	"github.com/cloveclovedev/cumin-trace/internal/platform"
	"github.com/cloveclovedev/cumin-trace/internal/requirement"
)

// Env is what the command takes from its environment. Only main.go reads the
// environment variables and runs git.
type Env struct {
	Token      string           // GITHUB_TOKEN
	Repository string           // GITHUB_REPOSITORY
	Remote     string           // the URL of `git remote get-url origin`, or empty
	GraphQLURL string           // GITHUB_GRAPHQL_URL. Empty means https://api.github.com/graphql
	Now        func() time.Time // the clock for read_at
}

// Run reads the requirement documents of the repository at root and the
// requirement issues on GitHub, links them, and writes the JSON to stdout
// (CTR-TRC-008). It writes the JSON at once, only after every read. On a
// format error, it writes the error lines to stderr and reads no GitHub
// (CTR-TRC-009). The returned number is the number of the format errors. The returned error says
// that the trace cannot run.
func Run(root string, env Env, stdout, stderr io.Writer) (int, error) {
	settings, err := core.LoadSettings(root)
	if err != nil {
		return 0, err
	}
	docs, err := requirement.Read(root, settings.RequirementsDir, settings.Prefix)
	if err != nil {
		return 0, err
	}
	form := requirement.NewIDForm(settings.Prefix)
	// A format error stops the trace before any GitHub call (CTR-TRC-009).
	if errs := check.Find(docs, form, settings.Areas); len(errs) > 0 {
		if err := check.WriteErrors(stderr, errs); err != nil {
			return 0, err
		}
		return len(errs), nil
	}
	if settings.RequirementLabel == "" {
		return 0, errors.New("invalid requirement_label: want a non-empty string")
	}
	repository, err := Repository(env.Repository, env.Remote)
	if err != nil {
		return 0, err
	}
	if env.Token == "" {
		return 0, errors.New("GITHUB_TOKEN is not set")
	}
	url := env.GraphQLURL
	if url == "" {
		url = platform.DefaultGraphQLURL
	}
	owner, name, _ := strings.Cut(repository, "/")

	readAt := env.Now().UTC().Format(time.RFC3339)
	issues, err := platform.NewGitHubClient(url, env.Token).IssuesWithLabel(owner, name, settings.RequirementLabel)
	if err != nil {
		return 0, err
	}
	output := Link(docs.Requirements, fromPlatform(issues), form, readAt, repository)

	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return 0, err
	}
	_, err = stdout.Write(append(encoded, '\n'))
	return 0, err
}
