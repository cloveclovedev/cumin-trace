# ページ

[ダッシュボードの設計](index.md) の細部。`site/trace.json` を読み、ダッシュボードのページを作るまでを決める。ページの並びは、[要件のワイヤーフレーム](../../requirements/dashboard/index.md) のとおり、上から、読んだ時刻と絞り込み、集計、次に起票する要件、不整合、要件の進捗である。

## ファイル

`templates/site/` に足し、`init` が `site/` の同じパスに書く。[サイトのファイルと設定](../site/files.md) の表の後ろに足す。

| 書く先 | 中身 | 要件 |
|---|---|---|
| `site/src/dashboard.mjs` | `trace.json` を読む `readTrace()`、木、集計、一覧、リンクを作る純粋な関数 | `CTR-DASH-001` から `CTR-DASH-006`、`CTR-DASH-009` |
| `site/src/pages/dashboard.astro` | `StarlightPage` で包んだページと、絞り込みのインラインのスクリプト | `CTR-DASH-001` から `CTR-DASH-007`、`CTR-DASH-009` |

ほかに、次のファイルを変える。

- `site/astro.config.mjs`: `sidebar()` の項目の前に、最初の項目として、ダッシュボードへのリンクを1つ置く。
- `site/package.json`: スクリプト `dev-trace` を足す (下の「コマンド」)。
- `site/.gitignore` (`templates/site/gitignore`): `trace.json` を足す。
- `site/scripts/check-site.mjs`: 下の「確かめ方」の行を足す。
- `templates/workflows/cumin-trace-site.yml` (新しい): 下の「workflow の例」。

## 入力

`dashboard.mjs` の `readTrace(root)` が、`<root>/site/trace.json` を `fs.readFileSync()` で読み、`JSON.parse()` する。`import` で読まないのは、ファイルがないときにビルドを止めないためである (推測)。

`dashboard.astro` は、`root` を `settings.mjs` から読まず、`process.cwd()` の親とする。ビルドはページを `site/dist/` の下にまとめるので、`settings.mjs` の `import.meta.url` から決めた `root` は `site/dist` になる (2026-10-20 にビルドで確かめた)。Astro は `site/` で動き、`site/` を自分の root とするので、その親がリポジトリの最上位である。設定は `readSettings(root)` に渡して読む。

次のどれかのとき、`readTrace()` は「読めない」を返す。

- ファイルがない、または読めない
- JSON として読めない
- `repository` と `read_at` が文字列でない、または `requirements` と `discrepancies` が配列でない
- `requirements` の要素に、`id`、`status`、`file` の文字列、`requirement_issues` の配列のどれかがない
- `tree()`、`summary()`、`fileNext()`、`discrepancies()` のどれかが例外を出す。`dashboard.astro` が4つの関数をまとめて `try` で呼ぶ。`children`、`implementation_issues`、`sub_issues` がない JSON でも、ビルドは失敗しない

「読めない」のとき、ページは見出しのあとに、次の1文だけを表示する (`CTR-DASH-006`)。絞り込み、集計、一覧、要件の進捗は表示しない。

> GitHub を読めなかったため、進捗を表示できません。

読めたときは、ページの先頭に `GitHub を読んだ時刻: <時刻>` を表示する。`<時刻>` は、`trace` が書いた UTC の `read_at` を、ビルドのときに `dashboard.mjs` の `japanTime()` が日本時間 (`Asia/Tokyo`) に直したものである。形は `<年>-<月>-<日> <時>:<分>:<秒>` で、`T` と `Z` を持たない。月、日、時は先頭に 0 を付けず、分と秒は2桁にする。`2026-10-17T09:10:55Z` は `2026-10-17 18:10:55`、`2026-01-04T20:05:03Z` は `2026-1-5 5:05:03` になる。`<time>` の `datetime` 属性は、`read_at` のままである。

- 直し方: `Intl.DateTimeFormat` に `timeZone: "Asia/Tokyo"` と `hourCycle: "h23"` を渡し、`formatToParts()` の部分から組み立てる。時間帯を名前で決めるので、ビルドする手元の時間帯で表示が変わらない。2026-10-20 に MDN の [Intl.DateTimeFormat() constructor](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat/DateTimeFormat) (`timeZone`、`hourCycle`、`numeric`) と [formatToParts()](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat/formatToParts) (部分の `type`) を読んだ。
- `hourCycle: "h23"` の `hour: "numeric"` でも、Node.js の出力の時は2桁になった (2026-10-20 に `check.sh` で確かめた)。そのため、各部分を数として読み直してから書く。
- `read_at` が時刻として読めなければ、`japanTime()` は例外を出し、ページは「読めない」ときと同じ表示になる。

