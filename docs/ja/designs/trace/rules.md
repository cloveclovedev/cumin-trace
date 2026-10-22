# 進捗と不整合

[要件と要求Issueの対応づけの設計](index.md) の細部。IDの見つけ方は `internal/requirement/markdown.go` の `ListedIDs(body []byte, form IDForm) []string` で、ほかの決まりは `internal/trace/domain.go` の純粋な関数である。

## IDの見つけ方

要求Issueの本文を `goldmark` v1 (CommonMark、拡張なし) で読み、次の決まりでIDを見つける (`CTR-TRC-002`)。

1. 節: テキストが `Requirements` の、レベル 2 の見出し (`## Requirements`) から始まる。次のレベル 1 か 2 の見出しの直前で終わる。そのような見出しがなければ、本文の最後で終わる。見出しの前後の空白は無視し、大文字と小文字は区別する。節が2つ以上あれば、どれも読む。要件文書の読み方 (`readDocument`) と同じく、最上位の見出しだけを数え、引用や項目の中の見出しは節を始めも終わりもしない。
2. 項目: 節の中の箇条書き (`-`、`*`、`+`、番号付き) の、すべての項目を読む。入れ子の項目も読む。
3. 先頭: 項目の最初の子が段落 (`ast.Paragraph`、または密な箇条書きの `ast.TextBlock`。`readFields` と同じ) のときだけ、その段落の先頭を見る。先頭の空白と、チェックボックス (`[ ]`、`[x]`、`[X]` と、その後ろの空白) を飛ばす。
4. ID: 残りが、正しいIDで始まれば、そのIDを対応にする。IDはコードスパン (`` `PC-TASK-003` ``) に入っていてもよい。IDの直後は、英字と数字ではなく、`.` と数字でもない (`IDForm.References` と同じ境目)。`- PC-TASK-003-draft` では `PC-TASK-003` を見つける。
5. 1つの項目からは、先頭の1つのIDだけを読む。

次のものは、対応にならない。

| 書き方 | 例 |
|---|---|
| ほかの節のID | `## Context` の中の `PC-TASK-003` |
| 項目の途中のID | `- see PC-TASK-003 for the reason` |
| コードブロックの中のID | 節の中のフェンス、または項目の中のコードブロック |
| 段落ではなく、見出しやコードブロックで始まる項目 | `- ## PC-TASK-003` |
| 後ろに空白のないチェックボックスの直後のID | `- [x]PC-TASK-003` |

- 設定の `prefix` のIDだけを見つける。ほかのプロダクトのIDは無視する。
- 1つの本文で同じIDを何度書いても、1つと数える。
- 子要件のID (`PC-TASK-003.1`) は、親のID (`PC-TASK-003`) を含まない。子のIDだけを見つける。
- 本文が空なら、IDはない。
- ページが同じ番号の要求Issueを2回返したら、最初の1つだけを使う。`requirement_issues` にも不整合にも、1回だけ出る。

見つけたIDごとに、次のどれかにする。

| IDの要件 | すること |
|---|---|
| 要件文書にない | 不整合 `unknown_id` にする。要件には結び付けない |
| 廃止した要件 | その要件の `requirement_issues` に入れる。要求Issueが開いていれば、不整合 `open_issue_for_deprecated` にもする |
| それ以外 | その要件の `requirement_issues` に入れる |

親のIDを書いた要求Issueは、親の `requirement_issues` に入る。子の進捗には影響しない。

## 進捗

子を持たず、廃止していない要件ごとに、`requirement_issues` から決める (`CTR-TRC-004`)。上から順に、最初に合う行を使う。

| 条件 | 進捗 |
|---|---|
| 開いた要求Issueが1つ以上ある | `in_progress` (`CTR-TRC-004.2`) |
| completed で閉じた要求Issueが1つ以上ある | `done` (`CTR-TRC-004.3`) |
| それ以外 (要求Issueがない、または not planned で閉じたものだけ) | `not_filed` (`CTR-TRC-004.1`) |

