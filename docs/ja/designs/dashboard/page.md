# ページ

[ダッシュボードの設計](index.md) の細部。`site/trace.json` を読み、ダッシュボードのページを作るまでを決める。ページの並びは、[要件のワイヤーフレーム](../../requirements/dashboard/index.md) のとおり、上から、読んだ時刻、要件の進捗 (タイルの地図)、不整合、要件の一覧である。

## ファイル

`templates/site/` に足し、`init` が `site/` の同じパスに書く。[サイトのファイルと設定](../site/files.md) の表の後ろに足す。

| 書く先 | 中身 | 要件 |
|---|---|---|
| `site/src/dashboard.mjs` | `trace.json` を読む `readTrace()`、要件の行、カテゴリの名前 (`areaTitle()`)、不整合、リンクを作る関数 | `CTR-DASH-001`、`CTR-DASH-005`、`CTR-DASH-006`、`CTR-DASH-009` |
| `site/src/pages/dashboard.astro` | `StarlightPage` で包んだページ | `CTR-DASH-001`、`CTR-DASH-005` から `CTR-DASH-007`、`CTR-DASH-009` |

ほかに、次のファイルを変える。

- `site/astro.config.mjs`: `sidebar()` の項目の前に、最初の項目として、ダッシュボードへのリンクを1つ置く。
- `site/package.json`: スクリプト `dev-trace` を足す (下の「コマンド」)。
- `site/.gitignore` (`templates/site/gitignore`): `trace.json` を足す。
- `site/src/requirements.css`: タイルの地図、不整合、要件の一覧の見た目を足す (下の「要件の進捗」と「不整合」)。
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
- `tree()`、`discrepancies()`、`japanTime()` のどれかが例外を出す。`dashboard.astro` がこれらの関数をまとめて `try` で呼ぶ。`children`、`implementation_issues`、`sub_issues` がない JSON でも、ビルドは失敗しない

「読めない」のとき、ページは見出しのあとに、次の1文だけを表示する (`CTR-DASH-006`)。要件の進捗、不整合、要件の一覧は表示しない。

> GitHub を読めなかったため、進捗を表示できません。

読めたときは、ページの先頭に `GitHub を読んだ時刻: <時刻>` を表示する。`<時刻>` は、`trace` が書いた UTC の `read_at` を、ビルドのときに `dashboard.mjs` の `japanTime()` が日本時間 (`Asia/Tokyo`) に直したものである。形は `<年>-<月>-<日> <時>:<分>:<秒>` で、`T` と `Z` を持たない。月、日、時は先頭に 0 を付けず、分と秒は2桁にする。`2026-10-17T09:10:55Z` は `2026-10-17 18:10:55`、`2026-01-04T20:05:03Z` は `2026-1-5 5:05:03` になる。`<time>` の `datetime` 属性は、`read_at` のままである。

- 直し方: `Intl.DateTimeFormat` に `timeZone: "Asia/Tokyo"` と `hourCycle: "h23"` を渡し、`formatToParts()` の部分から組み立てる。時間帯を名前で決めるので、ビルドする手元の時間帯で表示が変わらない。2026-10-20 に MDN の [Intl.DateTimeFormat() constructor](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat/DateTimeFormat) (`timeZone`、`hourCycle`、`numeric`) と [formatToParts()](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat/formatToParts) (部分の `type`) を読んだ。
- `hourCycle: "h23"` の `hour: "numeric"` でも、Node.js の出力の時は2桁になった (2026-10-20 に `check.sh` で確かめた)。そのため、各部分を数として読み直してから書く。
- `read_at` が時刻として読めなければ、`japanTime()` は例外を出し、ページは「読めない」ときと同じ表示になる。

`trace.json` は、`docs_dir` の文書と同じ時点のものとは限らない。文書にない ID は、下の「リンク」の決まりで、リンクにしない。

## 要件の進捗

`tree(trace)` が、行の配列を返す。2つの見せ方で同じ行を並べる (`CTR-DASH-001`)。見出し「要件の進捗」の下は、製品の全体を1画面で見るタイルの地図である。見出し「要件の一覧」の下は、要件ごとに1行の一覧である。2つの間に、下の「不整合」を置く。表と `<details>` は使わない。ダッシュボードのページは目次を持たない (`StarlightPage` の `frontmatter` の `tableOfContents: false`)。