`trace.json` は、`docs_dir` の文書と同じ時点のものとは限らない。文書にない ID は、下の「リンク」の決まりで、リンクにしない。

## 要件の進捗

`tree(trace, settings)` が、行の配列を返す。表示は、見出し「要件の進捗」の下の1つの表で、最初の列の字下げが深さを示す (`CTR-DASH-001`)。ダッシュボードのページは目次を持たない (`StarlightPage` の `frontmatter` の `tableOfContents: false`)。

- 廃止した要件 (`status` が `deprecated`) と、その子孫は出さない。
- カテゴリ: `requirements` の順に、カテゴリが最初に現れたところで、1つの行を置く。`trace` は要件を設定の `areas` の順に並べるので、カテゴリもその順になる。要件のカテゴリは、ID の2つ目の部分 (`CTR-DASH-001` の `DASH`) である。設定の `areas` は読まない。古い `trace.json` のカテゴリも、行を失わない。
- 要件: カテゴリの中に、`requirements` の順 (`trace` の並べ方) で置く。子の要件は、親の下に1段深く置く。

| 行 | 進捗 | 要求Issue | マイルストーン | 実装Issue |
|---|---|---|---|---|
| カテゴリ | `完了 <n> / <m>`。`m` はカテゴリの中の子を持たない要件の数、`n` はそのうち完了の数 | 空 | 空 | 空 |
| 子を持つ要件 | `完了 <children.done> / <children.total>` | 子を持たない要件と同じ | 子を持たない要件と同じ | 子を持たない要件と同じ |
| 子を持たない要件 | `未起票`、`進行中`、`完了` (`progress`) | not planned で閉じたものを除く要求Issueの番号 (`#<n>`) | その要求Issueの異なるマイルストーン。`、` で区切る。マイルストーンのない要求Issueがあれば `なし` を足す | `<closed> / <total>` (`implementation_issues`)。そのあとに閉じた `<details>` を置き、各 sub-issue を `<repository>#<number>` のリンクで並べる。`total` が 0 なら空 |

マイルストーンの並びは、下の「集計」と同じである。

子を持つ要件も、自分の要求Issueを持ちうる。親のIDを書いた要求Issueは、親の `requirement_issues` に入る ([進捗と不整合](../trace/rules.md))。そのため、子を持つ要件の行も、要求Issue、マイルストーン、実装Issueの列を、子を持たない要件と同じ決まりで埋める。要求Issueがなければ空である。

## 集計

`summary(trace)` が、集計の表を返す (`CTR-DASH-002`)。数えるのは、廃止していない、子を持たない要件だけである。子を持つ要件は進捗を持たないので、その要求Issueのマイルストーンは集計に使わない。

- 行: 子を持たない要件の要求Issue (not planned で閉じたものを除く) のマイルストーンごとに1行。マイルストーンのない要求Issueは `(マイルストーンなし)` の1行にまとめる。
- 1つの要件は、その要求Issueの異なるマイルストーンごとに1回数える。v0.1 と v0.2 の要求Issueを持つ要件は、両方の行に出る。同じマイルストーンの2つの要求Issueでは1回である。
- 列: 進行中の数と完了の数。要件の `progress` で数える。
- 並び: マイルストーンのタイトルの文字コードの順。`(マイルストーンなし)` は最後である。
- 最後の行: `未起票 (要求Issueなし): <数>`。`progress` が `not_filed` の要件の数である。

## 次に起票する要件

`fileNext(trace)` が、`progress` が `not_filed` の要件を、`requirements` の順に返す。列は ID とタイトルである (`CTR-DASH-003`)。空なら、`未起票の要件はありません。` の1文を表示する。

## 不整合

`discrepancies(trace)` が、`discrepancies` を同じ順に返す。列は、種類、要求Issue、ID である (`CTR-DASH-009`)。空なら、見出しも表も表示しない。

| `kind` | 表示する種類 |
|---|---|
| `unknown_id` | 要件文書にないID |
| `open_issue_for_deprecated` | 廃止した要件の開いた要求Issue |

