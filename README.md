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

2. Run the check at the top of the repository.

   ```sh
   go run github.com/cloveclovedev/cumin-trace/cmd/cumin-trace@latest check
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

Today the check finds two errors: an ID with a wrong form, and an ID whose area is not in the settings. The design is in [docs/ja/designs/check.md](docs/ja/designs/check.md).
