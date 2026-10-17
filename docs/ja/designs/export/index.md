# 構造化データの出力の設計

`cumin-trace export` が、要件文書のすべての要件をどう読み、どの形の JSON に出すかを決める。

## 要件

- [構造化データの出力](../../requirements/export/index.md)
  - `CTR-EXP-001` 出力する内容
  - `CTR-EXP-002` 同じ文書からは同じ出力
  - `CTR-EXP-003` 書式の誤りがあるとき
- [コマンド](../../requirements/command/index.md)
  - `CTR-CMD-001` コマンドの一覧
  - `CTR-CMD-003` 引数の誤り
- [書式検査](../../requirements/check/index.md)
  - `CTR-CHK-001` 検出するエラー (出力の前に適用する)

## 実装の状況

いまのコードは、この設計のすべてを持つ。

| 部分 | 状況 |
|---|---|
| コマンド `cumin-trace export` と終了コード | 実装した |
| 要件の JSON の構造 (`export.Of`) と並べ方 (`export.Build`) | 実装した |
| 書式のエラーでの停止 (`CTR-EXP-003`) | 実装した |

## 全体像

[ソースコード全体構成](../../architecture/overview.md) の `internal/export` にあたる。

![構造化データの出力のパッケージ](export-packages.svg)

図の元ファイル: [export-packages.puml](export-packages.puml)

## 決めたこと

### 出力は、独立したコマンド `cumin-trace export`

引数を取らない。`export` のあとに引数があれば、`CTR-CMD-003` のとおり、使い方を標準エラー出力に出して、終了コード 2 で終わる。読む先は現在のディレクトリで、結果の JSON は標準出力に書く。

| 終了コード | 意味 |
|---|---|
| 0 | JSON を書いた |
| 1 | 書式のエラーがある。JSON を書かず、`check` と同じエラーの行と件数を標準エラー出力に書く (`CTR-EXP-003`) |
| 2 | 出力ができない (設定がない、文書を読めないなど) |

- 理由: `trace` と同じ形にする ([要件と要求Issueの対応づけのコマンド](../trace/command.md))。token もネットワークも要らない。
- 書式検査は `check.Find` をそのまま使い、エラーの行は `check.WriteErrors` で書く。こうして `check` と同じエラーを出す。

### 要件の JSON の形は、`internal/export` の1か所で決める

`domain.go` の `Of` が、1つの要件を JSON のオブジェクトにする。キーの順は、[要件の例](../../requirements/export/index.md#全体像) と同じである。空の一覧は `[]`、ない値は `null` にする。

- 理由: 対応づけ (`trace`) も、あとで同じ関数で要件のオブジェクトを作る。形を1か所に置けば、2つの出力が食い違わない。
- 採らなかった案: `internal/requirement` の `Requirement` に JSON のタグを付ける。読み取りのモデルには、書式検査だけが使う値 (`Above`、`Statuses` など) もあり、出力の形と分けたい。

### 要件は、設定のカテゴリの順、その中では文書の順に並べる

`Build` は、要件を `.cumin-trace.toml` の `areas` の順に安定ソートする。同じカテゴリの中では、読んだ順 (ファイルの名前の順、ファイルの中は行の順) のままである (`CTR-EXP-002`)。出力した日時は含めない。

- 理由: ファイルの名前の順では、カテゴリの順が文書の目次と合わない。安定ソートなら、同じ文書から常に同じバイトが出る。

## JSON の全体の例

[要件の例](../../requirements/export/index.md#全体像) の2つの要件だけを持つ文書では、次の JSON を出す。

```json
{
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
      "line": 58
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
      "line": 62
    }
  ]
}
```

- インデントは空白2つで、最後に改行を1つ書く (`json.MarshalIndent`)。
- 廃止した要件は `"status": "deprecated"` になり、`replaced_by` に置き換え先のIDを持つ。置き換え先がなければ `null` である。

## パッケージとファイル

| ファイル | 役割 |
|---|---|
| `cmd/cumin-trace/main.go` | `export` を選び、結果を終了コードにする |
| `internal/export/service.go` | `Run(root string, stdout, stderr io.Writer) (int, error)`。設定と要件文書を読み、書式検査を適用し、JSON を書く。戻り値の数は書式のエラーの件数で、`error` は出力ができないことを表す |
| `internal/export/domain.go` | 要件の JSON の構造 (`Output`、`Requirement`、`Verification`)、`Of`、`Build`。純粋な関数で、I/O を持たない |

- `internal/export/` は、`internal/check/`、`internal/requirement/`、`internal/core/` を使う。
- 新しいライブラリは使わない。interface は作らない。
