# パッケージとファイル

[書式検査の設計](index.md) の細部。

## ファイル

| ファイル | 役割 |
|---|---|
| `cmd/cumin-trace/main.go` | 引数からコマンドを選び、結果を終了コードにする |
| `internal/check/service.go` | 設定と要件文書を読み、ルールを適用し、エラーを出力する |
| `internal/check/domain.go` | `CTR-CHK-001` の10個のルールと、エラーの並べ方。純粋な関数で、I/O を持たない |
| `internal/requirement/domain.go` | 要件のモデルと、IDの形式。純粋な関数で、I/O を持たない |
| `internal/requirement/markdown.go` | 要件文書のファイルを集め、Markdown を読んで、要件のモデルにする |
| `internal/core/settings.go` | `.cumin-trace.toml` を読み、値を確かめる |

## 依存

- ライブラリの型は、それを使う1ファイルの外に出さない。`toml` の型は `internal/core/settings.go` で、`goldmark` の型は `internal/requirement/markdown.go` で止める。
- `internal/check/` は、`internal/requirement/` と `internal/core/` を使う。
- `internal/requirement/` は `internal/core/` を import しない。読み取りは、リポジトリの最上位、要件文書のディレクトリ、接頭辞を引数で受け取り、`service.go` が設定から渡す。
- `internal/check/` は `net/http` を import せず、環境変数も読まない (`CTR-CHK-004`)。

## 入口

`check.Run(root string, stdout io.Writer) (int, error)` が入口である。`root` はリポジトリの最上位のディレクトリ、戻り値はエラーの件数である。`main.go` は `root` に `.` を渡す。
