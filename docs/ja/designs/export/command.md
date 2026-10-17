# コマンドと出力

[構造化データの出力の設計](index.md) の細部。

## 引数

```sh
cumin-trace export
```

引数を取らない。`export` のあとに引数があれば、`CTR-CMD-003` のとおり、使い方を標準エラー出力に出して、終了コード 2 で終わる。読む先は、現在のディレクトリである。`main.go` は、`check` と同じく `root` に `.` を渡す。

使い方の行は、`export` を足して次のようにする。

```text
cumin-trace: usage: cumin-trace check | cumin-trace export | cumin-trace init <prefix> [<area>] | cumin-trace trace | cumin-trace version
```

- 設定のキーは足さない。`prefix`、`areas`、`requirements_dir` を、書式検査と同じに読む。
- 環境変数を読まず、GitHub を読まない。同じ文書からは、どこで動かしても同じ出力になる (`CTR-EXP-002`)。

## 順序と終了コード

1. 設定を読む
2. 要件文書を読み、書式検査 (`check.Find`) を適用する
3. 書式のエラーがあれば、JSON を書かずに終わる (`CTR-EXP-003`)
4. 要件を並べ、JSON を標準出力に書く

| コード | 場合 | 標準出力 | 標準エラー出力 |
|---|---|---|---|
| 0 | JSON を書いた | JSON | なし |
| 1 | 書式のエラーがある | なし | 書式検査と同じエラーの行、続けて `cumin-trace: <N> error(s)` |
| 2 | 引数の誤り、設定や要件文書を読めない、JSON を書けない | なし | `cumin-trace: <message>` |

- 書式のエラーの行は、`check` と同じ `<file>:<line>: <ID>: <message>` で、`check.WriteErrors` で書く ([エラーと出力](../check/errors.md))。`trace` と同じく、標準エラー出力に書く。標準出力をファイルにリダイレクトしても、JSON でないものがファイルに入らないためである。
- 件数の行 (`cumin-trace: 1 error`、`cumin-trace: <N> errors`) は、`main.go` の `countExit` が書く。
- JSON は、すべてを読み終えてから1度に書く。途中で失敗したときは、何も書かない。

## JSON

`encoding/json` の `Encoder` で、`SetIndent("", "  ")` と `SetEscapeHTML(false)` を付けて書く。`Encode` が最後の改行を付ける。

- `SetEscapeHTML(false)` は、本文の `<`、`>`、`&` を `<` などに変えず、そのまま書くためである。JSON として同じ値で、人が読みやすい。
- キーの順は、下の表の順で固定する。Go の構造体のフィールドの順で決まる。
- 配列は、空でも `null` ではなく `[]` にする。
- 出力した日時のように、実行のたびに変わる値は持たない (`CTR-EXP-002`)。

```json
{
  "requirements": [
    {
      "id": "PC-TASK-003",
      "parent": null,
      "title": "タスクの期限",
      "text": "タスクは期限を持てる。",
      "verifications": [],
      "status": "active",
      "replaced_by": null,
      "file": "docs/ja/requirements/task/index.md",
      "line": 30
    },
    {
      "id": "PC-TASK-003.1",
      "parent": "PC-TASK-003",
      "title": "期限の表示",
      "text": "期限を過ぎたタスクを、赤で表示する。",
      "verifications": [
        {"method": "test", "how": "期限を過ぎたタスクが赤になる"}
      ],
      "status": "active",
      "replaced_by": null,
      "file": "docs/ja/requirements/task/index.md",
      "line": 42
    }
  ]
}
```

最上位:

| キー | 型 | 意味 |
|---|---|---|
| `requirements` | 配列 | すべての要件。廃止した要件も入る (`CTR-EXP-001`) |

最上位をオブジェクトにするのは、`trace` の JSON と同じ形にし、あとでキーを足せるようにするためである。

`requirements` の1つ ([export-json.puml](../../requirements/export/export-json.puml) のキー):