- 子を持つ要件は、`children` に子の進捗ごとの数を持つ (`CTR-TRC-005`)。廃止した子は数えない。`total` は、廃止していない子の数である。
- 子を持つ要件の `progress` は、廃止していない子の進捗と、その要件の `requirement_issues` から上の表で決まる進捗を合わせて決める (`CTR-TRC-005`)。要求Issueから決まる進捗が `not_filed` なら、合わせない。上から順に、最初に合う行を使う。

| 合わせる進捗 | 進捗 |
|---|---|
| 1つもない (子がすべて廃止で、要求Issueから決まる進捗が `not_filed`) | `not_filed` |
| すべて `done` | `done` |
| すべて `not_filed` | `not_filed` |
| それ以外 | `in_progress` |

- 廃止した要件は、`progress` と `children` が `null` である。廃止した親の子も、廃止していなければ、ほかの要件と同じに進捗を持つ。
- 子を持つかどうかは、`ParentID` がその要件のIDである要件が、文書に1つでもあるかで決める。

## 変更後の未起票

廃止していない要件ごとに、その要件の `requirement_issues` と、git の履歴から読んだ最後に変わった日時 ([変わった日時](index.md#変わった日時はgit-のコマンドで履歴を読んで決める)) から決める (`CTR-TRC-010`)。次のすべてに合えば、上の表で決めた進捗に代えて `not_filed` にし、`changed_after_done` を `true` にする (`ChangedAfterDone`)。

| 条件 | 内容 |
|---|---|
| 開いた要求Issueがない | 1つでも開いていれば、当てはめない |
| completed で閉じた要求Issueが1つ以上ある | not planned で閉じたものは数えない |
| 最後に変わった日時が、completed で閉じた要求Issueの `closedAt` のうち最も遅いものより後 | 同じ日時は、後ではない |

- 子を持たない要件では、子を持つ要件の進捗を決める前に当てはめる。親の `children` は、当てはめたあとの子の進捗を数える。
- 子を持つ要件では、その要件自身の要求Issueだけで当てはめる。子の進捗がどうであっても、合えば `not_filed` である。自身の completed で閉じた要求Issueがなければ当てはめず、上の表の進捗のままである。
- 子の変更は、親の変わった日時にならない。変わった日時は、IDごとに決まる。
- ほかの要件の `changed_after_done` は `false` である。廃止した要件も `false` である。

## 実装Issueの数

要件の `implementation_issues` は、その要件の `requirement_issues` のすべての `sub_issues` を、リポジトリと番号で重ねずに数える。`closed` は閉じたもの (閉じた理由を問わない)、`total` はすべてである。not planned で閉じた要求Issueの sub-issue も数える。

## テスト

要件ごとのテストは、`internal/trace/service_test.go` に置く。偽の GitHub (`httptest`) の後ろに、本物の `internal/platform` の client を置く。

| 要件 | 確かめる状態 |
|---|---|
| `CTR-TRC-001` | ラベルのあるIssueとないIssue。ラベルで絞った問い合わせだけを偽の GitHub が答え、ラベルのあるIssueだけが出る |
| `CTR-TRC-001` | `hasNextPage` が `true` で、`endCursor` が空か前と同じ。`GitHub: invalid page cursor` で止まり、JSON が出ない |
| `CTR-TRC-002` | 2つのIDを書いた要求Issueと、同じIDを書いた2つの要求Issue。チェックボックスのある項目、ない項目、コードスパンのID |
| `CTR-TRC-002` | ほかの節のID、項目の途中のID、節の中のコードブロックのID、後ろに空白のないチェックボックスの直後のIDが、対応にならない。`## Requirements` の節が次のレベル 2 の見出しで終わる |
| `CTR-TRC-003` | sub-issue のある要求Issue (Pull Request で閉じたもの、コミットで閉じたもの、開いたもの、ほかのリポジトリのもの。2ページにまたがる) と、sub-issue のない要求Issue |
| `CTR-TRC-003` | 2つの要求Issueの同じ閉じた sub-issue は、`closers` に1回だけ渡す。閉じた sub-issue がなければ、`closers` を呼ばない |
| `CTR-TRC-004.1` | 要求Issueのない要件と、not planned で閉じた要求Issueだけを持つ要件 |
| `CTR-TRC-004.2` | 開いた要求Issueと、completed で閉じた要求Issueを持つ要件 |
| `CTR-TRC-004.3` | completed で閉じた要求Issueだけを持つ要件 |
| `CTR-TRC-005` | 3つの進捗の子と、廃止した子を持つ要件。進捗は `in_progress` で、子の数は変わらない |
| `CTR-TRC-005` | 子がすべて完了で、要求Issueのない要件 (`done`)、開いた要求Issueを持つ要件 (`in_progress`)、not planned で閉じた要求Issueだけを持つ要件 (`done`)。子がすべて未起票で、completed で閉じた要求Issueを持つ要件 (`in_progress`) と、要求Issueのない要件 (`not_filed`)。廃止した子だけを持ち、要求Issueのない要件 (`not_filed`) |
| `CTR-TRC-010` | 要求Issueが閉じたあとに本文を変えた要件 (`not_filed`、`true`)。開いた要求Issueを持つ要件 (`in_progress`、`false`)。変更のあとに、もう1つの要求Issueが閉じた要件 (`done`)。not planned で閉じた要求Issueだけを持つ要件 (`not_filed`、`false`)。変えていない要件 (`done`)。自身の閉じた要求Issueを持ち、子がすべて完了の親 (`not_filed`、`true`)。自身の要求Issueがないか not planned だけで、子がすべて完了の親 (`done`、`false`)。廃止した要件 (`null`、`false`) |
| `CTR-TRC-006` | 文書にないIDを書いた要求Issueと、廃止した要件に対応する開いた要求Issue。廃止した要件に対応する閉じた要求Issueは、不整合にならない |
| `CTR-TRC-006` | ページが2回返した同じ番号の要求Issue。`requirement_issues` と不整合に1回だけ出て、最初の本文を使う |
| `CTR-TRC-008` | 上の状態をまとめた出力の全体を、決めた時刻で比べる。`discrepancies` に、2種類の不整合が1つずつ出る |
| `CTR-TRC-008` | 不整合がなければ、`discrepancies` は空の配列 `[]` である |
| `CTR-TRC-008` | 2つの要求Issueの同じ sub-issue と、not planned で閉じた要求Issueの sub-issue を持つ要件の `implementation_issues`。sub-issue のない要件は 0 / 0 |
| `CTR-TRC-009` | エラーを1つ含む文書。偽の GitHub への呼び出しが0回、標準出力が空、エラーの行がある、終了コード 1 |

`CTR-TRC-007` には、次のテストもある。

| 要件 | 確かめる状態 |
|---|---|
| `CTR-TRC-007` | token がヘッダにだけ入り、問い合わせが `query` の `POST` だけである。HTTP 401 のエラーにも token が入らない (`internal/trace/service_test.go`) |
| `CTR-TRC-007` | `origin` の URL の token が、リポジトリに入らない (`internal/trace/service_test.go`) |
| `CTR-TRC-007` | 接続の失敗が種類 (`timeout` など) を持ち、URL も token も持たない (`internal/platform/github_test.go`) |

リポジトリの決め方は、要件ではなく [コマンドと出力](command.md#リポジトリ) の決まりで、`internal/trace/service_test.go` で確かめる。

`CTR-TRC-007` の要件の検証は `inspection` で、`internal/platform/github.go` の問い合わせの文に `mutation` がないことと、token がヘッダの外に出ないことを読んで確かめる。
