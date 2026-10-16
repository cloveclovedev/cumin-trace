# パッケージとファイル

[対応づけの設計](index.md) の細部。

## ファイル

| ファイル | 役割 |
|---|---|
| `cmd/cumin-trace/main.go` | `trace` を選び、環境変数を `trace.Env` に入れ、結果を終了コードにする |
| `internal/trace/service.go` | 設定と要件文書を読み、書式検査を適用し、GitHub を読み、JSON を書く |
| `internal/trace/domain.go` | IDの見つけ方、進捗、不整合、実装Issueの数、JSON の構造と並べ方。純粋な関数で、I/O を持たない |
| `internal/platform/github.go` | GraphQL の3つの問い合わせ、ページの繰り返し、GitHub の値の変換 |
| `internal/core/issue.go` | プロバイダを知らないIssueのモデル (要求Issue、実装Issue、閉じた Pull Request) |
| `internal/core/settings.go` | `requirement_label` と `repository` のキーを足す |

## 入口

```go
// internal/trace
type Env struct {
	Token      string           // GITHUB_TOKEN
	Repository string           // GITHUB_REPOSITORY
	GraphQLURL string           // GITHUB_GRAPHQL_URL. Empty means https://api.github.com/graphql
	Now        func() time.Time // the clock for read_at
}

func Run(root string, env Env, stdout, stderr io.Writer) (int, error)
```

- 戻り値の数は書式のエラーの件数で、`error` は対応づけができないことを表す。`main.go` は、`check` と同じく、0 件なら 0、1 件以上なら 1、`error` なら 2 で終わる。
- `internal/platform` (パッケージ名 `platform`) の入口は、`platform.NewGitHubClient(url, token string) *GitHubClient` と `(*GitHubClient).RequirementIssues(owner, name, label string) ([]core.RequirementIssue, error)` である。sub-issue の続きと閉じた Pull Request も、この1つの呼び出しの中で読む。

## 依存

- `internal/trace/` は、`internal/check/`、`internal/requirement/`、`internal/core/`、`internal/platform/` を使う。書式検査は `check.Find` をそのまま使い、同じエラーを出す (`CTR-TRC-009`)。
- `internal/trace/domain.go` は、`net/http`、`os`、`internal/platform/` を import しない。
- `internal/platform/` は、`internal/core/` だけを使う。`internal/trace/` を import しない。
- GitHub の JSON の型は `internal/platform/github.go` で止め、外には `internal/core` のモデルだけを返す。
- 環境変数を読むのは `main.go` だけである。
- 新しいライブラリは使わない。HTTP と JSON は標準ライブラリで足りる。interface は作らない。テストは、偽の GitHub の URL を `GraphQLURL` に渡して、本物の client を使う。