## リンク

`CTR-DASH-005`:

| 対象 | 行き先 |
|---|---|
| 要件の ID (木、次に起票する要件、不整合) | `requirementURL()`: `requirements.mjs` が参照のリンクのために作る ID からページの URL への表で、ID のページを引き、`#<ID>` を足す。表にない ID は、リンクにしない |
| 要求Issue (木、不整合) | `issueURL()`: `https://github.com/<repository>/issues/<number>`。`repository` は最上位のもの |
| 実装Issue (木の `<details>`) | `issueURL()`: `https://github.com/<repository>/issues/<number>`。`repository` は sub-issue のもの |

- `trace.json` の `file` は使わない。`trace.json` が文書より古くても、移した要件は今のページに行く。不整合の行は `file` を持たないが、同じ表で引ける。
- 実装Issueは、ワイヤーフレームの列のとおり数で示し、リンクは閉じた `<details>` に置く。列を増やさずに `CTR-DASH-005` を満たすためである。同じ実装Issueは1回だけ並べる。

## 絞り込み

`dashboard.astro` の上部に、2つの `<select>` を置く (`CTR-DASH-004`)。

- マイルストーン: `すべて`、集計と同じ並びの各マイルストーン、`なし`
- 進捗: `すべて`、`未起票`、`進行中`、`完了`

木の各行は、`data-milestones` (マイルストーンのタイトルの JSON の配列。`なし` を含みうる) と `data-progress` を持つ。インラインのスクリプトは、選んだ値に合わない子を持たない要件の行に `hidden` を付ける。子を持つ要件の行は、残った子孫が1つもなく、自分の要求Issueのマイルストーンも選んだ値に合わなければ隠す。進捗は子を持たない要件だけが持つので、進捗を選んだときは、子を持つ要件の行は子孫だけで決める。カテゴリの行は、残った子孫が1つもなければ隠す。未起票の要件はマイルストーンを持たないので、`すべて` のときだけ残る。

- 絞り込むのは木だけである。集計と一覧は変えない。
- JavaScript が動かないときは、すべての行が見える。
- 推測: `StarlightPage` の中に置いた `<script is:inline>` は、そのまま HTML に出る。

## コマンド

`site/package.json` と `templates/site/package.json` に、次のスクリプトを足す。

| コマンド | すること |
|---|---|
| `npm --prefix site run dev-trace` | `cumin-trace trace` を動かし、成功したときだけ `trace.json` を置き、`astro dev` を始める |

スクリプトの中身 (`site/` で動く。npm がスクリプトを `sh` で動かし、`node_modules/.bin` を `PATH` に足すことは推測):

```sh
rm -f trace.json; (cd .. && cumin-trace trace > site/trace.json.tmp) && mv trace.json.tmp trace.json; rm -f trace.json.tmp; astro dev
```

- `trace` はリポジトリの最上位で動かす。設定と要件文書をそこから読むためである。
- 標準出力を一時ファイルに書き、成功したときだけ名前を変える。シェルのリダイレクトは `trace` の前にファイルを作るので、直接 `trace.json` に書くと、失敗しても空のファイルが残る。
- 前の `trace.json` も消す。失敗したときに古い進捗を表示しないためである。
- `cumin-trace` が `PATH` にない、token がない、のどちらでも `trace` が失敗し、`astro dev` は始まる。ダッシュボードは GitHub を読めなかったことだけを表示する。
- token は、`GITHUB_TOKEN=$(gh auth token) npm --prefix site run dev-trace` のように渡す ([対応づけのコマンド](../trace/command.md) の「環境変数」)。

## workflow の例

`templates/workflows/cumin-trace-site.yml` を、`init` が `.github/workflows/cumin-trace-site.yml` に書く (`CTR-DASH-008`)。`cumin-trace-check.yml` と同じく、バージョンを持つバイナリのときだけ書く ([テンプレートの生成のファイル](../templates/files.md))。

