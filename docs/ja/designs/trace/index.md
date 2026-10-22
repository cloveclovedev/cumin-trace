# 要件と要求Issueの対応づけの設計

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
  - `CTR-TRC-010` 変更後の未起票 (いまは、変わった日時を読むところまで)
- [コマンド](../../requirements/command/index.md)
  - `CTR-CMD-001` コマンドの一覧
  - `CTR-CMD-003` 引数の誤り
- [書式検査](../../requirements/check/index.md)
  - `CTR-CHK-001` 検出するエラー (対応づけの前に適用する)
- [構造化データの出力](../../requirements/export/index.md)
  - `CTR-EXP-001` 出力する内容 (要件のキーと値を借りる)
  - `CTR-EXP-002` 同じ文書からは同じ出力 (要件の並べ方を借りる)

## 実装の状況

いまのコードは、`CTR-TRC-010` の規則と `changed_after_done` のほかは、この設計のすべてを持つ。

| 部分 | 状況 |
|---|---|
| コマンド、設定の `requirement_label`、環境変数、リポジトリの決め方 | 実装した |
| 問い合わせ 1 `requirementIssues`、2 `moreSubIssues`、3 `closers`、IDの見つけ方 (`ListedIDs`) | 実装した |
| JSON の `repository`、`read_at`、`requirements` の出力のキー (`export.Of`)、`progress`、`children`、`requirement_issues` (`sub_issues` を含む)、`implementation_issues` | 実装した |
| 書式のエラーでの停止 (`CTR-TRC-009`) | 実装した |
| 進捗 (`CTR-TRC-004`) と子の進捗の数 (`CTR-TRC-005`)。`ProgressOf` と `setProgress` (`internal/trace/domain.go`) | 実装した |
| sub-issue と Pull Request (`CTR-TRC-003`)、実装Issueの数 (`CountImplementation`) | 実装した |
| 不整合 (`CTR-TRC-006`) と JSON の `discrepancies`。`Link` と `sortDiscrepancies` (`internal/trace/domain.go`) | 実装した |
| git の履歴から、IDごとに最後に変わった日時を読む (`CTR-TRC-010`)。`lastChangesOf` (`internal/trace/history.go`) と `LastChanges` (`internal/trace/domain.go`)。履歴が足りないときのエラー | 実装した |
| 変わった日時と要求Issueの閉じた日時を比べる規則 (`CTR-TRC-010`) と、JSON の `changed_after_done` | まだない。読んだ日時は、まだ出力に使わない |

## 全体像

[ソースコード全体構成](../../architecture/overview.md) の `internal/trace`、`internal/export` (要件のオブジェクトと並べ方)、`internal/platform`、`internal/core` (設定) にあたる。

![要件と要求Issueの対応づけのパッケージ](trace-packages.svg)

図の元ファイル: [trace-packages.puml](trace-packages.puml)

書式検査を先に済ませ、エラーがなければ git の履歴を読み、そのあとで GitHub を読む (`CTR-TRC-009`、`CTR-TRC-010`)。

![要件と要求Issueの対応づけのシーケンス](trace-sequence.svg)

図の元ファイル: [trace-sequence.puml](trace-sequence.puml)

## 決めたこと

### 要件のオブジェクトは、構造化データの出力の関数で作る

`Link` は、要件を `export.Sort` で並べ、`export.Of` で作ったオブジェクトの後ろに、対応づけのキーを足す (`CTR-TRC-008`)。Go では、`RequirementOutput` が `export.Requirement` を埋め込む。`encoding/json` は、埋め込んだ構造体のキーを、その位置に同じ順で書く。