| キー | 型 | モデルのフィールド | 意味 |
|---|---|---|---|
| `id` | 文字列 | `ID` | 要件のID |
| `parent` | 文字列か `null` | `ParentID` | 親のID。子要件でなければ `null` |
| `title` | 文字列 | `Title` | タイトル ([タイトル](reading.md#タイトル))。空でもよい |
| `text` | 文字列 | `Text` | 本文の Markdown ([本文](reading.md#本文))。空でもよい |
| `verifications` | 配列 | `Verifications` | 検証の一覧。文書に書いた順。子を持つ要件では空のことがある |
| `status` | 文字列 | `Deprecated` | `active` (廃止していない) か `deprecated` (廃止した) |
| `replaced_by` | 文字列か `null` | `ReplacedBy` | 置き換え先のID。廃止していない、または置き換え先がなければ `null` |
| `file` | 文字列 | `File` | リポジトリの最上位からの相対パス。区切りは `/` |
| `line` | 数 | `Line` | 見出しの行番号。ファイルの先頭が1 |

`verifications` の1つ:

| キー | 型 | モデルのフィールド | 意味 |
|---|---|---|---|
| `method` | 文字列 | `Verification.Method` | `test`、`demo`、`inspection`、`analysis` のどれか。ほかの値は書式のエラーなので、出力には来ない |
| `how` | 文字列 | `Verification.Description` | 確かめ方 |

- 書式検査だけが使う値 (`Level`、`Above`、`Statuses`、`ReplacedByLine`、`Verification.Line`) は出力しない。
- 同じIDの要件は、書式のエラー (`CTR-CHK-001` の 2) なので、出力には来ない。

## 並べ方

`requirements` は、設定の `areas` に書いたカテゴリの順、その中では要件文書を読んだ順に並べる (`CTR-EXP-002`)。

1. 要件のIDのカテゴリ (`requirement.Area`、接頭辞の次の部分) が、`areas` の何番目かを求める。`areas` にないカテゴリは書式のエラー (`CTR-CHK-001` の 9) なので、出力には来ない。
2. その番号で、安定な並べ替え (`sort.SliceStable`) をする。
3. 同じカテゴリの中は、読み取りの順のままである。読み取りは、ファイルのパスのバイト順、その中では行の順である ([要件文書の読み方](../check/reading.md))。

子要件は親と同じカテゴリなので、親の後ろ、次の要件の前に来る。

## パッケージとファイル

| ファイル | 役割 |
|---|---|
| `cmd/cumin-trace/main.go` | `export` を選び、結果を終了コードにする。使い方の行に `export` を足す |
| `internal/export/service.go` | 設定と要件文書を読み、書式検査を適用し、JSON を書く |
| `internal/export/domain.go` | JSON の構造 (`Output`、`RequirementOutput`、`VerificationOutput`) と、モデルからの変換と並べ方 (`Build`)。純粋な関数で、I/O を持たない |
| `internal/requirement/domain.go` | `Requirement` に `Title` と `Text` を足す |
| `internal/requirement/markdown.go` | タイトルと本文を読む ([タイトルと本文の読み方](reading.md)) |

入口:

```go
// internal/export
func Run(root string, stdout, stderr io.Writer) (int, error)
func Build(requirements []requirement.Requirement, areas []string) Output
```

- `Run` の戻り値の数は書式のエラーの件数で、`error` は出力ができないことを表す。`main.go` は、`trace` と同じく、0 件なら 0、1 件以上なら 1、`error` なら 2 で終わる。
- `Build` は、カテゴリを `requirement.Area` でIDから取り出す。
- `internal/export/` は、`internal/check/`、`internal/requirement/`、`internal/core/` を使う。`net/http`、`os/exec`、`internal/platform/` を import せず、環境変数を読まない。
- 新しいライブラリは使わない。interface は作らない。テストは、一時ディレクトリに書いた文書と設定を、本物の読み取りで読む。
