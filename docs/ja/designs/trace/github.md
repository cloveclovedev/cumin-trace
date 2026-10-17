# GitHub の読み方

[要件と要求Issueの対応づけの設計](index.md) の細部。

## 呼び出し

GitHub は GraphQL API だけで読む。どの呼び出しも、`GITHUB_GRAPHQL_URL` (既定 `https://api.github.com/graphql`) への `POST` で、本文は `{"query": ..., "variables": ...}` の JSON である。ヘッダは `Authorization: bearer <token>`、`Content-Type: application/json`、`User-Agent: cumin-trace` の3つである。HTTP の client は `net/http` で、時間切れは 60 秒とする。

問い合わせの文は、`internal/platform/github.go` の定数の3つだけである。どれも `query` で、`mutation` を持たない (`CTR-TRC-007`)。値は変数で渡し、文を組み立てない。

| # | 問い合わせ | 読むもの | 1回の範囲 |
|---|---|---|---|
| 1 | `requirementIssues` | ラベルの付いた要求Issueと、その sub-issue | 要求Issue 100件。それぞれの sub-issue 100件 |
| 2 | `moreSubIssues` | 101件目からの sub-issue | 要求Issue 1件の sub-issue 100件 |
| 3 | `closers` | 閉じた実装Issueを閉じた Pull Request | 実装Issue 100件 |

### 1. `requirementIssues`

```graphql
query requirementIssues($owner: String!, $name: String!, $label: String!, $after: String) {
  repository(owner: $owner, name: $name) {
    issues(labels: [$label], states: [OPEN, CLOSED], first: 100, after: $after,
           orderBy: {field: CREATED_AT, direction: ASC}) {
      pageInfo { hasNextPage endCursor }
      nodes {
        id number state stateReason body
        milestone { title }
        subIssues(first: 100) {
          pageInfo { hasNextPage endCursor }
          nodes {
            id number state
            repository { nameWithOwner }
            milestone { title }
          }
        }
      }
    }
  }
}
```

- `labels` で、ラベルの付いたIssueだけを読む (`CTR-TRC-001`)。`issues` は Pull Request を含まない。
- `states: [OPEN, CLOSED]` で、開いたIssueも閉じたIssueも読む。
- `orderBy` を作成日時の昇順に固定し、読んでいる間にIssueが増えても、ページの境目がずれないようにする。
- `hasNextPage` が `false` になるまで、`endCursor` を `after` に渡して繰り返す。

### 2. `moreSubIssues`

1 で、ある要求Issueの `subIssues` の `hasNextPage` が `true` のとき、その要求Issueの `id` で読む。

```graphql
query moreSubIssues($id: ID!, $after: String) {
  node(id: $id) {
    ... on Issue {
      subIssues(first: 100, after: $after) {
        pageInfo { hasNextPage endCursor }
        nodes { id number state repository { nameWithOwner } milestone { title } }
      }
    }
  }
}
```

### 3. `closers`

閉じた実装Issueの `id` を100件ずつ渡す。同じ実装Issueは1回だけ渡す。閉じた実装Issueがなければ、呼ばない。

```graphql
query closers($ids: [ID!]!) {
  nodes(ids: $ids) {
    ... on Issue {
      id
      timelineItems(itemTypes: [CLOSED_EVENT], last: 1) {
        nodes {
          ... on ClosedEvent {
            closer { ... on PullRequest { number repository { nameWithOwner } } }
          }
        }
      }
    }
  }
}
```

- 最後の `ClosedEvent` の `closer` が Pull Request なら、その Pull Request が実装Issueを閉じた (`CTR-TRC-003`)。コミットや手作業で閉じたときは、`closer` が Pull Request ではないので、`closed_by` は `null` になる。
- 採らなかった案: `closedByPullRequestsReferences`。公式の説明は「List of open pull requests referenced from this issue」で、閉じたものではなく、閉じる予定の参照である。

## GitHub の値の変換

`internal/platform/github.go` は、GraphQL の応答を、このファイルで定義した Go の型に、値を変えずに移す。GraphQL の型は、このファイルの外に出さない。`internal/trace/convert.go` が、その値を `internal/trace/domain.go` のモデルに次のように変える。

