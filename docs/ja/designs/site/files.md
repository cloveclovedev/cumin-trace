# ファイルと設定

[サイトの設計](index.md) の細部。

## テンプレートのファイル

`init` は、[テンプレートの生成のファイルの一覧](../templates/files.md) の 14 (`tools/plantuml/Dockerfile`) のあと、15 (workflow) の前に、次の順に書く。埋め込みは `templates/site/` の下で、書く先は `site/` の下の同じパスである。ただし、3 と 4 は、埋め込みでは先頭の `.` のない名前 (`site/node-version`、`site/gitignore`) にする。Go の `embed` は、ディレクトリの指定から `.` で始まるファイルを除くので、埋め込むファイルの名前を `.` で始めない決まり ([ファイルと埋め込み](../templates/files.md)) に合わせる。書く先のパスは `internal/scaffold/domain.go` の表が持つ。`templates.go` の `//go:embed` の指示に `site` を足す。差し込む値はない。権限は `0644` である。

| # | 書く先 | 中身 | 要件 |
|---|---|---|---|
| 1 | `site/package.json` | 固定したバージョン、`engines`、スクリプト (下の表) | `CTR-SITE-001`、`CTR-SITE-005` |
| 2 | `site/package-lock.json` | 1 の依存とその依存の、固定したバージョン。`npm ci` が使う | `CTR-SITE-001`、`CTR-SITE-005` |
| 3 | `site/.node-version` | `24` の1行 | `CTR-SITE-001` |
| 4 | `site/.gitignore` | `node_modules/`、`dist/`、`.astro/` | `CTR-SITE-001` |
| 5 | `site/astro.config.mjs` | 設定を読み、`base`、`satteri()` とその `hastPlugins`、Starlight の `title`、`customCss`、`head` (図を開くスクリプト)、`markdown.processedDirs` を決める | `CTR-SITE-001`、`CTR-SITE-008` |
| 6 | `site/src/settings.mjs` | `.cumin-trace.toml` を読み、既定値を補う | `CTR-SITE-002` |
| 7 | `site/src/content.config.mjs` | `docs` のコレクションと、`title` を補うローダー | `CTR-SITE-002`、`CTR-SITE-009` |
| 8 | `site/src/requirements.mjs` | アンカー、枠、参照のリンク、文書のリンク、`.puml` の折りたたみ、図を開くスクリプトの文、文書の最初の見出し | `CTR-SITE-003`、`CTR-SITE-006`、`CTR-SITE-007`、`CTR-SITE-010` |
| 9 | `site/src/requirements.css` | 要件の枠と ID の見た目 | `CTR-SITE-006` |
| 10 | `site/scripts/check-site.mjs` | 出力の HTML を調べる | `CTR-SITE-001` から `CTR-SITE-010` |

- いまのテンプレートは 1 から 10 を持つ。5 は `base`、`satteri()` と 8 のプラグイン、Starlight の `title` (接頭辞、なければ `Requirements`)、`customCss` (9 だけ)、`head` (図を開くスクリプト) を決める。8 は ID のアンカーと参照のリンク (`CTR-SITE-003`)、要件の枠と ID のクラス (`CTR-SITE-006`)、タイトルになった見出しを本文から消すこと (`CTR-SITE-002`)、文書のリンクと `.puml` の折りたたみ (`CTR-SITE-010`)、図を開くスクリプトの文 (`CTR-SITE-007`) を作る。8 は、7 と 10 が使う、文書の最初の見出し (`firstHeading()`) も持つ。10 は、下の表のすべての行を確かめる。
- 埋め込むパスに、`.claude`、`CLAUDE.md`、`.github/workflows` は現れない。
- `site/package-lock.json` もテンプレートに含める。`npm ci` が、どのプロダクトでも同じ依存を入れる。
- Starlight の既定の見た目のまま使う (`CTR-SITE-001`)。`requirements.css` は、Starlight の CSS の変数 (`--sl-color-*`) だけで色を決める。

## 設定

`.cumin-trace.toml` に、省略できる表 `[site]` を足す。`internal/core` は、この2つのキーを知っているキーとして読み、ほかのキーと同じく、知らないキーをエラーにする ([設定ファイル](../check/settings.md))。

