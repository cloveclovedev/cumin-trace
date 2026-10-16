# 進捗と不整合

[対応づけの設計](index.md) の細部。ここに書く決まりは、すべて `internal/trace/domain.go` の純粋な関数である。

## IDの見つけ方

要求Issueの本文の全体に、`IDForm.References(body, 0, len(body))` を適用する (`CTR-TRC-002`)。

- 本文は Markdown として読まない。コードブロックの中のIDも数える。
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
| `CTR-TRC-002` | 2つのIDを書いた要求Issueと、同じIDを書いた2つの要求Issue |
| `CTR-TRC-003` | sub-issue のある要求Issue (Pull Request で閉じたものと、開いたもの) と、sub-issue のない要求Issue |
| `CTR-TRC-004.1` | 要求Issueのない要件と、not planned で閉じた要求Issueだけを持つ要件 |
| `CTR-TRC-004.2` | 開いた要求Issueと、completed で閉じた要求Issueを持つ要件 |
| `CTR-TRC-004.3` | completed で閉じた要求Issueだけを持つ要件 |
| `CTR-TRC-005` | 3つの進捗の子と、廃止した子を持つ要件 |
| `CTR-TRC-006` | 文書にないIDを書いた要求Issueと、廃止した要件に対応する開いた要求Issue |
| `CTR-TRC-008` | 上の状態をまとめた出力の全体を、決めた時刻で比べる |
| `CTR-TRC-009` | エラーを1つ含む文書。偽の GitHub への呼び出しが0回、標準出力が空、エラーの行がある、終了コード 1 |

`CTR-TRC-007` は `inspection` で、`internal/platform/github.go` の問い合わせの文に `mutation` がないことと、token がヘッダの外に出ないことを読んで確かめる。
