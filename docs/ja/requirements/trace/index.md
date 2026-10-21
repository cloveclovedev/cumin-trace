# GitHub との対応づけ

要件が、どの要求Issueと実装Issueで実現されているか、どこまで進んでいるかを、GitHub から読み取るための要件を定める。cumin-trace の中心になる機能である。

## 全体像

要件と要求Issueは、要求Issueの本文に書いたIDで結び付く。要求Issueから下は、GitHub の sub-issue と Pull Request のリンクをたどる。

![要件とIssueの結び付き](trace-links.svg)

図の元ファイル: [trace-links.puml](trace-links.puml)

進捗は、要件ごとに次の順で決める。使うのは、要求Issueが開いているか、閉じた理由と日時、要件が最後に変わった日時である。

![進捗の決め方](trace-progress.svg)

図の元ファイル: [trace-progress.puml](trace-progress.puml)

図の中の名前と、文書での呼び方の対応は次のとおりである。

| 図 | 呼び方 | 意味 |
|---|---|---|
| not filed | 未起票 | 要求Issueがまだない |
| in progress | 進行中 | 開いた要求Issueがある |
| done | 完了 | 要求Issueが受け入れられて閉じた |
| not filed (changed) | 変更後の未起票 | 完了した要求Issueが閉じたあとに、要件が変わった |

要求Issueがあれば、あとは cumin-works の流れか、Maintainer の作業で進む。そこで、起票したあとの状態は細かく分けない。`cumin/status/*` のラベルは読まない。

GitHub は、cumin-trace が API を直接呼んで読む。GitHub App は使わない。token は、GitHub Actions では workflow に渡される `GITHUB_TOKEN` を、手元では `gh auth token` で得られるものを、環境変数で渡す。手元で動かすのは Maintainer で、Maintainer 自身のアカウントの権限で読む。cumin-works を動かす Host のアカウントで cumin-trace を動かす必要はない。

## 要件

### `CTR-TRC-001` 要求Issueの見分け方

設定で指定したラベルの付いたIssueを、要求Issueとして扱う。既定のラベルは `cumin/type/requirement` とする。開いているIssueも、閉じたIssueも読む。

- 検証: `test` 偽の GitHub で、ラベルのあるIssueだけが要求Issueになる

---

### `CTR-TRC-002` 要件と要求Issueの結び付き

要求Issueの本文の `## Requirements` の節で、箇条書きの項目の先頭にIDが書いてあれば、その要求Issueはその要件に対応する。項目にチェックボックスがあってもなくてもよい。節は、次の同じか上の階層の見出しで終わる。ほかの節のID、項目の途中にある説明のための参照、コードブロックの中のIDは、対応にならない。1つの要求Issueが複数の要件に対応してよく、1つの要件に複数の要求Issueが対応してもよい。

- 検証: `test` 2つのIDを書いた要求Issueと、同じIDを書いた2つの要求Issueで確かめる
- 検証: `test` ほかの節のID、項目の途中のID、コードブロックの中のIDが対応にならないことを確かめる

---

### `CTR-TRC-003` 実装Issueと Pull Request

要求Issueに sub-issue があれば、実装Issueとして読む。sub-issue のない要求Issueも、そのまま扱える。実装Issueごとに、開いているか閉じているかと、そのIssueを閉じた Pull Request を読む。要求Issueと実装Issueのマイルストーンも読む。

- 検証: `test` 偽の GitHub で、sub-issue と Pull Request が読み取れる

---

### `CTR-TRC-004` 進捗

子を持たない要件ごとに、進捗を次の3つのうちの1つに決める。廃止した要件には、進捗を付けない。廃止した要件は、未起票の一覧にも出ない。

#### `CTR-TRC-004.1` 未起票

対応する要求Issueが1つもない要件は、未起票とする。対応する要求Issueがすべて not planned で閉じている要件も、未起票とする。

- 検証: `test` 要求Issueのない要件と、not planned で閉じた要求Issueだけを持つ要件が、未起票になる

---

#### `CTR-TRC-004.2` 進行中

対応する要求Issueのうち、開いているものが1つでもある要件は、進行中とする。

- 検証: `test` 開いた要求Issueを持つ要件が、進行中になる。completed で閉じた要求Issueがほかにあっても、進行中になる

---

#### `CTR-TRC-004.3` 完了

対応する要求Issueに開いているものがなく、completed で閉じたものが1つ以上ある要件は、完了とする。

- 検証: `test` completed で閉じた要求Issueだけを持つ要件が、完了になる

---

### `CTR-TRC-005` 子を持つ要件の進捗

子を持つ要件にも、3つの進捗のうちの1つを付ける。廃止していない子の進捗と、その要件の要求Issueから `CTR-TRC-004` で決まる進捗を合わせて見る。要求Issueから決まる進捗が未起票なら、合わせない。すべて完了なら完了、すべて未起票なら未起票、それ以外は進行中とする。あわせて、子の進捗を数えて表す (例: 完了 3 / 5)。

- 検証: `test` 進捗の違う子を持つ要件で、進捗と数が正しい。子がすべて完了で、開いた要求Issueを持つ要件が、進行中になる

---

### `CTR-TRC-006` 不整合の報告

次の不整合を報告する。

- 要件文書にないIDを書いた要求Issue
- 廃止した要件に対応する、開いた要求Issue

- 検証: `test` それぞれの不整合を1つ含む状態で、報告される

---

### `CTR-TRC-007` GitHub には書き込まない

