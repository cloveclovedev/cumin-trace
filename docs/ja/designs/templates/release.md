# リリースと workflow

[テンプレートの生成の設計](index.md) の細部。

## このリポジトリのリリース

Maintainer が、次の全文を `.github/workflows/release.yml` として足す。Agent は `.github/workflows/` を書けないためである。

```yaml
name: release

# Runs on a version tag, for example v0.1.0. The tag is the version that the
# binaries print and that cumin-trace init writes into the check workflow.
on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    name: release
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version-file: go.mod
      - name: go test
        run: go test ./...
      - name: Build the binaries
        env:
          VERSION: ${{ github.ref_name }}
        run: |
          mkdir dist
          for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
            os="${target%/*}"
            arch="${target#*/}"
            CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath \
              -ldflags "-X main.version=$VERSION" \
              -o "dist/cumin-trace-$os-$arch" ./cmd/cumin-trace
          done
          cd dist
          sha256sum cumin-trace-* > checksums.txt
      - name: Publish the release
        env:
          GH_TOKEN: ${{ github.token }}
          VERSION: ${{ github.ref_name }}
        run: gh release create "$VERSION" --verify-tag --title "$VERSION" --generate-notes dist/*
```

リリースのファイル:

| ファイル | 中身 |
|---|---|
| `cumin-trace-linux-amd64` | GitHub のランナー用。check の workflow が使う |
| `cumin-trace-linux-arm64`、`cumin-trace-darwin-amd64`、`cumin-trace-darwin-arm64` | 手元で `init` と `check` を実行する人のため |
| `checksums.txt` | 4つのバイナリの SHA-256。`sha256sum` の出力の形 (`<hash>  <file>`) |

## プロダクトの check の workflow

`init` が書く `.github/workflows/cumin-trace-check.yml` である (#38)。`{{version}}` は、`init` を実行したバイナリの版になる。

```yaml
name: cumin-trace check

# No path filter on pull_request. A workflow that is skipped leaves a required
# check pending, and the pending check blocks the merge.
on:
  pull_request:

permissions:
  contents: read

jobs:
  cumin-trace-check:
    name: cumin-trace check
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
      - name: Download cumin-trace
        env:
          GH_TOKEN: ${{ github.token }}
          VERSION: {{version}}
        run: |
          dir="$RUNNER_TEMP/cumin-trace"
          gh release download "$VERSION" --repo cloveclovedev/cumin-trace --dir "$dir" \
            --pattern cumin-trace-linux-amd64 --pattern checksums.txt
          cd "$dir"
          grep ' cumin-trace-linux-amd64$' checksums.txt | sha256sum --check -
          chmod +x cumin-trace-linux-amd64
      - name: cumin-trace check
        run: '"$RUNNER_TEMP/cumin-trace/cumin-trace-linux-amd64" check'
```

- `checksums.txt` には4つのバイナリが並ぶので、取ったバイナリの行だけを `sha256sum --check` に渡す。行がなければ `grep` が失敗し、step が失敗する。
- 版を上げるときは、新しい版の `cumin-trace` で workflow を消してから `init` を実行するか、`VERSION` の行を書き換える。

## 確かめたこと

公式ページ (2026-10-14 に読んだ):

- <https://cli.github.com/manual/gh_release_download>: `gh release download [<tag>] [flags]`。`--pattern` は「Download only assets that match a glob pattern」、`--dir` の既定は `.`、`--repo` で別のリポジトリを選ぶ。
- <https://cli.github.com/manual/gh_release_create>: `gh release create [<tag>] [<filename>... | <pattern>...]`。`--verify-tag` は、リモートにタグがなければ止める。`--generate-notes` は、タイトルとノートを自動で作る。
- <https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/using-github-cli-in-workflows>: GitHub CLI は GitHub のランナーに最初から入っている。step ごとに `GH_TOKEN` を設定する。
- <https://docs.github.com/en/actions/writing-workflows/workflow-syntax-for-github-actions>: `on.push.tags` は `refs/tags` に対するパターンである。`contents: write` で、リリースを作れる。権限を1つでも書くと、書かない権限は `none` になる。
- <https://docs.github.com/en/actions/reference/workflows-and-actions/contexts>: `github.ref_name` は、実行のきっかけのブランチかタグの短い名前である。
- `actions/checkout` と `actions/setup-go` のコミットは、このリポジトリの `.github/workflows/ci.yml` と同じである。

手元の実行 (Go 1.27.2、macOS): workflow の「Build the binaries」と同じ `for` の文で、4つのバイナリをビルドできた。`sha256sum` で `checksums.txt` を作り、`grep ' cumin-trace-linux-amd64$' checksums.txt | sha256sum --check -` が `cumin-trace-linux-amd64: OK` を出した。

未確認 (自分の推論):

- `GITHUB_TOKEN` で、別の公開リポジトリのリリースを `gh release download` できること。公開リポジトリのリリースは誰でも読めるので、できると考える。#38 の demo で確かめる。
- ubuntu のランナーでの `sha256sum` の動き。手元では通った (上)。最初のリリースのあとに、ランナーで確かめる。
