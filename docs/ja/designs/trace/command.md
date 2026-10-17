# コマンドと出力

[要件と要求Issueの対応づけの設計](index.md) の細部。

## 引数

```sh
cumin-trace trace
```

引数を取らない。`trace` のあとに引数があれば、`CTR-CMD-003` のとおり、使い方を標準エラー出力に出して、終了コード 2 で終わる。読む先は、現在のディレクトリである。`main.go` は、`check` と同じく `root` に `.` を渡す。

使い方の行は、`trace` を足して次のようにする。

```text
cumin-trace: usage: cumin-trace check | cumin-trace export | cumin-trace init <prefix> [<area>] | cumin-trace trace | cumin-trace version
```

## 設定

`.cumin-trace.toml` の構造体 (`internal/core/settings.go`) に、次の1つのキーを足す。書式検査も同じ構造体を読むので、このキーがあっても知らないキーにならない。書式検査は、このキーの値を確かめない。

```toml
# 要求Issueのラベル。省略できる
requirement_label = "cumin/type/requirement"
```

| キー | 型 | 必須 | 決まり |
|---|---|---|---|
| `requirement_label` | 文字列 | 省略できる | 空ではない文字列。既定は `cumin/type/requirement` (`CTR-TRC-001`) |

## リポジトリ

要求Issueは、要件文書と同じリポジトリにある。読むリポジトリは、設定に書かず、次の順で決める。

1. 環境変数 `GITHUB_REPOSITORY`。GitHub Actions が `octocat/Hello-World` の形で渡す。空でなければ使い、`<owner>/<name>` の形でなければエラーにする (次の手に進まない)
2. 現在のディレクトリの `git remote get-url origin` の URL。下の表の github.com の形のときだけ使う
3. どちらもなければ、`no repository: set GITHUB_REPOSITORY to <owner>/<name>` のエラー

`<owner>/<name>` の形: `/` がちょうど1つで、両側は空ではない。

`origin` の URL で使う形 (`.git` は、あってもなくてもよい):

| 形 | 例 |
|---|---|
| https | `https://github.com/octocat/Hello-World.git` |
| ssh (scp の書き方) | `git@github.com:octocat/Hello-World.git` |
| ssh (URL の書き方) | `ssh://git@github.com/octocat/Hello-World.git` |

- github.com 以外のホスト、ほかの形、`git` がない、`origin` がない、のどれも、2 では決まらないとして 3 に進む。
- `git remote get-url origin` は `main.go` が実行し、URL を `trace.Env.Remote` に入れる。失敗したときは空にする。URL を読む規則 (`RepositoryFromRemote`) は `internal/trace/domain.go` の純粋な関数である。
- URL に利用者名や token が入っていても (`https://<user>:<token>@github.com/...`)、使うのは `<owner>/<name>` だけで、URL は出力にもエラーにも入れない。

## 環境変数

環境変数は `main.go` だけが読み、`trace.Env` に入れて `trace.Run` に渡す。`internal/trace` と `internal/platform` は、環境変数を読まない。

