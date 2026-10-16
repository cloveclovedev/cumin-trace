# 対応づけの設計

`cumin-trace trace` が、要件文書と GitHub の要求Issueをどう読み、進捗と不整合をどう決めて JSON に出すかを決める。

## 要件

- [GitHub との対応づけ](../../requirements/trace/index.md)
  - `CTR-TRC-001` 要求Issueの見分け方
  - `CTR-TRC-002` 要件と要求Issueの結び付き
  - `CTR-TRC-003` 実装Issueと Pull Request
  - `CTR-TRC-004` 進捗 (`CTR-TRC-004.1` 未起票、`CTR-TRC-004.2` 進行中、`CTR-TRC-004.3` 完了)
  - `CTR-TRC-005` 子を持つ要件の進捗
  - `CTR-TRC-006` 不整合の報告
  - `CTR-TRC-007` GitHub には書き込まない
  - `CTR-TRC-008` 結果の出力
  - `CTR-TRC-009` 書式の誤りがあるとき
- [コマンド](../../requirements/command/index.md)
  - `CTR-CMD-001` コマンドの一覧
  - `CTR-CMD-003` 引数の誤り
- [書式検査](../../requirements/check/index.md)
  - `CTR-CHK-001` 検出するエラー (対応づけの前に適用する)
- [構造化データの出力](../../requirements/export/index.md)
  - `CTR-EXP-002` 同じ文書からは同じ出力 (要件の並べ方を借りる)

## 全体像

[ソースコード全体構成](../../architecture/overview.md) の `internal/trace`、`internal/platform`、`internal/core` にあたる。

![対応づけのパッケージ](trace-packages.svg)

図の元ファイル: [trace-packages.puml](trace-packages.puml)

書式検査を先に済ませ、エラーがなければ GitHub を読む (`CTR-TRC-009`)。

![対応づけのシーケンス](trace-sequence.svg)

図の元ファイル: [trace-sequence.puml](trace-sequence.puml)

## 決めたこと

### 対応づけは、独立したコマンド `cumin-trace trace`

引数を取らない。結果の JSON は標準出力に書く。

- 理由: `check` と `export` は token もネットワークも要らない (`CTR-CHK-004`)。GitHub を読むコマンドを分けると、その2つは手元でも Pull Request の check でも、token なしで動き続ける。
- 採らなかった案: `export` に GitHub の読み取りを足す。`CTR-EXP-002` (同じ文書からは同じ出力) と、読んだ時刻を持つ出力 (`CTR-TRC-008`) が両立しない。
- 採らなかった案: 出力先のファイルを引数で受け取る。標準出力をリダイレクトすれば足りる。
- `CTR-CMD-001` の表に `trace` の行を足すのは、Maintainer の作業である。
- 細部: [コマンドと出力](command.md)

### 設定は2つのキーを足す。token は環境変数だけ

`.cumin-trace.toml` に `requirement_label` (既定 `cumin/type/requirement`) と `repository` (省略できる) を足す。`repository` がなければ、環境変数 `GITHUB_REPOSITORY` を使う。token は環境変数 `GITHUB_TOKEN` から読む。

- 理由: GitHub Actions は `GITHUB_REPOSITORY` を必ず渡すので、設定に書かなくても動く。手元では、設定に1度書けば毎回渡さなくてよい。token はファイルに書かない (`CTR-TRC-007`)。
- 細部: [コマンドと出力](command.md)

### GitHub は GraphQL API だけで読む

読むのは、要求Issueの一覧と sub-issue の一覧 (1種類の問い合わせ)、閉じた実装Issueを閉じた Pull Request (もう1種類の問い合わせ) である。問い合わせは `query` だけで、`mutation` を持たない。

- 理由: ラベル、閉じた理由、マイルストーン、sub-issue を、100件の要求Issueにつき1回の呼び出しで読める。REST API では、要求Issueごとに sub-issue の呼び出しが、実装Issueごとに timeline の呼び出しが要る。
- 細部: [GitHub の読み方](github.md)

### IDは、本文の全体に `IDForm.References` を使って探す

要求Issueの本文を Markdown として読まず、文字列の全体に `IDForm.References` (`internal/requirement`) を適用する。

- 理由: 要件文書の参照 (`CTR-FMT-010`) と同じ決まりで、同じIDを見つける。要求Issueの本文は cumin-works が書く短い文で、コードブロックの中の例を気にしなくてよい。
- 細部: [進捗と不整合](rules.md)

### パッケージは `internal/trace` と `internal/platform`

GitHub の型と GraphQL の文は `internal/platform/github.go` で止める。`internal/core/issue.go` が、プロバイダを知らないIssueのモデルを持つ。

- 理由: `CLAUDE.md` の構成のとおり、規則 (`internal/trace/domain.go`) が GitHub を知らずにテストできる。
- 細部: [パッケージとファイル](packages.md)