- 廃止した要件 (`status` が `deprecated`) と、その子孫は出さない。
- カテゴリ: `requirements` の順に、カテゴリが最初に現れたところで、1つの行を置く。`trace` は要件を設定の `areas` の順に並べるので、カテゴリもその順になる。要件のカテゴリは、ID の2つ目の部分 (`CTR-DASH-001` の `DASH`) である。設定の `areas` は読まない。古い `trace.json` のカテゴリも、行を失わない。
- 要件: カテゴリの中に、`requirements` の順 (`trace` の並べ方) で置く。子の要件は、親の下に1段深く置く。
- カテゴリの名前: `areaTitle()` が、カテゴリの最初の要件の `file` の文書のタイトルを、サイトと同じ `documentTitle()` で読む。その文書が `docs_dir` にないとき (古い `trace.json`) は、カテゴリの ID を名前にする。

行の値:

| 行 | 進捗 | 要求Issue | マイルストーン | 実装Issue |
|---|---|---|---|---|
| カテゴリ | `完了 <n> / <m>`。`m` はカテゴリの中の子を持たない要件の数、`n` はそのうち完了の数。進捗ごとの数 (`counts`) も持つ | なし | なし | なし |
| 子を持つ要件 | `完了 <children.done> / <children.total>`。`state` は自分の `progress` (`not_filed`、`in_progress`、`done`)。`progress` が `null` (古い cumin-trace の `trace.json`) か、3 つの進捗のどれでもない値なら、`state` は `parent` | 子を持たない要件と同じ | 子を持たない要件と同じ | 子を持たない要件と同じ |
| 子を持たない要件 | `未起票`、`進行中`、`完了` (`progress`)。`progress` が `not_filed` で `changed_after_done` が `true` なら `変更後の未起票`。`state` は `not_filed`、`in_progress`、`done` | not planned で閉じたものを除く要求Issueの番号 (`#<n>`) | その要求Issueの異なるマイルストーン。マイルストーンのない要求Issueがあれば `なし` を足す | `<closed> / <total>` (`implementation_issues`) と、各 sub-issue (開いているか閉じたか)。`total` が 0 なら出さない |

進捗の色は、完了が `--sl-color-green`、進行中が `--sl-color-orange`、未起票が `--sl-color-gray-5` の輪である。子を持つ要件も、自分の進捗の色を持つ。`state` が `parent` の要件 (`progress` が `null` の古い `trace.json` か、3 つの進捗のどれでもない値) だけが、色のない枠である。色だけに頼らず、進捗は読み上げ用の文字 (`sr-only` か `aria-label`) でも持つ。子を持つ要件の読み上げ用の文字は、子の数 (`完了 <done> / <total>`) だけで、進捗の文字を重ねない。進捗は点とタイルの色が示す。色はすべて Starlight の変数で、ダークとライトのどちらのテーマでも同じ決まりで決める (`requirements.css`)。

変更後の未起票 (`CTR-TRC-010`) は、ほかの未起票と見分けられるように示す。`tree()` は、`progress` が `not_filed` で `changed_after_done` が `true` の要件の行に `changed: true` を置く。ほかの行は `false` である。`changed_after_done` を持たない古い `trace.json` では、どの行も `false` で、ビルドは失敗しない。`changed` の行のタイル、一覧の行、その点は `data-changed="true"` を持ち、`--sl-color-red` の輪になる。輪の太さはほかの未起票の輪と同じで、色だけが違う。タイルの文字も同じ色である。読み上げ用の文字と進捗の文字は `変更後の未起票` である。

### タイルの地図

- カテゴリごとに1行。左に、カテゴリの名前、カテゴリの ID、`完了 <n> / <m>` を置く。
- 右に、要件ごとに1つのタイルを、一覧と同じ順 (子も含む) に並べる。タイルは、接頭辞とカテゴリを除いた ID (`001`、`004.1`) を示し、進捗の色を持つ。
- 実装Issueを持つ要件のタイルは、下に細い線を持ち、閉じた割合だけ塗る。
- タイルは要件へのリンク (`requirementURL()`) である。表にない ID のタイルはリンクでない。
- タイルの `title` と `aria-label` は、ID、タイトル、進捗 (子を持つ要件は子の数)、実装Issueの `<closed> / <total>` である。
- 凡例は、完了、進行中、未起票、変更後の未起票と、タイルの下の線の意味である。

### 要件の一覧

