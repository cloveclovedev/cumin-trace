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

- 設定の `prefix` のIDだけを見つける。ほかのプロダクトのIDは無視する。
- 1つの本文で同じIDを何度書いても、1つと数える。
- 子要件のID (`PC-TASK-003.1`) は、親のID (`PC-TASK-003`) を含まない。子のIDだけを見つける。
- 本文が空なら、IDはない。

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

- 子を持つ要件は、`progress` が `null` で、`children` に子の進捗ごとの数を持つ (`CTR-TRC-005`)。廃止した子は数えない。`total` は、廃止していない子の数である。
- 廃止した要件は、`progress` と `children` が `null` である。廃止した親の子も、廃止していなければ、ほかの要件と同じに進捗を持つ。
- 子を持つかどうかは、`ParentID` がその要件のIDである要件が、文書に1つでもあるかで決める。

## 実装Issueの数

要件の `implementation_issues` は、その要件の `requirement_issues` のすべての `sub_issues` を、リポジトリと番号で重ねずに数える。`closed` は閉じたもの (閉じた理由を問わない)、`total` はすべてである。not planned で閉じた要求Issueの sub-issue も数える。

## テスト

要件ごとのテストは、`internal/trace/service_test.go` に置く。偽の GitHub (`httptest`) の後ろに、本物の `internal/platform` の client を置く。

| 要件 | 確かめる状態 |
|---|---|
| `CTR-TRC-001` | ラベルのあるIssueとないIssue。ラベルで絞った問い合わせだけを偽の GitHub が答え、ラベルのあるIssueだけが出る |
| `CTR-TRC-002` | 2つのIDを書いた要求Issueと、同じIDを書いた2つの要求Issue。チェックボックスのある項目、ない項目、コードスパンのID |
| `CTR-TRC-002` | ほかの節のID、項目の途中のID、節の中のコードブロックのIDが、対応にならない。`## Requirements` の節が次のレベル 2 の見出しで終わる |
| `CTR-TRC-003` | sub-issue のある要求Issue (Pull Request で閉じたものと、開いたもの) と、sub-issue のない要求Issue |
| `CTR-TRC-004.1` | 要求Issueのない要件と、not planned で閉じた要求Issueだけを持つ要件 |
| `CTR-TRC-004.2` | 開いた要求Issueと、completed で閉じた要求Issueを持つ要件 |
| `CTR-TRC-004.3` | completed で閉じた要求Issueだけを持つ要件 |
| `CTR-TRC-005` | 3つの進捗の子と、廃止した子を持つ要件 |
| `CTR-TRC-006` | 文書にないIDを書いた要求Issueと、廃止した要件に対応する開いた要求Issue |
| `CTR-TRC-008` | 上の状態をまとめた出力の全体を、決めた時刻で比べる |
| `CTR-TRC-009` | エラーを1つ含む文書。偽の GitHub への呼び出しが0回、標準出力が空、エラーの行がある、終了コード 1 |

`CTR-TRC-007` は `inspection` で、`internal/platform/github.go` の問い合わせの文に `mutation` がないことと、token がヘッダの外に出ないことを読んで確かめる。
