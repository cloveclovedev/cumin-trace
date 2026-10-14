# エラーと出力

[書式検査の設計](index.md) の細部。

## エラーの一覧

書式検査は、`CTR-CHK-001` の表にある10個だけを検出する。表にない書き方 (タイトルが空、IDの後ろの空白が2つ、など) は、エラーにしない。

| # | エラー (`CTR-CHK-001`) | 条件 | 行 | ID | メッセージ |
|---|---|---|---|---|---|
| 1 | IDの形式が正しくない | 要件の見出しのコードスパンの中身が、正しいIDではない | 見出し | コードスパンの中身 | `invalid ID format: want <prefix>-<area>-<3 digits>, or that with .<child number>` |
| 2 | 同じIDが2か所以上にある | 同じIDの要件の見出しが、2つ以上ある。そのすべてについて1行ずつ報告する | 見出し | そのID | `duplicate ID: also at <file>:<line>` (ほかの場所が複数あれば `, ` で並べる) |
| 3 | 子要件が、親の見出しの外にある | 子要件の上の見出しが、親のIDの要件の見出しではない。または、レベルが親よりちょうど1つ深くない | 見出し | 子要件のID | `child requirement is not under its parent: put it one level below the heading of <parent ID>` |
| 4 | 検証の書き始めが、4つの方法のどれでもない | `検証` の方法が、`test`、`demo`、`inspection`、`analysis` のどれでもない | `検証` の行 | 要件のID | ``verification must start with `test`, `demo`, `inspection`, or `analysis` in a code span`` |
| 5 | 子を持たない要件に、検証がない | 子を持たない要件に、`検証` の行が1つもない。方法が正しくない行も、1行と数える。廃止した要件も対象である | 見出し | 要件のID | `no verification: a requirement without children needs a 検証 line` |
| 6 | 状態に、廃止以外が書いてある | `状態` の値が、正しい2つの形のどちらでもない | `状態` の行 | 要件のID | `invalid status: want 廃止, or 廃止。<ID> に置き換え` |
| 7 | 存在しないIDを参照している | 参照、または置き換え先のIDを持つ要件が、文書にない | 参照の行。置き換え先は、それを取った `状態` の行 | 参照したID | `reference to an ID that does not exist` |
| 8 | 廃止した要件を、置き換え先の指定以外で参照している | 参照のIDを持つ要件が、廃止である。置き換え先は、廃止した要件を指してもよい | 参照の行 | 参照したID | `reference to a deprecated requirement` |
| 9 | 設定にないカテゴリのIDがある | 要件のIDのカテゴリが、`areas` にない。参照には適用しない (7 になる) | 見出し | 要件のID | `area <area> is not in .cumin-trace.toml` |
| 10 | 連番、または子の番号が空いている | 下の決まり | 見出し | 空きの次の要件のID | `number gap: <ID> is missing`、`number gap: <ID> to <ID> are missing`、`number gap: numbers start at 001` |

10 の決まり (`CTR-FMT-012`):

- カテゴリごとに、子要件ではない要件の連番を集める。`areas` にないカテゴリも、同じように調べる。
- 親のIDごとに、子の番号を集める。
- 集めた番号は、1 から最大の番号まで、すべてそろっていなければならない。
- 空きごとに1件を報告する。報告する場所は、空きのすぐ上の番号を持つ要件である。同じ番号の要件が複数あるときは、読み取りの順で最初のものである。メッセージには、空いているID (1つ、または最初と最後) を書く。
- 連番が `000` の要件は、`number gap: numbers start at 001` で報告する。
- 要件を並べた順は調べない。

## エラーごとのテスト文書

`CTR-CHK-001` の表の行ごとに、そのエラーを1つだけ含むテスト文書がある。テストは `internal/check/service_test.go` にある。

| # | エラー | テスト |
|---|---|---|
| 1 | IDの形式が正しくない | `TestReportsAnIDWithAWrongForm` |
| 2 | 同じIDが2か所以上にある | `TestReportsTheSameIDOnTwoPages` |
| 3 | 子要件が、親の見出しの外にある | `TestReportsAChildOutsideTheHeadingOfItsParent` |
| 4 | 検証の書き始めが、4つの方法のどれでもない | `TestReportsAVerificationThatStartsWithNoMethodOfTheFour` |
| 5 | 子を持たない要件に、検証がない | `TestReportsARequirementWithoutChildrenAndWithoutVerification` |
| 6 | 状態に、廃止以外が書いてある | `TestReportsAStateThatSaysAnotherThingThanDeprecated` |
| 7 | 存在しないIDを参照している | `TestReportsAReferenceToAnIDThatDoesNotExist` |
| 8 | 廃止した要件を、置き換え先の指定以外で参照している | `TestReportsAReferenceToADeprecatedRequirement` |
| 9 | 設定にないカテゴリのIDがある | `TestReportsAnAreaThatIsNotInTheSettings` |
| 10 | 連番、または子の番号が空いている | `TestReportsAGapInTheNumbersOfAnArea` |

## 出力と終了コード

エラーは、1件を1行で、標準出力に書く (`CTR-CHK-002`)。

```text
<file>:<line>: <ID>: <message>
```

```text
docs/ja/requirements/check/index.md:15: CTR-CHK-001: duplicate ID: also at docs/ja/requirements/format/index.md:82
docs/ja/requirements/format/index.md:120: CTR-FMT-099: reference to an ID that does not exist
```

- `<file>` は、リポジトリの最上位からの相対パスで、区切りは `/` である。
- `<ID>` の制御文字は、Go のエスケープ (たとえば `\x1b`) にして書く。出力が1行のままになり、端末の制御コードを含まないようにするためである。
- エラーは、ファイル、行番号、上の表の #、ID、メッセージの順に並べる。ID とメッセージは、バイト順で比べる。
- 同じ行にある同じIDへの参照は、何回書いてあっても1つの参照として数える。参照の一覧は、ファイル、行番号、IDの組ごとに1件である。
- この2つにより、5つのキーがすべて同じエラーは2件にならず、並び順は1通りに決まる。同じ入力からは、常に同じ出力になる。
- 標準出力には、エラーの行だけを書く。エラーがなければ、何も書かない。
- エラーがあるときは、最後に、標準エラー出力へ `cumin-trace: <n> errors` を書く (1件のときは `1 error`)。
- 書式検査を実行できないときは、標準エラー出力へ `cumin-trace: <message>` を書く。構文エラーのように場所が分かるときは、`cumin-trace: .cumin-trace.toml:<line>: <message>` と書く。

| 終了コード | 意味 (`CTR-CHK-003`) |
|---|---|
| 0 | エラーがない |
| 1 | 書式のエラーが1件以上ある |
| 2 | 書式検査を実行できない。設定ファイルがない、設定が正しくない、要件文書のディレクトリがない、ファイルを読めない、コマンドの使い方が正しくない |

- `cumin-trace check` は、リポジトリの最上位のディレクトリで実行する。引数とフラグは取らない。余分な引数があれば、終了コード 2 にする。
- 要件文書のディレクトリはあるが、要件が1つもないときは、終了コード 0 にする。
- 書式検査はログを出さない。
