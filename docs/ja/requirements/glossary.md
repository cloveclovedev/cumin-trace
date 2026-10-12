# 用語集

cumin-trace の文書で使う語の意味を決める。「使わない言い方」にある言い方は、文書では使わない。「英語」は、要求Issueなどを英語で書くときの訳である。

## 要件文書

| 語 | 意味 | 使わない言い方 | 英語 |
|---|---|---|---|
| 要件文書 | 製品の要件を書いた Markdown の文書。製品リポジトリに置く | 要求文書 | requirement document |
| 要件 | IDを持つ、真偽を確かめられる1つのルール | | requirement |
| 製品リポジトリ | 要件文書を置くリポジトリ | 対象リポジトリ | product repository |
| 領域 | 要件を分ける単位。1つの領域が、1つのディレクトリと1つのページになる | カテゴリ、エリア | area |
| 親の要件 | 子の要件を持つ要件 | | parent requirement |
| 子の要件 | 親の要件の見出しの下に書く要件 | 下位要件、サブ要件 | child requirement |
| フィールド | 要件の範囲の中に、箇条書きで書く項目。`検証` と `状態` がある | 属性 | field |
| 検証 | 要件が満たされたことの確かめ方を書くフィールド | 確認方法 | verification |
| 廃止 | 要件を、文書に残したまま無効にすること | 非推奨、無効 | deprecated |

## IDと番号

| 語 | 意味 | 使わない言い方 | 英語 |
|---|---|---|---|
| ID | 要件を指す名前の全体。例: `CTR-FMT-007`、`CTR-TRC-004.1` | 番号、要件番号 | ID |
| 接頭辞 | IDの先頭にある、製品を表す英数字。例: `CTR` | プレフィックス | prefix |
| 連番 | IDのうち、領域の中で付ける3桁の数字。例: `007` | 番号、通し番号 | sequence number |
| 子の番号 | 子の要件のIDの、`.` のあとの数字。例: `.1` | 枝番 | child number |
| 番号 | 連番と子の番号をまとめて指す語。ID全体を指すときには使わない | | number |

## GitHub との対応づけ

| 語 | 意味 | 使わない言い方 | 英語 |
|---|---|---|---|
| 要求Issue | 要件を実現するために Maintainer が書くIssue。本文にIDを書く | 要件Issue、親Issue | requirement issue |
| 実装Issue | 要求Issueの sub-issue | 子Issue | implementation issue |
| 対応づけ | 要件と、要求Issue、実装Issueを結び付けること | トレース、紐付け | trace |
| 進み具合 | 要件がどこまで進んだか。未起票、進行中、完了のいずれか | 進捗、ステータス | progress |
| 未起票 | 対応する要求Issueが、まだない | 未着手 | not filed |
| 進行中 | 開いた要求Issueがある | 対応中 | in progress |
| 完了 | 要求Issueが受け入れられて閉じた | 済み | done |
| 食い違い | 要件文書と要求Issueが合っていないこと | 不整合 | discrepancy |

## ツールと人

| 語 | 意味 | 使わない言い方 | 英語 |
|---|---|---|---|
| 検査 | 要件文書が書式に沿っているかを確かめること。`cumin-trace check` | チェック、lint | check |
| Maintainer | 要件文書と要求Issueを書く人 | メンテナ | Maintainer |
| Agent | Maintainer と一緒に要件文書を書く AI | エージェント | Agent |