- 理由: 要件の JSON の形を1か所に置けば、`export` と `trace` の出力が食い違わない。廃止したかどうかは `status` が持つので、`deprecated` のキーは持たない。
- 採らなかった案: `export` が書いた JSON のファイルを読む。ファイルの受け渡しが要り、`trace` は文書を読んで書式検査をするままでよい。
- 細部: [コマンドと出力](command.md#json)

### 要件と要求Issueの対応づけは、独立したコマンド `cumin-trace trace`

引数を取らない。結果の JSON は標準出力に書く。

- 理由: `check` と `export` は token もネットワークも要らない (`CTR-CHK-004`)。GitHub を読むコマンドを分けると、その2つは手元でも Pull Request の check でも、token なしで動き続ける。
- 採らなかった案: `export` に GitHub の読み取りを足す。`CTR-EXP-002` (同じ文書からは同じ出力) と、読んだ時刻を持つ出力 (`CTR-TRC-008`) が両立しない。
- 採らなかった案: 出力先のファイルを引数で受け取る。標準出力をリダイレクトすれば足りる。
- `CTR-CMD-001` の表は、`trace` の行を持つ。
- 細部: [コマンドと出力](command.md)

### 設定は1つのキーを足す。リポジトリと token は環境から決める

`.cumin-trace.toml` に `requirement_label` (既定 `cumin/type/requirement`) だけを足す。読むリポジトリは、環境変数 `GITHUB_REPOSITORY`、なければ現在のディレクトリの `origin` の remote から決める。token は環境変数 `GITHUB_TOKEN` から読む。

- 理由: 要求Issueは、要件文書と同じリポジトリにある。GitHub Actions は `GITHUB_REPOSITORY` を必ず渡し、手元のクローンは `origin` を持つので、設定に書かなくても決まる。token はファイルに書かない (`CTR-TRC-007`)。
- 採らなかった案: 設定の `repository` キー。リポジトリの名前を変えたり fork したりすると、設定と実際のリポジトリが食い違う。
- 細部: [コマンドと出力](command.md)

### GitHub は GraphQL API だけで読む

読むのは、要求Issueの一覧と sub-issue の一覧 (`requirementIssues` と、101件目からの `moreSubIssues`)、閉じた実装Issueを閉じた Pull Request (`closers`) である。問い合わせは `query` だけで、`mutation` を持たない。

- 理由: ラベル、閉じた理由、マイルストーン、sub-issue を、100件の要求Issueにつき1回の呼び出しで読める。REST API では、要求Issueごとに sub-issue の呼び出しが、実装Issueごとに timeline の呼び出しが要る。
- 細部: [GitHub の読み方](github.md)

### IDは、本文の `## Requirements` の節の、箇条書きの項目の先頭だけから探す

要求Issueの本文を `goldmark` で読み、`## Requirements` の節の項目の先頭のIDだけを対応にする (`CTR-TRC-002`)。読む関数は `internal/requirement/markdown.go` に置き、IDの形は `IDForm` で確かめる。

- 理由: 節と項目とコードブロックの区別には、Markdown の構文木が要る。要件文書と同じ読み方 (CommonMark、拡張なし) で、GitHub での見え方と合う。`goldmark` の型は `markdown.go` の外に出さない ([パッケージとファイル](../check/packages.md))。
- 採らなかった案: 本文の全体に `IDForm.References` を適用する。ほかの節の説明やコードブロックの例のIDも、対応になってしまう。
- 細部: [進捗と不整合](rules.md)

### パッケージは `internal/trace` と `internal/platform`

GraphQL の client は `internal/platform/github.go` に置き、その中で定義した Go の型 (GraphQL の型ではない) を返す。Issueのモデルは `internal/trace/domain.go` に置き、`internal/trace/convert.go` が `internal/platform` の型からモデルに変える。

- 理由: プラットフォームのパッケージは具体的な型を返し、機能ごとに自分のモデルへ変える。GraphQL の型はプラットフォームで止まり、機能の規則 (`domain.go`) は、プロバイダのパッケージにも `internal/platform` の型にも依存しない。ほかの機能も、あとで同じ client で GitHub を読める。
- 採らなかった案: 共有のモデルを `internal/core` に置く。使うのは対応づけだけで、プロバイダを知らない土台ではない。
- 採らなかった案: `internal/trace` が持つ interface。2つ目の実装か、テストの本当の必要が出るまで作らない。テストは偽の GitHub の後ろに本物の client を置く。interface は、あとで `internal/trace` の側に、`internal/platform` を変えずに足せる。
- 細部: [パッケージとファイル](packages.md)

### 変わった日時は、`git` のコマンドで履歴を読んで決める

要件文書のディレクトリ (`requirements_dir`) に触れた commit を、古い順に読む。commit ごとに、その commit と最初の親の要件文書を `requirement.Parse` で読み、IDごとに比べる (`CTR-TRC-010`)。

- 変わったとするのは、タイトル、本文、検証 (方法と説明) のどれかが親と違うとき、または親にそのIDがないときである。ファイルや行の位置は比べない。そのため、別のファイルや別の行への移動は変更にならない。
- commit のメッセージに `Requirement-Unchanged: <ID>` の行があれば、その commit でのそのIDの変更は数えない。行の前後の空白は無視する。1つの commit に複数の行を書ける。
- 変わった日時は、commit の日時 (`%cI`) である。author の日時ではない。rebase や squash merge で main に入った時刻に近いためである。
- shallow なリポジトリ (`git rev-parse --is-shallow-repository` が `true`) では、`shallow git repository: fetch the whole history of the requirement documents (fetch-depth: 0)` のエラーで終わる。git のリポジトリでないときは、`no git repository: the trace reads the git history of the requirement documents` のエラーで終わる。どちらも終了コード 2 で、GitHub を呼ばない。
- 履歴を読むのは、リポジトリと token を確かめたあと、GitHub を読む前である。
- 理由: 新しいライブラリが要らない。`git` は、GitHub Actions のランナーにも手元にもある。比べる規則は純粋な関数 (`LastChanges`) にして、`git` を実行する部分 (`history.go`) と分ける。
- 採らなかった案: Go の git ライブラリ。ライブラリを足すことになり、親 (#182) の制約で、足す前に Maintainer に聞く必要がある。
- 採らなかった案: `git log -p` の差分の行から変更を探す。移動と変更の区別や、要件の範囲の判断が、要件文書の読み方と食い違う。
- 範囲外: merge commit で取り込む履歴 ([先送りした要件](../../requirements/backlog.md))。merge commit は、親の1つと同じ内容なら `git log` が省く。
- 細部: [パッケージとファイル](packages.md)