- カテゴリごとに、見出しの行を置く。カテゴリの名前、カテゴリの ID、完了、進行中、未起票の割合の棒、`完了 <n> / <m>` である。
- 凡例は、完了、進行中、未起票、変更後の未起票と、実装Issueの四角の意味である。
- 要件ごとに1行。進捗の点、ID (要件へのリンク)、タイトル、要求Issueのリンクとそのマイルストーン、実装Issueの四角と `<closed> / <total>` を並べる。
- タイトルは1行に収め、あふれた分を省略する。省略しないタイトルは `title` に置く。子を持つ要件は、タイトルのあとに子の数を置く。
- 子の要件は、深さの分だけ字下げする。
- 実装Issueは、sub-issue ごとに1つの小さな四角で示す。四角は sub-issue へのリンク (`issueURL()`) で、閉じたものは緑である。同じ実装Issueは1回だけ並べる。
- 幅が `40rem` 以下の画面では、要求Issueと実装Issueをタイトルの下の段に移し、地図のカテゴリの名前をタイルの上に置く。横にはみ出さない。
- 本文の幅は、ほかのページと同じ `60rem` のままである。

マイルストーンの並びは、タイトルの文字コードの順である。マイルストーンのない要求Issueがあれば、最後に `なし` を置く。

子を持つ要件も、自分の要求Issueを持ちうる。親のIDを書いた要求Issueは、親の `requirement_issues` に入る ([進捗と不整合](../trace/rules.md))。そのため、子を持つ要件の行も、要求Issue、マイルストーン、実装Issueを、子を持たない要件と同じ決まりで示す。要求Issueがなければ出さない。

## 不整合

`discrepancies(trace)` が、`discrepancies` を同じ順に返す (`CTR-DASH-009`)。タイルの地図のあと、見出し「要件の一覧」の前に、見出し「不整合」を置き、不整合ごとに1行を並べる。空なら、見出しも行も表示しない。

- 行は、要件の一覧の行 (`.dashboard-line`) と同じ列、高さ、文字の大きさ、間隔である。点の列は空ける。
- 行の中身は、ID、種類、要求Issue の順である。ID は要件へのリンク (`requirementURL()`) で、表にない ID はリンクでない `<code>` である。要求Issueは `issueURL()` へのリンクである。
- 種類は、下の表の文字である。表にない `kind` は、`kind` のまま表示する。
- 表と `<details>` は使わない。
- 幅が `40rem` 以下の画面では、要求Issueを種類の下の段に移す。横にはみ出さない。

| `kind` | 表示する種類 |
|---|---|
| `unknown_id` | 要件文書にないID |
| `open_issue_for_deprecated` | 廃止した要件の開いた要求Issue |

## リンク

`CTR-DASH-005`:

| 対象 | 行き先 |
|---|---|
| 要件の ID (タイル、一覧、不整合) | `requirementURL()`: `requirements.mjs` が参照のリンクのために作る ID からページの URL への表で、ID のページを引き、`#<ID>` を足す。表にない ID は、リンクにしない |
| 要求Issue (一覧、不整合) | `issueURL()`: `https://github.com/<repository>/issues/<number>`。`repository` は最上位のもの |
| 実装Issue (一覧の四角) | `issueURL()`: `https://github.com/<repository>/issues/<number>`。`repository` は sub-issue のもの |

- リンクには `trace.json` の `file` を使わない。`trace.json` が文書より古くても、移した要件は今のページに行く。不整合の行は `file` を持たないが、同じ表で引ける。`file` を読むのは、カテゴリの名前 (`areaTitle()`) だけである。
- 実装Issueは、一覧の行で数と四角で示し、四角をリンクにする。`<details>` を開かずに、どの実装Issueが閉じたかが見える。

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
- step: checkout (`fetch-depth: 0` ですべての履歴を取る。`CTR-TRC-010` の変更後の未起票は git の履歴で決める)、`cumin-trace-check.yml` と同じ cumin-trace のダウンロード、`cumin-trace trace > site/trace.json` (`GITHUB_TOKEN: ${{ github.token }}`)、`actions/setup-node` (`node-version-file: site/.node-version`)、`npm --prefix site ci`、`npm --prefix site run build`。
- trace が失敗すると、step が失敗し、続く step は動かない (公式文書で確かめた。[リリースと workflow](../templates/release.md) の「プロダクトのサイトの workflow」)。ビルドしないので、公開している場所のサイトは前のまま残る。
- `site/dist/` をどこに置くかは書かない。プロダクトが、最後に step を足す。
- このリポジトリの `.github/workflows/site.yml` も、公開する実行 (`main` への `push` と毎日 19:42 UTC (日本時間 4:42) の `schedule`) で、`npm ci` のあとに `go run ./cmd/cumin-trace trace > site/trace.json` を動かす。権限は例と同じ3つである。trace が失敗すると、ビルドも公開もしない。Pull Request のビルドは trace を動かさない。

