# リリースと workflow

[テンプレートの生成の設計](index.md) の細部。

## このリポジトリのリリース

`.github/workflows/release.yml` の全文は次のとおりである。Agent は `.github/workflows/` を書けないので、Maintainer がこのファイルを書き換える。

```yaml
name: release

# Runs on a version tag, for example v0.1.0. The tag is the version that the
# binaries print and that cumin-trace init writes into the check workflow.
# A pre-release tag (v0.1.0-rc.1) does not match: init treats it as no version.
on:
  push:
    tags:
      - 'v[0-9]+.[0-9]+.[0-9]+'

permissions:
  contents: write

jobs:
  release:
    name: release
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      # The publish step authenticates with GH_TOKEN. Keep the write token out
      # of .git/config while the tests and their dependencies run.
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version-file: go.mod
      - name: go vet
        run: go vet ./...
      - name: go test
        run: go test -race ./...
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

- checkout は `persist-credentials: false` にする。job は `contents: write` を持つので、既定の `true` では、書き込みのできるトークンが `.git/config` に残ったまま、テストと依存先のコードが動く。公開の step は `GH_TOKEN` で `gh` を認証するので、git の認証情報は要らない。
- テストは `ci.yml` と同じ `go vet ./...` と `go test -race ./...` にする。CI を通っていないコミットにタグを打っても、同じ検査を通ったものだけを公開する。
- `concurrency` は書かない。タグごとに別のリリースを作るので、並んで動いてもぶつからない。同じ group で待てる run は1つだけで、待っている run は次の run に取り消される。タグをまとめて push すると、間のリリースが黙って作られない。

## プロダクトの check の workflow

`init` が書く `.github/workflows/cumin-trace-check.yml` である。埋め込みは `templates/workflows/cumin-trace-check.yml` で、文は下と同じである。`{{version}}` は、`init` を実行したバイナリのバージョンになる。

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
      # The steps need no git credentials. Keep the token out of .git/config.
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - name: Download cumin-trace
        shell: bash
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

- checkout は `persist-credentials: false` にする。どの step も git の認証情報を使わない。`gh` は step の `GH_TOKEN` で認証するので、トークンを `.git/config` に残さない。
- `checksums.txt` には4つのバイナリが並ぶので、取ったバイナリの行だけを `sha256sum --check` に渡す。行がなければ `grep` が失敗し、step が失敗する。`shell: bash` で、GitHub は `bash --noprofile --norc -eo pipefail {0}` を実行するので、パイプの中の失敗も step の失敗になる。
- バージョンを上げるときは、新しいバージョンの `cumin-trace` で workflow を消してから `init` を実行するか、`VERSION` の行を書き換える。

## プロダクトのサイトの workflow

`init` が書く `.github/workflows/cumin-trace-site.yml` である (`CTR-DASH-008`)。埋め込みは `templates/workflows/cumin-trace-site.yml` で、文は下と同じである。`{{version}}` は、check の workflow と同じく、`init` を実行したバイナリのバージョンになる。ページの側の設計は [ダッシュボードのページ](../dashboard/page.md) の「workflow の例」にある。

```yaml
name: cumin-trace site

# Rebuilds the site with the progress of the requirements on every push to
# main and once a day. The workflow does not publish the site. Add a last
# step that puts site/dist/ where the site is published.
on:
  push:
    branches:
      - main
  schedule:
    - cron: '0 0 * * *'

permissions:
  contents: read
  issues: read
  pull-requests: read

jobs:
  cumin-trace-site:
    name: cumin-trace site
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      # The steps need no git credentials. Keep the token out of .git/config.
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - name: Download cumin-trace
        shell: bash
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
      # A failed trace fails the job, and the later steps do not run. The
      # site is not built, and the published site stays as it was.
      - name: cumin-trace trace
        env:
          GITHUB_TOKEN: ${{ github.token }}
        run: '"$RUNNER_TEMP/cumin-trace/cumin-trace-linux-amd64" trace > site/trace.json'
      - uses: actions/setup-node@949feb2413d6458794dcd2491c4babbbce0c15c1 # v7.1.0
        with:
          node-version-file: site/.node-version
      - name: npm ci
        run: npm --prefix site ci
      - name: Build the site
        run: npm --prefix site run build