- きっかけ: `main` への `push` と `schedule`。
- 権限: `contents: read`、`issues: read`、`pull-requests: read`。
- step: checkout、`cumin-trace-check.yml` と同じ cumin-trace のダウンロード、`cumin-trace trace > site/trace.json` (`GITHUB_TOKEN: ${{ github.token }}`)、`actions/setup-node` (`node-version-file: site/.node-version`)、`npm --prefix site ci`、`npm --prefix site run build`。
- trace が失敗すると、step が失敗し、続く step は動かない (公式文書で確かめた。[リリースと workflow](../templates/release.md) の「プロダクトのサイトの workflow」)。ビルドしないので、公開している場所のサイトは前のまま残る。
- `site/dist/` をどこに置くかは書かない。プロダクトが、最後に step を足す。
- このリポジトリの `.github/workflows/` は変えない。

## 確かめ方

### 今の行の変更

ダッシュボードのページがあると、今の `check-site` の2つの行が失敗する。次のとおりに変える。

| 今の行 | 失敗する理由 | 変更 |
|---|---|---|
| `CTR-SITE-012` 各ページが `pageURL()` の URL にある | `site/dist/dashboard/index.html` は文書のページではない | 文書のページのほかに、`dashboard/index.html` の1つだけを受け付ける。ほかの `index.html` は今のとおり失敗する |
| `CTR-SITE-011` サイドバーの `<a>` が `sidebar()` の項目と同じ順 | サイドバーの最初に、ダッシュボードへのリンクが1つ増える | サイドバーの最初のリンクが `<base>dashboard/` であることを確かめ、そのあとのリンクの並びを、今のとおり `sidebar()` の項目と比べる。最上位の項目の確かめ方も、ダッシュボードへのリンクを飛ばす。入口の文書のページは最上位の2つ目の項目で、入口の文書のリンクの順との比べ方は、そのあとから始まる |

`CTR-SITE-002` から `CTR-SITE-010` の行は、`docs_dir` の文書から作ったページだけを調べるので、変えない。

### 足す行

`check-site` に、次の行を足す。HTML の形と、関数の出力との一致を確かめる。

| 確かめること | 要件 |
|---|---|
| `site/dist/dashboard/index.html` がある。各ページのサイドバーの最初のリンクの `href` が `<base>dashboard/` である | `CTR-DASH-001`、`CTR-DASH-005` |
| `trace.json` が「読めない」とき、ページの本文が GitHub を読めなかった1文だけで、表がない | `CTR-DASH-006` |
| 読めたとき、読んだ時刻の `<time>` の `datetime` が `read_at` と同じで、その文字が `japanTime()` の値と同じである | `CTR-DASH-006` |
| 木の行が `tree()` と同じ順と値である。廃止した要件の行がない | `CTR-DASH-001` |
| 集計の各行の数が、HTML の木の子を持たない要件の行から数えた数と同じである | `CTR-DASH-002` |
| 次に起票する要件の行が、木の `未起票` の行と同じ ID の並びである | `CTR-DASH-003` |
| 各行が `data-milestones` と `data-progress` を持ち、絞り込みのスクリプトがある | `CTR-DASH-004` |
| 要件のリンクの行き先のページと `id` が出力にある。要求Issueと実装Issueのリンクが `issueURL()` と同じである | `CTR-DASH-005` |
| 不整合の行が `discrepancies` と同じである。空なら不整合の見出しがない | `CTR-DASH-009` |
| `site/dist/` に `trace.json` がない。サーバの関数がない (今の `CTR-SITE-008` の行) | `CTR-DASH-007` |

`testdata/site-cases/trace.json` は、`testdata/site-cases/docs/` の要件の ID を使う fixture である。次を含む。

- 2つのマイルストーンの要求Issueを持つ要件
- not planned で閉じた要求Issueだけを持つ要件 (未起票)
- マイルストーンのない要求Issue
- 廃止した要件
- `unknown_id` と `open_issue_for_deprecated` の不整合
- ほかのリポジトリの sub-issue

`check.sh` は、fixture を `site/trace.json` に写して `check-site` を動かす。`check-site` が関数と同じ値を期待するだけでは、関数の決まりの誤りを見つけられない。そのため、`check.sh` は、サイドバーの確かめ方と同じく、`tree()`、`summary()`、`fileNext()` の出力を、決まりから手で書いた固定の値と比べる。そのあと、JSON でない `trace.json` を書いて、もう一度動かし、読めないときの行を確かめる。このリポジトリの `site.yml` は `trace.json` なしでビルドするので、ファイルがないときの行を確かめる。

`CTR-DASH-004` の絞り込みの動きと、`CTR-DASH-005` のクリックの行き先は、Maintainer が `demo` で見る。
