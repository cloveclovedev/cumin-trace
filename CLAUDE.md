# cumin-trace

cumin-trace keeps the requirement documents of a product as the source of truth and shows how far each requirement has come. It reads Markdown requirement documents, checks their format, reads the requirement issues on GitHub, and shows the progress of every requirement on one page of a Starlight site.

The product name and the command name are both `cumin-trace`. It works beside cumin-works and meets it only on GitHub: it reads the label of requirement issues and their sub-issues.

The first goal (v0.1) is that the requirements of a new product can be written in this format, and that one page shows the state of each requirement, its requirement issues, and its implementation issues.

## Read first

All documents are in Japanese under `docs/ja/`. Start at `docs/ja/index.md` and follow the links to every document that your task touches. Read them before you change code.

- `docs/ja/requirements/index.md` lists the areas. Each area has one directory with its page (`index.md`) and its diagrams.
- `docs/ja/requirements/format/index.md` is the source of truth for how a requirement is written. The documents of this repository follow it too.
- `docs/ja/requirements/backlog.md` lists what is decided to be out of scope for now. Do not build those things.

## Rules that you must not break

- Do not change anything under `docs/ja/requirements/` without a Maintainer's approval in the conversation. If the implementation needs a requirement change, stop and ask first.
- Do not commit to `main`. Create a branch from `main` for every change.
- Do not read or print secret values: tokens, private keys, `.env` files. Tests generate what they need.
- Do not write employer information, personal circumstances, or local absolute paths in documents, code, tests, fixtures, commit messages, or pull requests. This repository is public.

## How we work

- Simple first. Look for the simplest way that meets the requirement.
- Requirement first. Every behavior traces to a requirement by its ID (for example `CTR-CHK-002`). If no requirement covers the behavior, ask.
- Document first. Documents stay current with the code, so that another person can read what works today.
- Test first. Each requirement names how it is verified (`test`, `demo`, `inspection`, `analysis`). A requirement with `test` gets a test that proves it. Prefer a few tests that prove a requirement over many unit tests.
- Verify external APIs and tools in the official documentation before you use them: the GitHub REST and GraphQL API, Astro, Starlight, Go libraries. Do not rely on memory. Say which page you checked. Separate verified facts from your own reasoning.

## Design constraints

These come from the principles in `docs/ja/requirements/overview.md`.

- cumin-trace follows fixed rules only. It contains no AI judgment. The same input always gives the same output. Reading the meaning of a requirement is the work of an agent or of a Maintainer.
- The requirement documents are the source of truth. Structured data (`requirements.json`, the trace result) is generated and never edited by hand. The tool never changes a source document.
- cumin-trace only reads GitHub. It creates no issue, changes no label, and writes no comment.
- The progress of a requirement and the version that it belongs to are never written in a requirement document. They follow from the issues and the milestones on GitHub.
- The check and the export need no network and no token.
- The token for GitHub comes from an environment variable. Never store it and never log it.
- Configuration of a product repository is TOML (`.cumin-trace.toml`).
- The site and the dashboard are static. Nothing needs a server.

## Code

- Go, standard library first. Do not rebuild a large or security-sensitive component that a well-maintained library already provides: propose the library with your reasons and ask before you add it.
- Module path: `github.com/cloveclovedev/cumin-trace`.
- The architecture is a lightweight clean architecture, organized by feature, as in cumin-works. Dependencies point inward: I/O adapters depend on the rules, and the rules depend on nothing external.
  - One feature owns it: `internal/<feature>/`.
  - Shared and provider-neutral: `internal/core/`.
  - Shared and provider-specific (GitHub): `internal/platform/`. The provider's types stop at this boundary.
  - Wiring: `cmd/cumin-trace/`.
  - Text that is written into a product repository (templates, the skill, workflows, the site template): the top level, embedded by its own package.
- Inside a feature package, separate roles by file: pure rules and models (`domain.go`), orchestration (`service.go`), outbound I/O (`store.go` or a file named for the external system). The pure rules import no HTTP client, no `os/exec`, and no provider type.
- Do not add an interface unless a second implementation or a real testing need exists. Do not create empty packages.
- Logs are structured (`log/slog`). Messages for the person who runs a command (errors of the check, with file and line) go to the output of the command in plain text.
- Code, comments, test names, commit messages, and pull requests are in English.

## Tests

- Tests run with `go test ./...` and need no network and no token. Tests of the trace use a fake GitHub (`httptest`) behind the real client.
- Name each test by what it proves, and name the requirement that it proves in the test name or in a comment, by its ID.
- Make every test give the same result on any machine, at any time, and in any order.

## Commands

```sh
go build ./...
go vet ./...
go test -race ./...
gofmt -l .                   # must print nothing
```

## Documents

- Documents are in Japanese under `docs/ja/`, in natural technical Japanese. Before you open a pull request, find every document that the change makes wrong, and update it in the same pull request.
- A requirement is a heading with its ID in a code span and its title, then its text, then its `検証` lines, then a horizontal rule. An ID is never used again.
- A design document under `docs/ja/designs/` describes the current design of one feature only. Rely on git history for how it changed.
- Diagrams are PlantUML with English labels. Keep the `.puml` next to the document, render the SVG, and commit both. Until this repository has its own render script (`CTR-TPL-006`), render with the PlantUML image of cumin-works.

## Commits and pull requests

- Conventional Commits, and one purpose for each pull request.
- A pull request says what changed, how it was checked (what, how, result), and whether documents changed.