| キー | 型 | 既定 | 決まり |
|---|---|---|---|
| `docs_dir` | 文字列 | `docs/ja` | `requirements_dir` と同じ決まりの相対パス。`requirements_dir` は、その中になければならない (`CTR-SITE-002`) |
| `base` | 文字列 | `/` | `/` で始まり `/` で終わる |

このリポジトリの値:

```toml
[site]
base = "/cumin-trace/"
```

`site/src/settings.mjs` は、`site/` の親をリポジトリの最上位として、`.cumin-trace.toml` を `smol-toml` で読む。`prefix`、`requirements_dir` (既定 `docs/ja/requirements`)、`docs_dir`、`base` を返す。値を確かめるのは Go の `check` で、サイトは確かめない。

## コマンド

`site/package.json` のスクリプト。すべて `site/` で動き、token もネットワークの GitHub も使わない (`CTR-SITE-005`、`CTR-SITE-009`)。

| コマンド | すること |
|---|---|
| `npm --prefix site ci` | 最初に1回。`package-lock.json` のとおりに依存を入れる。CI も同じ |
| `npm --prefix site run dev` | 手元のプレビュー (`astro dev`)。`CTR-SITE-005` の1つのコマンド |
| `npm --prefix site run build` | 静的なファイルを `site/dist/` に書く (`astro build`)。`CTR-SITE-008` |
| `npm --prefix site run preview` | `site/dist/` を手元で配信する (`astro preview`)。配信先と同じ base path で見る |
| `npm --prefix site run check-site` | `node scripts/check-site.mjs`。スクリプトが自分でビルドする (下の「確かめるコマンド」) |

## 確かめるコマンド

`scripts/check-site.mjs` は、次の順に動く。

1. 前の出力 `site/dist/` と、Astro のキャッシュ `site/.astro/` と `site/node_modules/.astro/` を消す。Astro は、`.cumin-trace.toml` が変わっても、キャッシュの描画済みの文書を使い回すためである (`prefix` を変えて確かめた)。
2. `docs_dir` の下のすべてのファイルの SHA-256 を取る。
3. `astro build` を子のプロセスで動かす (`node_modules/.bin/astro build`)。動かせなければその理由を、失敗したらその終了コードを1行出して、失敗で終わる。
4. もう一度 SHA-256 を取り、2 と比べる。
5. `site/dist/` の HTML を読む。

確かめるのは次のことである。1つでも合わなければ、ファイルと理由を1行ずつ出し、終了コード 1 で終わる。