## 確かめ方

### 今の行の変更

ダッシュボードのページがあると、今の `check-site` の2つの行が失敗する。次のとおりに変える。

| 今の行 | 失敗する理由 | 変更 |
|---|---|---|
| `CTR-SITE-012` 各ページが `pageURL()` の URL にある | `site/dist/dashboard/index.html` は文書のページではない | 文書のページのほかに、`dashboard/index.html` の1つだけを受け付ける。ほかの `index.html` は今のとおり失敗する |
| `CTR-SITE-011` サイドバーの `<a>` が `sidebar()` の項目と同じ順 | サイドバーの最初に、ダッシュボードへのリンクが1つ増える | サイドバーの最初のリンクが `<base>dashboard/` であることを確かめ、そのあとのリンクの並びを、今のとおり `sidebar()` の項目と比べる。最上位の項目の確かめ方も、ダッシュボードへのリンクを飛ばす。`docs_dir` の `index.md` のページはサイドバーに出ない。`docs_dir` の `index.md` のリンクの順との比べ方は、ダッシュボードへのリンクのあとから始まる |

`CTR-SITE-002` から `CTR-SITE-010` の行は、`docs_dir` の文書から作ったページだけを調べるので、変えない。

### 足す行

`check-site` に、次の行を足す。HTML の形と、関数の出力との一致を確かめる。

| 確かめること | 要件 |
|---|---|
| `site/dist/dashboard/index.html` がある。各ページのサイドバーの最初のリンクの `href` が `<base>dashboard/` である | `CTR-DASH-001`、`CTR-DASH-005` |
| `trace.json` が「読めない」とき、ページの本文が GitHub を読めなかった1文だけで、表がない | `CTR-DASH-006` |
| 読めたとき、読んだ時刻の `<time>` の `datetime` が `read_at` と同じで、その文字が `japanTime()` の値と同じである | `CTR-DASH-006` |
| 地図と一覧のカテゴリが `tree()` のカテゴリと `areaTitle()` の名前と同じである。タイルと一覧の行が `tree()` の要件の行と同じ順と値 (変更後の未起票の `data-changed` を含む) である。廃止した要件がない。表と `<details>` がない | `CTR-DASH-001` |
| 要件のリンクの行き先のページと `id` が出力にある。要求Issueと実装Issueのリンクが `issueURL()` と同じである | `CTR-DASH-005` |
| 不整合の見出しが地図と一覧の間にある。各行の種類、要求Issueのリンク、ID とそのリンクが、`discrepancies()` と同じ順と値である。空なら不整合の見出しも行もない | `CTR-DASH-009` |
| `site/dist/` に `trace.json` がない。サーバの関数がない (今の `CTR-SITE-008` の行) | `CTR-DASH-007` |

`testdata/site-cases/trace.json` は、`testdata/site-cases/docs/` の要件の ID を使う fixture である。次を含む。

- 2つのマイルストーンの要求Issueを持つ要件
- not planned で閉じた要求Issueだけを持つ要件 (未起票)
- マイルストーンのない要求Issue
- 廃止した要件
- `unknown_id` と `open_issue_for_deprecated` の不整合
- ほかのリポジトリの sub-issue
- `changed_after_done` が `true` の要件 (変更後の未起票)。ほかの要件は `changed_after_done` を持たず、古い `trace.json` の代わりになる

`check.sh` は、fixture を `site/trace.json` に写して `check-site` を動かす。`check-site` が関数と同じ値を期待するだけでは、関数の決まりの誤りを見つけられない。そのため、`check.sh` は、サイドバーの確かめ方と同じく、`tree()` と `discrepancies()` の出力を、決まりから手で書いた固定の値と比べる。地図と一覧で `data-changed="true"` を持つのが、変更後の未起票の要件だけであることも確かめる。そのあと、`discrepancies` を空にした `trace.json` で、もう一度動かし、不整合の見出しがないことを確かめる。最後に、JSON でない `trace.json` を書いて、もう一度動かし、読めないときの行を確かめる。このリポジトリの `site.yml` は、Pull Request では `trace.json` なしでビルドするので、ファイルがないときの行を確かめる。公開する実行では、trace の出力で、読めたときの行を確かめる。

`CTR-DASH-005` のクリックの行き先は、Maintainer が `demo` で見る。
