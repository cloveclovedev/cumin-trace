# パッケージとファイル

[要件と要求Issueの対応づけの設計](index.md) の細部。

## ファイル

| ファイル | 役割 |
|---|---|
| `cmd/cumin-trace/main.go` | `trace` を選び、環境変数と `origin` の URL を `trace.Env` に入れ、結果を終了コードにする |
| `internal/trace/service.go` | 設定と要件文書を読み、書式検査を適用し、GitHub を読み、JSON を書く |
| `internal/trace/domain.go` | Issueのモデル (要求Issue、実装Issue、閉じた Pull Request)。`origin` の URL からリポジトリを決める規則、進捗、不整合、実装Issueの数、JSON の構造と並べ方、commit ごとに要件を比べて最後に変わった日時を決める規則 (`LastChanges`、`UnchangedIDs`)。要件のオブジェクトは `export.Requirement` を埋め込む。純粋な関数で、I/O を持たない |
| `internal/trace/history.go` | `git` を実行して、要件文書のディレクトリに触れた commit と、その commit と親の要件文書を読み、IDごとに最後に変わった日時を返す (`lastChangesOf`、`CTR-TRC-010`)。shallow なリポジトリと、git のリポジトリでないときはエラーにする |
| `internal/trace/convert.go` | `internal/platform` の型を、`domain.go` のモデルに変える ([GitHub の値の変換](github.md#github-の値の変換)) |
| `internal/requirement/markdown.go` | 要求Issueの本文の `## Requirements` の節から、項目の先頭のIDを読む関数 (`ListedIDs`) を足す。commit のファイルの中身から要件を読む関数 (`Parse`) を足す |
| `internal/platform/github.go` | GraphQL の3つの問い合わせとページの繰り返し。問い合わせが読む値をすべて持つ、このファイルで定義した Go の型 (`Issue`、`SubIssue`、`PullRequest`) を返す |
| `internal/core/settings.go` | `requirement_label` のキーを足す |

## 入口

```go
// internal/trace
type Env struct {
	Token      string           // GITHUB_TOKEN
	Repository string           // GITHUB_REPOSITORY
	Remote     string           // the URL of `git remote get-url origin`, or empty
	GraphQLURL string           // GITHUB_GRAPHQL_URL. Empty means https://api.github.com/graphql
	Now        func() time.Time // the clock for read_at
}

func Run(root string, env Env, stdout, stderr io.Writer) (int, error)
```

- 戻り値の数は書式のエラーの件数で、`error` は対応づけができないことを表す。`main.go` は、`check` と同じく、0 件なら 0、1 件以上なら 1、`error` なら 2 で終わる。
- `internal/platform` (パッケージ名 `platform`) の入口は、`platform.NewGitHubClient(url, token string) *GitHubClient` と `(*GitHubClient).IssuesWithLabel(owner, name, label string) ([]platform.Issue, error)` である。sub-issue の続きと閉じた Pull Request も、この1つの呼び出しの中で読む。

## 依存

- `internal/trace/` は、`internal/check/`、`internal/requirement/`、`internal/core/`、`internal/platform/`、`internal/export/` を使う。要件のオブジェクトと並べ方は、`domain.go` が `export.Of` と `export.Sort` で作る (`CTR-TRC-008`)。書式検査は `check.Find` をそのまま使い、エラーの行も `check` と同じ `check.WriteErrors` で書く。こうして同じエラーを出す (`CTR-TRC-009`)。
- `internal/trace/domain.go` は、`net/http`、`os`、`internal/platform/` を import しない。`internal/platform/` を import するのは、`service.go` と `convert.go` だけである。
- `internal/platform/` は、ほかの `internal/` のパッケージを import しない。
- GraphQL の応答の型は `internal/platform/github.go` で止め、外にはこのファイルで定義した Go の型だけを返す。値は GitHub のまま (`OPEN`、`NOT_PLANNED` など) で、意味づけは `convert.go` がする。
- 環境変数を読むのは `main.go` だけである。`git` を実行するのは、`origin` の URL を読む `main.go` と、履歴を読む `internal/trace/history.go` だけである。`domain.go` は `git` を実行しない。
- 新しいライブラリは使わない。HTTP と JSON は標準ライブラリで足りる。git の履歴は `os/exec` で `git` を実行して読む。interface は作らない。テストは、偽の GitHub の URL を `GraphQLURL` に渡して、本物の client を使う。