| GitHub | `internal/trace` のモデル |
|---|---|
| `state` が `OPEN` | 開いている |
| `state` が `CLOSED`、`stateReason` が `NOT_PLANNED` か `DUPLICATE` | not planned で閉じた |
| `state` が `CLOSED`、`stateReason` がそれ以外 (`COMPLETED`、`null`) | completed で閉じた |
| `milestone` が `null` | マイルストーンなし |
| `body` | そのまま |

- `DUPLICATE` は、ほかのIssueに任せて閉じたもので、受け入れて閉じたものではない。そこで not planned と同じに扱う。
- `stateReason` が `null` の閉じたIssueは、閉じた理由がなかったころのIssueである。GitHub の既定の閉じ方と同じく、completed として扱う。

## 呼び出しの費用

公式の計算方法 (下の「確かめたこと」) で見積もった値である。測った値ではない。

| 問い合わせ | 接続の数 | 点数 |
|---|---|---|
| 1 (1ページ) | 1 + 100 = 101 | 1 |
| 2 (1ページ) | 1 | 1 |
| 3 (100件) | 100 | 1 |

- 要求Issueが N 件、閉じた実装Issueが M 件なら、およそ ⌈N/100⌉ + ⌈M/100⌉ 点である。GitHub Actions の `GITHUB_TOKEN` の上限 (リポジトリごとに1時間 1,000 点) に十分収まる。
- 1 の1回のノードの数は、最大 100 + 100 × 100 = 10,100 で、上限の 500,000 に収まる。
- 採らなかった案: 1 の sub-issue の中で `timelineItems` も読む。接続の数が 1 + 100 + 100 × 100 で、1ページが 101 点になる。

## 確かめたこと

2026-10-16 に、次の公式ページを読んだ。

- <https://docs.github.com/en/graphql/guides/forming-calls-with-graphql>: エンドポイント `https://api.github.com/graphql`、`POST`、`Authorization: bearer TOKEN`。query は読み取り、mutation は変更である。
- <https://docs.github.com/en/graphql/reference/repos>: `Repository.issues` の引数 `labels`、`states`、`orderBy`、`first`、`after`。`Repository.nameWithOwner`。
- <https://docs.github.com/en/graphql/reference/issues>: `Issue` の `number`、`state`、`stateReason`、`body`、`milestone`、`subIssues`、`timelineItems` (`itemTypes`)、`closedByPullRequestsReferences`。`ClosedEvent.closer` (union `Closer`: Commit、ProjectV2、PullRequest)。`IssueState` (OPEN、CLOSED)。`IssueStateReason` (COMPLETED、DUPLICATE、NOT_PLANNED、REOPENED)。`IssueTimelineItemsItemType.CLOSED_EVENT`。
- <https://docs.github.com/en/graphql/reference/meta>: `Query.nodes(ids: [ID!]!)`「Lookup nodes by a list of IDs」。`ids` の数の上限は書いていない。100件ずつ渡すのは、ほかの接続の `first` の上限に合わせた、この設計の決めごとである。
- <https://docs.github.com/en/graphql/guides/using-pagination-in-the-graphql-api>: `first` は 1 から 100。`pageInfo` の `hasNextPage` と `endCursor` を `after` に渡す。
- <https://docs.github.com/en/graphql/overview/rate-limits-and-query-limits-for-the-graphql-api>: 1回のノードは 500,000 まで。点数は、接続ごとの要求の数の合計を 100 で割って丸め、最小 1。上限は、ユーザで1時間 5,000 点、GitHub Actions の `GITHUB_TOKEN` でリポジトリごとに1時間 1,000 点。
- <https://docs.github.com/en/actions/reference/workflows-and-actions/variables>: `GITHUB_REPOSITORY` (`octocat/Hello-World` の形)、`GITHUB_GRAPHQL_URL`。

未確認:

- GitHub Actions の `GITHUB_TOKEN` で、sub-issue とマイルストーンを読めるか ([GitHub との対応づけ](../../requirements/trace/index.md) の未決事項)。実装のあとに、実際に呼んで確かめる。
- ほかのリポジトリの sub-issue を、そのリポジトリの権限のない token で読んだときの結果。