| 環境変数 | 使い方 |
|---|---|
| `GITHUB_TOKEN` | GitHub を読む token。必須 |
| `GITHUB_REPOSITORY` | 読むリポジトリ。なければ `origin` の remote から決める ([リポジトリ](#リポジトリ)) |
| `GITHUB_GRAPHQL_URL` | GraphQL API の URL。既定は `https://api.github.com/graphql`。テストでは偽の GitHub の URL を渡す |

token の扱い (`CTR-TRC-007`):

- GitHub Actions では、workflow が `GITHUB_TOKEN: ${{ github.token }}` を渡す。権限は `issues: read` と `pull-requests: read` だけにする。
- 手元では、`GITHUB_TOKEN=$(gh auth token) cumin-trace trace` のように渡す。
- token がない、または空のときは、`GITHUB_TOKEN is not set` のエラーにする。GitHub は呼ばない。
- token は `Authorization` ヘッダにだけ入れる。ファイルに書かず、ログ、エラー、出力のどれにも入れない。

## 順序と終了コード

1. 設定を読む
2. 要件文書を読み、書式検査 (`check.Find`) を適用する
3. 書式のエラーがあれば、GitHub を呼ばずに終わる (`CTR-TRC-009`)
4. リポジトリと token を確かめる。`requirement_label` が空なら `invalid requirement_label: want a non-empty string`、`GITHUB_REPOSITORY` が形に合わなければ `invalid GITHUB_REPOSITORY "<value>": want <owner>/<name>` のエラーにする
5. GitHub を読み、対応づけて、JSON を標準出力に書く

| コード | 場合 | 標準出力 | 標準エラー出力 |
|---|---|---|---|
| 0 | JSON を書いた | JSON | なし |
| 1 | 書式のエラーがある | なし | 書式検査と同じエラーの行、続けて `cumin-trace: <N> error(s)` |
| 2 | 引数の誤り、設定や要件文書を読めない、リポジトリが決まらない、token がない、GitHub を読めない | なし | `cumin-trace: <message>` |

- 書式のエラーの行は、`check` と同じ `<file>:<line>: <ID>: <message>` である ([エラーと出力](../check/errors.md))。`check` と違って標準エラー出力に書く。標準出力をファイルにリダイレクトしても、JSON でないものがファイルに入らないためである。
- GitHub のエラーは、次のどれかとして出す。再試行しない。
  - `GitHub: HTTP <status>`
  - GraphQL の `errors` の最初の `message` を `GitHub: <message>`
  - 接続の失敗を `GitHub: cannot connect (<kind>)`。`<kind>` は `timeout`、`DNS lookup failed`、`TLS failed`、`connection failed` のどれかで、URL も token も入れない
  - `hasNextPage` が `true` なのに、`endCursor` が空か前と同じなら `GitHub: invalid page cursor`。同じページを読み続けないためである
- JSON は、すべてを読み終えてから1度に書く。途中で失敗したときは、何も書かない。

## JSON

`encoding/json` で、2文字の字下げと、最後の改行を付けて書く。キーの順は、下の表の順で固定する。配列は、空でも `null` ではなく `[]` にする。

[要件の例](../../requirements/trace/index.md) と同じ2つの要件では、次の JSON を出す。要件ごとに、[構造化データの出力](../export/index.md) と同じキーと値を同じ順に持ち、その後ろに対応づけのキーを持つ (`CTR-TRC-008`)。

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
      "progress": null,
      "children": {
        "not_filed": 0,
        "in_progress": 0,
        "done": 3,
        "total": 3
      },
      "requirement_issues": [],
      "implementation_issues": {
        "closed": 0,
        "total": 0
      }
    },
    {
      "id": "CTR-TRC-004.1",
      "parent": "CTR-TRC-004",
      "title": "未起票",
      "text": "対応する要求Issueが1つもない要件は、未起票とする。対応する要求Issueがすべて not planned で閉じている要件も、未起票とする。",
      "verifications": [
        {
          "method": "test",
          "how": "要求Issueのない要件と、not planned で閉じた要求Issueだけを持つ要件が、未起票になる"
        }
      ],
      "status": "active",
      "replaced_by": null,
      "file": "docs/ja/requirements/trace/index.md",
      "line": 62,
      "progress": "done",
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
              "closed_by": {
                "repository": "cloveclovedev/cumin-trace",
                "number": 70
              }
            }
          ]
        }
      ],
      "implementation_issues": {
        "closed": 1,
        "total": 1
      }
    }
  ],
  "discrepancies": [
    {
      "kind": "open_issue_for_deprecated",
      "issue": 18,
      "id": "CTR-TRC-002"
    },
    {
      "kind": "unknown_id",
      "issue": 31,
      "id": "CTR-TRC-099"
    }
  ]
}
```

- 不整合の2件は、形を示すための仮の例である。`discrepancies` は要求Issueの番号の順に並ぶ。
- 1つの要求Issueが複数の要件を挙げているときは、その要件ごとに、同じ要求Issueを sub-issue まで含めて繰り返す。

最上位:

| キー | 型 | 意味 |
|---|---|---|
| `repository` | 文字列 | 読んだリポジトリ (`<owner>/<name>`) |
| `read_at` | 文字列 | GitHub への最初の呼び出しの直前の時刻。UTC、RFC 3339、秒まで (`CTR-TRC-008`) |
| `requirements` | 配列 | すべての要件。廃止した要件も入る |
| `discrepancies` | 配列 | 不整合 (`CTR-TRC-006`) |

`requirements` の1つ:

| キー | 型 | 意味 |
|---|---|---|
| `id` から `line` まで | | 構造化データの出力の要件と同じキー、同じ値、同じ順 ([JSON の全体の例](../export/index.md#json-の全体の例))。`export.Of` で作る。廃止した要件は `status` が `deprecated` になる |
| `progress` | 文字列か `null` | `not_filed`、`in_progress`、`done` (`CTR-TRC-004`)。子を持つ要件と、廃止した要件は `null` |
| `children` | オブジェクトか `null` | 子を持つ要件では、廃止していない子の進捗ごとの数と、その合計 (`total`) (`CTR-TRC-005`)。子を持たない要件と、廃止した要件は `null` |
| `requirement_issues` | 配列 | `## Requirements` の節の項目の先頭にこのIDを書いた要求Issue ([IDの見つけ方](rules.md#idの見つけ方))。番号の順 |
| `implementation_issues` | オブジェクト | `requirement_issues` のすべての sub-issue の、閉じた数 (`closed`) と全体の数 (`total`)。同じ sub-issue は1回だけ数える |

