# cumin-trace

A tool that keeps requirement documents as the source of truth and shows how far each requirement has come: requirement, requirement issue, implementation issue.

It reads Markdown requirement documents in a product repository, checks their format, and reads GitHub to show the progress of every requirement on one page. It works beside [cumin-works](https://github.com/cloveclovedev/cumin-works) and meets it only on GitHub.

The documents are in Japanese. Start at [docs/ja/index.md](docs/ja/index.md).

## Check the format

`cumin-trace check` reads the requirement documents of a repository and prints every format error.

1. Write `.cumin-trace.toml` at the top of the repository.

   ```toml
   # The prefix of the product
   prefix = "CTR"

   # The allowed areas
   areas = ["FMT", "CHK"]

   # The directory of the requirement documents. Optional.
   requirements_dir = "docs/ja/requirements"
   ```

2. Build the command in a clone of cumin-trace.

   ```sh
   go build -o cumin-trace ./cmd/cumin-trace
   ```

3. Run the built command at the top of the repository that holds the requirement documents.

   ```sh
   cumin-trace check
   ```

The check prints one line for each error: file, line, ID, and message.

```text
docs/ja/requirements/format/index.md:82: CTR-FMT-01: invalid ID format: want <prefix>-<area>-<3 digits>, or that with .<child number>
docs/ja/requirements/site/index.md:15: CTR-SITE-001: area SITE is not in .cumin-trace.toml
```

| Exit code | Meaning |
|---|---|
| 0 | No error |
| 1 | One or more format errors |
| 2 | The check cannot run: the settings or the directory is missing or wrong |

The check reads only the settings file and the requirement documents. It needs no token and no network.

The check finds the ten errors of `CTR-CHK-001`. The design is in [docs/ja/designs/check.md](docs/ja/designs/check.md).

## Check the documents of this repository

This repository holds its own settings file, [.cumin-trace.toml](.cumin-trace.toml): the prefix `CTR` and seven areas. Run the check at the top of a clone.

```sh
go run ./cmd/cumin-trace check
```

The check prints nothing and exits with 0. `go test ./...` runs the same check in `TestTheDocumentsOfThisRepositoryPassTheCheck`.
