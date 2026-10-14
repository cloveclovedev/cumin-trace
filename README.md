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
| `.claude/skills/cumin-trace-docs/templates/` | The nine templates of the skill: the seven documents above, and the entry and the detail of a design document |
| `scripts/render-diagrams.sh` | The script that renders each PlantUML diagram (`.puml`) to SVG with Docker, as in cumin-works. Mode `0755` |
| `tools/plantuml/Dockerfile` | The PlantUML image of the script, with a Japanese font |

Without an area, `init` writes `.cumin-trace.toml` with `areas = []`, and every page except the area page and its diagram. The skill still holds all nine templates. Its area template holds `<AREA>` in place of the area.

`init` prints one line for each file: `written <path>`, or `skipped <path>: already exists`. It never changes a file that exists. The exit code is 0 when every file is written or skipped. The exit code is 2 when an argument is wrong or a file cannot be written.

`scripts/render-diagrams.sh` has three uses. It needs Docker.

| Command | Result |
|---|---|
| `scripts/render-diagrams.sh` | Renders every `.puml` under `docs/` to an SVG next to it |
| `scripts/render-diagrams.sh <in.puml> <out.svg>` | Renders one diagram to the given SVG |
| `scripts/render-diagrams.sh --check <base>` | Renders the diagrams that differ from `<base>`, and compares them with the SVGs. It changes no file |

The files pass `cumin-trace check` at once. The design is in [docs/ja/designs/templates/index.md](docs/ja/designs/templates/index.md).

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

## Check the documents of this repository

This repository holds its own settings file, [.cumin-trace.toml](.cumin-trace.toml): the prefix `CTR` and seven areas. Run the check at the top of a clone.

```sh
go run ./cmd/cumin-trace check
```

The check prints nothing and exits with 0. `go test ./...` runs the same check in `TestTheDocumentsOfThisRepositoryPassTheCheck`.