`requirement_issues` の1つ:

| キー | 型 | 意味 |
|---|---|---|
| `number` | 数 | 要求Issueの番号 |
| `state` | 文字列 | `open` か `closed` |
| `state_reason` | 文字列か `null` | 閉じた理由。`completed` か `not_planned` ([進捗と不整合](rules.md))。開いていれば `null` |
| `milestone` | 文字列か `null` | マイルストーンのタイトル |
| `sub_issues` | 配列 | 実装Issue。リポジトリ、番号の順 |

`sub_issues` の1つ:

| キー | 型 | 意味 |
|---|---|---|
| `repository` | 文字列 | 実装Issueのリポジトリ。sub-issue は、ほかのリポジトリのIssueでもよいため |
| `number` | 数 | 実装Issueの番号 |
| `state` | 文字列 | `open` か `closed` |
| `milestone` | 文字列か `null` | マイルストーンのタイトル |
| `closed_by` | オブジェクトか `null` | 実装Issueを閉じた Pull Request の `repository` と `number`。開いている、または Pull Request 以外 (コミット、手作業) で閉じたときは `null` |

`discrepancies` の1つ:

| キー | 型 | 意味 |
|---|---|---|
| `kind` | 文字列 | `unknown_id` (要件文書にないID) か `open_issue_for_deprecated` (廃止した要件に対応する、開いた要求Issue) |
| `issue` | 数 | 要求Issueの番号 |
| `id` | 文字列 | `## Requirements` の節の項目の先頭に書いてあったID ([IDの見つけ方](rules.md#idの見つけ方)) |

## 並べ方

同じ文書と同じ GitHub の状態からは、`read_at` のほかは同じ JSON になる。

- `requirements`: 設定の `areas` の順、その中では要件文書を読んだ順。`CTR-EXP-002` と同じ並べ方で、`export.Sort` で並べる。
- `requirement_issues`: 番号の順。
- `sub_issues`: リポジトリ、番号の順。
- `discrepancies`: 要求Issueの番号、ID、`kind` の順。