| 確かめること | 要件 |
|---|---|
| `docs_dir` の下の各文書に、ページがある。ページの `<title>` が、文書のタイトル (frontmatter の `title`、なければ最初の見出し) で始まる。タイトルが空なら合わない。`<title>` はタグを持たないので、文字参照を戻し、空白を除いた文字で比べる | `CTR-SITE-002` |
| ビルドの前とあとで、`docs_dir` の下のファイルの中身が同じである (SHA-256 で比べる) | `CTR-SITE-002` |
| frontmatter に `title` のない文書のページで、`<h1>` が1つだけで、本文がタイトルの見出しを繰り返さない (最初の見出しが要件の見出しのときを除く) | `CTR-SITE-002` |
| frontmatter に `title` のある文書のページで、本文が最初の見出しを残す | `CTR-SITE-002` |
| 各要件の見出しに、ID と同じ `id` がある。各要件の見出しが `section.requirement` の最初の子で、その最初の子が `code.requirement-id` の ID である。見出しで始まらない `section.requirement` がない | `CTR-SITE-003`、`CTR-SITE-006` |
| `docs_dir` の下の各文書のページで、ID の参照のリンクが、`base` で始まり、行き先のページと `id` が出力にある。コードブロックの中にリンクがない | `CTR-SITE-003`、`CTR-SITE-008` |
| 元の文書の相対 `.md` のリンクで、宛先がサイトのページ ([ビルド](build.md) の「文書のリンク」) のものが、`base` で始まるページへのリンクになり、行き先のページがある。`#見出し` 付きのリンクは、行き先のページにその `id` がある。ない `id` は、文書の切れたリンクとして報告する | `CTR-SITE-010`、`CTR-SITE-008` |
| 元の文書の相対 `.puml` のリンクで、宛先が読める `.puml` のものごとに、閉じた `<details>` があり、中のコードブロックがその `.puml` の中身と同じである。ページに、`.puml` への相対のリンクがない。URL と `/` で始まるリンクは残る | `CTR-SITE-010` |
| 元の文書の、ほかの相対パスへのリンク (サイトのページでない `.md`、読める `.puml` でない `.puml`、画像でもないもの) が、出力では `<a>` のない文字である | `CTR-SITE-010` |
| 元の文書の相対パスの画像 (frontmatter、コードブロック、コードスパン、HTML のコメントの外の `![...](...)`) ごとに、ページに、出力のファイルの名前が `<名前>.<ハッシュ><拡張子>` (SVG でない画像は、拡張子が `.webp` でもよい) の `<img>` があり、その `src` が `base` で始まり、出力にファイルがある。同じバイトの SVG の画像は、Astro が1つの名前の1つのファイルにまとめるので、名前の合う `<img>` がなければ、出力のファイルが元の画像と同じ SHA-256 の `<img>` でもよい。1つの `<img>` は1つの画像にだけ数える。各ページに、`diagramScript` と同じ文の `<script>` がある | `CTR-SITE-004`、`CTR-SITE-007`、`CTR-SITE-008` |
| サーバの関数がない。`astro.config.mjs` に `adapter` がなく、`output` が `server` でない。`site/dist/` に `_worker.js` がない。`site/dist/` の `_functions/`、`functions/`、`server/` は、文書のページ (`docs_dir` の下の文書の `index.html`) だけを持つ。`server` や `functions` という名前の文書のディレクトリ (要件の領域など) のページはそこにできるので、受け付ける。空のディレクトリや、文書のページでないファイルを持つものは失敗する | `CTR-SITE-008` |

文書のリンクの確かめ方 (`CTR-SITE-010`):

- ページのリンクと画像は、本文だけから読む。本文は、`sl-markdown-content` の要素から、Starlight の前後のページへのリンク (`pagination-links`) の前までで、HTML のコメントを除く。
- 元の文書の、画像でないインラインのリンク (`[文字](宛先)`) と画像を、frontmatter、コードブロック、コードスパン、HTML のコメントの外から読む。ページはコメントの中を表示しないためである。次の行まで続くコメントは、行の頭の `<!--` (HTML のブロック) で始まるものだけである。宛先の種類は、プラグインと同じ `linkTarget()` で決める。
- 出力の `href` と宛先は、パーセントエンコードを戻してから比べる。Markdown の処理系が、`#見出し` の日本語をエンコードするためである。
- `.puml` の中身とコードブロックは、タグ、文字参照、空白を除いた文字で比べる。Expressive Code が、行ごとにタグで包むためである。

ファイルの種類は限らない。Starlight は既定で、検索の索引 (Pagefind) の `.pf_meta`、`.pf_fragment`、`.pf_index`、`.wasm` などを `site/dist/pagefind/` に書く (推測)。これらは静的なファイルである。

`CTR-SITE-007` の拡大の見た目と、配信先での動き (`CTR-SITE-008`) は、Maintainer が `demo` で見る。

## このリポジトリ

- `site/` は、`go run ./cmd/cumin-trace init CTR` で書く。すでにあるファイルは飛ばされる。
- Go のテスト (`templates` のパッケージ) が、`site/` の10のファイルが、`templates/site/` の対応するファイル (3 と 4 は `.` のない名前) と同じバイトであることを確かめる。git が追跡する `site/` のファイル (`git ls-files` の追跡済みと、無視されない未追跡のファイル) にテンプレートのないものがあっても失敗する。git が無視するファイル (`site/.gitignore` が名前を挙げる、どの深さの `node_modules/`、`dist/`、`.astro/` と、`.DS_Store` のような git の除外設定にあるファイル) は調べない。`go test ./...` はネットワークも Node.js も使わない。
- GitHub Pages への公開と、Pull Request でのサイトのビルドの workflow は、Maintainer が書く (要求Issue #88 の制約)。workflow は `npm --prefix site ci` と `npm --prefix site run check-site` を動かし、`site/dist/` を公開する。
