# cumin-trace

A tool that keeps requirement documents as the source of truth and shows how far each requirement has come: requirement, requirement issue, implementation issue.

It reads Markdown requirement documents in a product repository, checks their format, and reads GitHub to show the progress of every requirement on one page. It works beside [cumin-works](https://github.com/cloveclovedev/cumin-works) and meets it only on GitHub.

The documents are in Japanese. Start at [docs/ja/index.md](docs/ja/index.md).

## Start the documents

`cumin-trace init` writes the first requirement documents into the current directory. Run it at the top of the repository of the product. Install the command as in step 2 of "Check the format".

```sh
cumin-trace init <prefix> [<area>]
```

| Argument | Required | Rule |
|---|---|---|
| `<prefix>` | Yes | Upper-case letters and digits. The first part of every ID, for example `PC` |
| `<area>` | No | Upper-case letters and digits. The first area, for example `TASK` |

`cumin-trace init PC TASK` writes these files:

| File | Content |
|---|---|
| `.cumin-trace.toml` | `prefix = "PC"` and `areas = ["TASK"]` |
| `docs/ja/requirements/index.md` | The entry page, with links to the other pages and a row for the area |
| `docs/ja/requirements/overview.md` | The overview: problem, goal, measures, people, out of scope, and phases |
| `docs/ja/requirements/technology.md` | The technology page: structure, technologies with reasons, library policy, and open choices |
| `docs/ja/requirements/glossary.md` | The glossary: one term for each row, with the words not to use and the English term |
| `docs/ja/requirements/backlog.md` | The backlog: each item with the reason to put it off and the trigger to review it |
| `docs/ja/requirements/task/index.md` | The area page: overview, requirements with the example `PC-TASK-001`, and open questions |
| `docs/ja/requirements/task/overview.puml` and `overview.svg` | The example diagram of the area page |
| `docs/ja/architecture/overview.md` | The source code structure: a place for a diagram, and a table of packages, roles, and designs |
| `.claude/skills/cumin-trace-docs/SKILL.md` | The skill `cumin-trace-docs` for Claude Code, which tells an agent how to write the requirement and design documents |
| `.claude/skills/cumin-trace-docs/templates/` | The nine templates of the skill: the entry page, the overview, the technology page, the glossary, the backlog, the area page, the source code structure, and the entry and the detail of a design document |
| `scripts/render-diagrams.sh` | The script that renders each PlantUML diagram (`.puml`) to SVG with Docker, as in cumin-works. Mode `0755` |
| `tools/plantuml/Dockerfile` | The PlantUML image of the script, with a Japanese font |
| `site/` | The Astro + Starlight site of the documents: `package.json` and `package-lock.json` with pinned versions, `.node-version`, `.gitignore`, `astro.config.mjs`, `src/settings.mjs`, `src/content.config.mjs`, `src/requirements.mjs`, `src/requirements.css`, and `scripts/check-site.mjs` |
| `.github/workflows/cumin-trace-check.yml` | The workflow `cumin-trace check`. It runs the format check on every pull request with the release binary of the same version as `init` |

Without an area, `init` writes `.cumin-trace.toml` with `areas = []`, and every page except the area page and its diagram. The skill still holds all nine templates. Its area template holds `<AREA>` in place of the area.

`init` prints one line for each file: `written <path>`, or `skipped <path>: already exists`. A binary without a release version (`v<number>.<number>.<number>`) writes no workflow. Such a binary has a pseudo-version, a `+dirty` build, or `(devel)`. It prints `skipped .github/workflows/cumin-trace-check.yml: this cumin-trace has no version; use a release binary`. It never changes a file that exists. The exit code is 0 when every file is written or skipped. The exit code is 2 when an argument is wrong or a file cannot be written.

`scripts/render-diagrams.sh` has three uses. It needs Docker.

| Command | Result |
|---|---|
| `scripts/render-diagrams.sh` | Renders every `.puml` under `docs/` to an SVG next to it |
| `scripts/render-diagrams.sh <in.puml> <out.svg>` | Renders one diagram to the given SVG |
| `scripts/render-diagrams.sh --check <base>` | Renders the diagrams that differ from `<base>`, and compares them with the SVGs. It changes no file |

The workflow downloads `cumin-trace-linux-amd64` and `checksums.txt` of that release with `gh`, checks the checksum, and runs `cumin-trace check`. It needs no Go. To use a newer version, edit the `VERSION` line, or remove the workflow and run `init` of the newer version.

The files pass `cumin-trace check` at once. The design is in [docs/ja/designs/templates/index.md](docs/ja/designs/templates/index.md).

### Preview the site

The site shows every Markdown document under `docs/ja/` as one page. A document with no title in its frontmatter takes its first heading as its title. The site reads the documents where they are and changes no file under `docs/ja/`. It needs Node.js 22.22.0 or later, and no token.

The first page of the sidebar is the dashboard (`<base>dashboard/`). It reads `site/trace.json`, the output of `cumin-trace trace`, at build time. It shows the time of the read, and the tree of the requirements with their progress, requirement issues, milestones, and implementation issues. Without a valid `site/trace.json`, the dashboard shows only that GitHub could not be read, and the other pages stay as they are. `dev-trace` writes `site/trace.json` only when the trace succeeds; pass the token as `GITHUB_TOKEN=$(gh auth token) npm --prefix site run dev-trace`. Git ignores `site/trace.json`.

```sh
npm --prefix site ci          # once; installs the pinned versions
npm --prefix site run dev     # the preview
npm --prefix site run dev-trace # runs cumin-trace trace into site/trace.json, then the preview
npm --prefix site run build   # static files in site/dist/
npm --prefix site run preview # serves site/dist/ with the base path
```

The optional table `[site]` of `.cumin-trace.toml` sets the site. `docs_dir` is the directory of the documents (default `docs/ja`); it holds `requirements_dir`. `base` is the base path of the site (default `/`); it starts and ends with `/`, with no empty, `.`, or `..` segment. `name` is the name of the site at the top left; it is not empty, not only white space, and holds no line break. With no `name`, the site takes the repository name from the URL of `origin`, then the prefix. `cumin-trace check` checks the three values.

## Check the format

`cumin-trace check` reads the requirement documents of a repository and prints every format error.

1. Write `.cumin-trace.toml` at the top of the repository, or let `cumin-trace init` write it.

   ```toml
   # The prefix of the product
   prefix = "BIL"

   # The allowed areas
   areas = ["PAY", "INV"]

   # The directory of the requirement documents. Optional. The default is docs/ja/requirements.
   requirements_dir = "docs/requirements"

   # The site. Optional. docs_dir must hold requirements_dir. The default is docs/ja.
   [site]
   docs_dir = "docs"
   ```

2. Install the command from a clone of cumin-trace, and add the directory of Go commands to `PATH`.

   ```sh
   go install ./cmd/cumin-trace
   export PATH="$PATH:$(go env GOPATH)/bin"
   ```

   When `go env GOBIN` prints a directory, `go install` writes the command there. Add that directory to `PATH` instead.

3. Run the command at the top of the repository that holds the requirement documents.

   ```sh
   cumin-trace check
   ```

The check prints one line for each error: file, line, ID, and message.

```text
docs/requirements/billing/index.md:12: BIL-PAY-01: invalid ID format: want <prefix>-<area>-<3 digits>, or that with .<child number>
docs/requirements/tax/index.md:5: BIL-TAX-001: area TAX is not in .cumin-trace.toml
```

| Exit code | Meaning |
|---|---|
| 0 | No error |
| 1 | One or more format errors |
| 2 | The check cannot run: the settings or the directory is missing or wrong |

The check reads only the settings file and the requirement documents. It needs no token and no network.

The check finds the ten errors of `CTR-CHK-001`. The design is in [docs/ja/designs/check/index.md](docs/ja/designs/check/index.md).

## Export the requirements

`cumin-trace export` writes every requirement of the documents as JSON to the standard output. Run it at the top of the repository.

```sh
cumin-trace export > requirements.json
```

Each requirement holds its ID, the parent ID, the title, the text, the verifications, the status (`active` or `deprecated`), the replacement ID, the file, and the line. The requirements follow the order of `areas` in `.cumin-trace.toml`, then the order of the documents. The same documents give the same bytes. The export needs no token and no network. The exit code is 0 when the JSON is written, 1 when the documents have a format error, and 2 when the export cannot run. On a format error, the export writes no JSON and writes the errors to the standard error. The design is in [docs/ja/designs/export/index.md](docs/ja/designs/export/index.md).

## Trace the requirements to the issues

`cumin-trace trace` reads the requirement documents and the requirement issues on GitHub, links them, and writes JSON to the standard output. Run it at the top of a clone of the product.

```sh
GITHUB_TOKEN=$(gh auth token) cumin-trace trace > trace.json
```

A requirement issue is an issue with the label `cumin/type/requirement`, open or closed. Set another label with `requirement_label` in `.cumin-trace.toml`. An issue links to each ID that an item of its `## Requirements` section starts with.

| Environment variable | Use |
|---|---|
| `GITHUB_TOKEN` | The token that reads GitHub. Required. It needs only read access to issues and pull requests |
| `GITHUB_REPOSITORY` | The repository, as `<owner>/<name>`. Without it, the trace uses the github.com URL of the remote `origin` |
| `GITHUB_GRAPHQL_URL` | The URL of the GraphQL API. The default is `https://api.github.com/graphql` |

The JSON holds the time of the read, the repository, and each requirement with its progress and its requirement issues: number, state, reason of the close, and milestone. Each requirement issue holds its sub-issues (the implementation issues), each with the pull request that closed it. Each requirement holds the count of its closed and all implementation issues. The trace only reads GitHub. It writes the token nowhere. The exit code is 0 when the JSON is written, 1 when the documents have a format error, and 2 when the trace cannot run.

The JSON also lists the discrepancies, each with its kind, the number of the requirement issue, and the ID: `unknown_id` (an ID outside the documents) and `open_issue_for_deprecated` (an open issue for a deprecated requirement). The design is in [docs/ja/designs/trace/index.md](docs/ja/designs/trace/index.md).

## Print the version

`cumin-trace version` prints the version of the binary in one line and exits with 0.

```sh
cumin-trace version
```

A release binary prints its tag, for example `v0.1.0`. A binary that `go build` or `go install` built from a clone prints the version that Go records, for example a pseudo-version. A binary without any version prints `(devel)`.

Wrong arguments print the usage line on the standard error and exit with 2:

```text
cumin-trace: usage: cumin-trace check | cumin-trace export | cumin-trace init <prefix> [<area>] | cumin-trace trace | cumin-trace version
```

## Check the documents of this repository

This repository holds its own settings file, [.cumin-trace.toml](.cumin-trace.toml): the prefix `CTR` and seven areas. Run the check at the top of a clone.

```sh
go run ./cmd/cumin-trace check
```

The check prints nothing and exits with 0. `go test ./...` runs the same check in `TestTheDocumentsOfThisRepositoryPassTheCheck`.

## Build the site of this repository

The site of this repository is in `site/`, written by `go run ./cmd/cumin-trace init CTR`. Its name `cumin-trace` and its base path `/cumin-trace/` are in `[site]` of `.cumin-trace.toml`, and nowhere else. It needs Node.js 22.22.0 or later.

```sh
npm --prefix site ci                 # once; installs the pinned versions
npm --prefix site run build          # static files in site/dist/
npm --prefix site run preview        # serves site/dist/ under /cumin-trace/
npm --prefix site run check-site     # builds, then checks the pages
```

`go test ./...` checks that each file of `site/` is the same as the template in `templates/site/`, and that `site/` holds no other file outside `node_modules/`, `dist/`, and `.astro/`.