cumin-trace は GitHub を読むだけで、Issue、ラベル、コメントのどれも変更しない。token は環境変数で受け取り、ファイルに保存せず、ログにも出さない。必要な権限は、Issue と Pull Request の読み取りだけである。

- 検証: `inspection` GitHub を呼び出すコードに、書き込みの呼び出しがない

---

### `CTR-TRC-008` 結果の出力

対応づけの結果を、読んだリポジトリと GitHub を読んだ時刻とともに JSON で出力する。要件ごとに、構造化データの出力 (`CTR-EXP-001`) と同じキーをすべて同じ順に持ち、その後ろに対応づけのキーを足す。対応づけのキーは、進捗、変更後の未起票か (`CTR-TRC-010`)、子の進捗の数 (`CTR-TRC-005`)、対応する要求Issue (状態、マイルストーン、実装Issue)、実装Issueの数 (閉じた数と全体の数) である。不整合 (`CTR-TRC-006`) の一覧も持つ。不整合ごとに、種類、要求Issueの番号、IDを持つ。

この出力だけで、要件の構造と進捗が分かる。

例は、[構造化データの出力](../export/index.md) の例と同じ2つの要件である。キーの名前と値の形は、この例のとおりとする。短くするため、実装Issueは1つだけを載せた。不整合の2件は、形を示すための仮の例である。

```json
{
  "repository": "cloveclovedev/cumin-trace",
  "read_at": "2026-10-17T09:10:55Z",
  "requirements": [
    {
      "id": "CTR-TRC-004",
      "parent": null,
      "title": "進捗",
      "text": "子を持たない要件ごとに、進捗を次の3つのうちの1つに決める。廃止した要件には、進捗を付けない。廃止した要件は、未起票の一覧にも出ない。",
      "verifications": [],
      "status": "active",
      "replaced_by": null,
      "file": "docs/ja/requirements/trace/index.md",
      "line": 58,
      "progress": "done",
      "changed_after_done": false,
      "children": {"not_filed": 0, "in_progress": 0, "done": 3, "total": 3},
      "requirement_issues": [],
      "implementation_issues": {"closed": 0, "total": 0}
    },
    {
      "id": "CTR-TRC-004.1",
      "parent": "CTR-TRC-004",
      "title": "未起票",
      "text": "対応する要求Issueが1つもない要件は、未起票とする。対応する要求Issueがすべて not planned で閉じている要件も、未起票とする。",
      "verifications": [
        {"method": "test", "how": "要求Issueのない要件と、not planned で閉じた要求Issueだけを持つ要件が、未起票になる"}
      ],
      "status": "active",
      "replaced_by": null,
      "file": "docs/ja/requirements/trace/index.md",
      "line": 62,
      "progress": "done",
      "changed_after_done": false,
      "children": null,
      "requirement_issues": [
        {
          "number": 4,
          "state": "closed",
          "state_reason": "completed",
          "milestone": "v0.1",
          "sub_issues": [
            {
              "repository": "cloveclovedev/cumin-trace",
              "number": 63,
              "state": "closed",
              "milestone": "v0.1",
              "closed_by": {"repository": "cloveclovedev/cumin-trace", "number": 70}
            }
          ]
        }
      ],
      "implementation_issues": {"closed": 1, "total": 1}
    }
  ],
  "discrepancies": [
    {"kind": "unknown_id", "issue": 31, "id": "CTR-TRC-099"},
    {"kind": "open_issue_for_deprecated", "issue": 18, "id": "CTR-TRC-002"}
  ]
}
```

- `progress` は `not_filed`、`in_progress`、`done` のどれかである。廃止した要件では `null` になる。
- `changed_after_done` は、`CTR-TRC-010` で未起票になった要件で `true` になる。ほかの要件では `false` になる。
- `children` は、子を持つ要件だけが持つ。子を持たない要件では `null` になる。
- 1つの要求Issueが複数の要件を挙げているときは、その要件ごとに、同じ要求Issueを sub-issue まで含めて繰り返す。

- 検証: `test` 偽の GitHub で、出力を確かめる。要件ごとに、構造化データの出力と同じキーと値を持つ。不整合を1つ含む状態で、一覧に出る

---

### `CTR-TRC-009` 書式の誤りがあるとき

書式検査 (`CTR-CHK-001`) でエラーになる文書では、対応づけをしない。GitHub を読まず、結果も出力しない。書式検査と同じエラーを報告し、0 以外の終了コードで終わる。誤ったIDのまま要求Issueと突き合わせると、進捗と不整合が誤って出るためである。

- 検証: `test` エラーを1つ含む文書で、偽の GitHub への呼び出しがなく、結果が出ず、エラーが報告され、終了コードが 0 以外になる

---

### `CTR-TRC-010` 変更後の未起票

要件のタイトル、本文、検証のどれかが、completed で閉じた要求Issueのうち最後に閉じたものより後に変わっていて、開いた要求Issueがなければ、その要件は未起票とする。`CTR-TRC-004.3` で完了になる要件にも当てはめる。変わった日時は、git の履歴で、その変更を含む commit の日時とする。commit のメッセージに `Requirement-Unchanged: <ID>` の行があれば、その commit でのそのIDの変更は数えない。git の履歴が足りないときは、エラーで終わる。

- 検証: `test` 要求Issueが閉じたあとに本文を変えた要件が、未起票になる。開いた要求Issueがあれば、進行中になる。`Requirement-Unchanged` の行を持つ commit の変更は数えない
- 検証: `test` ファイルや行の位置だけが変わった commit は、変更として数えない

---