```

- きっかけは、`main` への `push` と、毎日 0:00 (UTC) の `schedule` である。文書の変更と、GitHub の Issue の変化の両方を、1日以内にサイトに出す。
- 権限は `contents: read`、`issues: read`、`pull-requests: read` だけである。`trace` に渡す token は、Issue と pull request を読むだけで足りる (`CTR-TRC-007`)。
- cumin-trace のダウンロードは、check の workflow と同じ step である。
- `trace` の step には `shell` を書かない。GitHub は `bash -e {0}` を実行するので、`trace` が 0 以外の終了コードで終わると、step が失敗する。続く step は既定の `if: success()` で動かないので、サイトはビルドされない。公開している場所のサイトは前のまま残る。
- `GITHUB_REPOSITORY` は、GitHub Actions が設定する。`trace` は、そこから読むリポジトリを決める ([対応づけのコマンド](../trace/command.md))。
- 公開の step も、成果物のアップロードもない。`site/dist/` をどこに置くかは、プロダクトが最後に step を足して決める。
- `actions/checkout` と `actions/setup-node` のコミットは、このリポジトリの `.github/workflows/site.yml` と同じである。

## 確かめたこと

公式ページ (2026-10-14 に読んだ):

- <https://cli.github.com/manual/gh_release_download>: `gh release download [<tag>] [flags]`。`--pattern` は「Download only assets that match a glob pattern」、`--dir` の既定は `.`、`--repo` で別のリポジトリを選ぶ。
- <https://cli.github.com/manual/gh_release_create>: `gh release create [<tag>] [<filename>... | <pattern>...]`。`--verify-tag` は、リモートにタグがなければ止める。`--generate-notes` は、タイトルとノートを自動で作る。
- <https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/using-github-cli-in-workflows>: GitHub CLI は GitHub のランナーに最初から入っている。step ごとに `GH_TOKEN` を設定する。
- <https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax>: `on.push.tags` は `refs/tags` に対するパターンである。フィルタのパターンで、`+` は直前の文字の1回以上、`[]` は括弧の中の1文字に合う (例 `v[12].[0-9]+.[0-9]+`)。`shell` を書かないと `bash -e {0}`、`shell: bash` では `bash --noprofile --norc -eo pipefail {0}` を実行する。「By default, fail-fast behavior is enforced using `set -e` for both `sh` and `bash`.」`continue-on-error` は「Prevents a job from failing when a step fails」で、サイトの workflow には書かない。`contents: write` で、リリースを作れる。権限を1つでも書くと、書かない権限は `none` になる。
- <https://docs.github.com/en/actions/reference/workflows-and-actions/expressions> (2026-10-20 に読んだ): `success()` は「Returns `true` when all previous steps have succeeded」。「A default status check of `success()` is applied unless you include one of these functions.」このため、失敗した step のあとの step は動かない。
- <https://docs.github.com/en/actions/reference/workflows-and-actions/contexts>: `github.ref_name` は、実行のきっかけのブランチかタグの短い名前である。
- `actions/checkout` と `actions/setup-go` のコミットは、このリポジトリの `.github/workflows/ci.yml` と同じである。

手元の実行 (Go 1.27.2、macOS): workflow の「Build the binaries」と同じ `for` の文で、4つのバイナリをビルドできた。`sha256sum` で `checksums.txt` を作り、`grep ' cumin-trace-linux-amd64$' checksums.txt | sha256sum --check -` が `cumin-trace-linux-amd64: OK` を出した。

未確認 (自分の推論):

- `GITHUB_TOKEN` で、別の公開リポジトリのリリースを `gh release download` できること。公開リポジトリのリリースは誰でも読めるので、できると考える。
- ubuntu のランナーでの `sha256sum` の動き。手元では通った (上)。
