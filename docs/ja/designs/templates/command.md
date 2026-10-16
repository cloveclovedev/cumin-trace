# コマンド

[テンプレートの生成の設計](index.md) の細部。

## 引数

```sh
cumin-trace init <prefix> [<area>]
```

| 引数 | 必須 | 決まり |
|---|---|---|
| `<prefix>` | 必須 | 英大文字と数字が1文字以上。`.cumin-trace.toml` の `prefix` と同じ決まり ([設定ファイル](../check/settings.md)) |
| `<area>` | 省略できる | 英大文字と数字が1文字以上。`areas` の1つと同じ決まり。省略すると、カテゴリのページと図の例を書かない ([ファイルの一覧](files.md))。空の文字列を渡すと、省略ではなく `invalid area ""` のエラーにする |

書く先は、現在のディレクトリである。`main.go` は、`check` と同じく `root` に `.` を渡す。

## 出力

標準出力に、[ファイルの一覧](files.md) の順で、ファイルごとに1行を書く。パスは、リポジトリの最上位からの相対パスで、区切りは `/` である。

| 場合 | 行 |
|---|---|
| ファイルを書いた | `written <path>` |
| ファイルがすでにある | `skipped <path>: already exists` |
| バージョンのないバイナリで、workflow がまだない | `skipped .github/workflows/cumin-trace-check.yml: this cumin-trace has no version; use a release binary` |

ファイルがあるかを先に確かめ、なければバージョンを確かめる。バージョンのないバイナリでも、workflow がすでにあれば `already exists` の行になる。

例 (`cumin-trace init PC TASK`、空のリポジトリ):

```text
written .cumin-trace.toml
written docs/ja/requirements/index.md
...
written docs/ja/requirements/backlog.md
written docs/ja/requirements/task/index.md
written docs/ja/requirements/task/overview.puml
written docs/ja/requirements/task/overview.svg
written docs/ja/architecture/overview.md
...
written .github/workflows/cumin-trace-check.yml
```

例 (`cumin-trace init PC`、空のリポジトリ)。カテゴリのページと図の例の行がない:

```text
written .cumin-trace.toml
written docs/ja/requirements/index.md
...
written docs/ja/requirements/backlog.md
written docs/ja/architecture/overview.md
...
written .github/workflows/cumin-trace-check.yml
```

## 終了コード

| コード | 場合 | 標準エラー出力 |
|---|---|---|
| 0 | すべてのファイルを、書いたか飛ばした | なし |
| 2 | 引数の数が1つでも2つでもない | `cumin-trace: usage: cumin-trace init <prefix> [<area>]` |
| 2 | 接頭辞、またはカテゴリが決まりに合わない | `cumin-trace: invalid prefix "<値>": want upper-case letters and digits` (カテゴリは `invalid area`) |
| 2 | ファイルかディレクトリを作れない | `cumin-trace: <OS のエラー>`。そのファイルで止まり、それまでに書いたファイルは残す。作ったあとで書き込みか close に失敗したファイルは消すので、次の実行で書き直せる |

`check` の使い方の行は、`cumin-trace: usage: cumin-trace check | cumin-trace init <prefix> [<area>]` に変える。コマンドがないとき、知らないコマンドのときに出す。

## パッケージとファイル

| ファイル | 役割 |
|---|---|
| `cmd/cumin-trace/main.go` | 引数からコマンドを選ぶ。`var version string` を持ち、空なら `runtime/debug.ReadBuildInfo` の `Main.Version` を使う。結果を終了コードにする |
| `internal/scaffold/service.go` | 入口 `scaffold.Run(root, prefix, area, version string, stdout io.Writer) error`。`area` が空なら、カテゴリがない。値を確かめ、ファイルごとに埋め込みの文を読み、置き換え、書き、1行を出す |
| `internal/scaffold/domain.go` | ファイルの一覧、置き換える値、バージョンの決まり、出力の行。純粋な関数で、I/O を持たない。接頭辞とカテゴリの決まり (`core.IsName`) と設定ファイルのパスは、`internal/core` のものを使う |
| `internal/scaffold/store.go` | `os.OpenFile` の `O_CREATE\|O_EXCL` でファイルを作る。作ったファイルは `File.Chmod` で一覧の権限にし、umask によらない。ないディレクトリは `os.MkdirAll` で作る |
| `templates/templates.go` | パッケージ `templates`。`//go:embed` で埋め込んだ `embed.FS` を `Files` として公開する |

- バージョンとして使うのは、`v<数>.<数>.<数>` の形の値だけである。空、`(devel)`、`-` や `+` を含む値 (疑似バージョン、変更のあるビルド) は、バージョンがないものとして扱う。
- `internal/scaffold/` は `internal/check/` を import しない。生成した直後に検査が通ることは、テストで `check.Run` を呼んで確かめる。

## 確かめたこと

- `os.OpenFile` の `O_EXCL` は、`O_CREATE` と一緒に使い、ファイルがあれば失敗する: <https://pkg.go.dev/os#pkg-constants> (Go 1.27.2 の `go doc os.O_EXCL` で読んだ)。
- `os.OpenFile` の権限は umask を引く前の値で、`File.Chmod` は権限をその値にする: `go doc os.OpenFile`、`go doc os.File.Chmod` (Go 1.27.2)。
- `-X importpath.name=value` は、初期値のない、または定数で初期化した文字列の変数に値を入れる: `go doc cmd/link` (Go 1.27.2)。
- Go 1.24 から、`go build` は VCS のタグとコミットから、本体のモジュールのバージョンをバイナリに記録する。変更があれば `+dirty` を足す: <https://go.dev/doc/go1.24>。
- 未確認: `go install ...@v0.1.0` で入れたバイナリの `Main.Version` が `v0.1.0` になること。
