# サイトの設計

要件文書を Astro + Starlight のサイトとして表示する仕組みを決める。`cumin-trace init` がサイトのテンプレートを `site/` に書き、Starlight のビルドが、文書と図だけから文書のページを作る。ダッシュボードのページ (`CTR-DASH-001`) は、対応づけの結果 (`site/trace.json`) を読む。その設計は [ダッシュボードの設計](../dashboard/index.md) で決める。

## 要件

- [サイト](../../requirements/site/index.md)
  - `CTR-SITE-001` サイトのテンプレート ([ファイルと設定](files.md))
  - `CTR-SITE-002` 文書を書き換えずに表示する ([ビルド](build.md))
  - `CTR-SITE-003` 要件へのリンク ([ビルド](build.md))
  - `CTR-SITE-004` 図の表示 ([ビルド](build.md))
  - `CTR-SITE-005` 手元でのプレビュー ([ファイルと設定](files.md))
  - `CTR-SITE-006` 要件の区切りの表示 ([ビルド](build.md))
  - `CTR-SITE-007` 図の拡大 ([ビルド](build.md))
  - `CTR-SITE-008` 配信先で動く ([ファイルと設定](files.md))
  - `CTR-SITE-009` 文書だけから作る ([ビルド](build.md))
  - `CTR-SITE-010` 文書の間のリンク ([ビルド](build.md))
  - `CTR-SITE-011` サイドバーの順 ([ビルド](build.md))
  - `CTR-SITE-012` 表示の言語と階層 ([ビルド](build.md))
  - `CTR-SITE-013` サイトの名前 ([ファイルと設定](files.md))
  - `CTR-SITE-014` 本文のデザイン ([ファイルと設定](files.md))
  - `CTR-SITE-015` サイドバーの見出し ([ビルド](build.md))
- [コマンド](../../requirements/command/index.md)
  - `CTR-CMD-001` コマンドの一覧 (`init` がサイトのテンプレートも書く)

## 全体像

[ソースコード全体構成](../../architecture/overview.md) の `internal/scaffold`、最上位の `templates` の `site/`、プロダクトのリポジトリの `site/` にあたる。黄色が、この文書で決めるものである。

![サイトのパッケージ](site-packages.svg)

図の元ファイル: [site-packages.puml](site-packages.puml)

![サイトのビルドのシーケンス](site-sequence.svg)

図の元ファイル: [site-sequence.puml](site-sequence.puml)

## 公式文書の確認

Astro 7.4.0 と Starlight 0.42.6 に固定する (下の「バージョン」)。公式文書のサイトは最新版だけを示す。2026-10-17 に `npm view` で、最新版がこの2つであることを確かめてから読んだ。

| 事実 | 確かめたページ |
|---|---|
| Starlight は、すべてのページに `title` を求める | Starlight [Frontmatter Reference](https://starlight.astro.build/reference/frontmatter/) |
| `docs` のコレクションは `docsLoader()` と `docsSchema()` で定義する。既定では `src/content/docs/` の文書がページになる | Starlight [Manual Setup](https://starlight.astro.build/manual-setup/) |
| `glob()` の `base` は、`pattern` を解決するディレクトリである。ローダーの文脈の `parseData()` は、コレクションのスキーマでデータを確かめる | Astro [Content Loader API](https://docs.astro.build/en/reference/content-loader-reference/) |
| Astro 7 の既定の Markdown の処理系は Sätteri である。`@astrojs/markdown-satteri` の `satteri()` を `markdown.processor` に渡し、`satteri` の `defineHastPlugin` で書いたプラグインを `hastPlugins` に足せる | Astro [Markdown](https://docs.astro.build/en/guides/markdown-content/) |
| Sätteri の構文木は remark と rehype と同じ mdast と hast で、プラグインの API だけが違う。`ctx.setProperty()`、`ctx.replaceNode()`、`ctx.parent()`、`ctx.fileURL` を使える | Sätteri [Plugin API](https://satteri.bruits.org/docs/plugin-api/) |
| Starlight の `customCss` で CSS を、`head` でタグを足せる。`markdown.processedDirs` で、`src/content/docs/` の外のディレクトリにも Starlight の Markdown の処理をかける | Starlight [Configuration Reference](https://starlight.astro.build/reference/configuration/) |
| `base` は、開発とビルドの両方で、ページと資産の根になる | Astro [Configuration Reference](https://docs.astro.build/en/reference/configuration-reference/) |
| Astro は Node.js `v22.22.0` 以上を求め、奇数のバージョンを対象にしない | Astro [Install and set up](https://docs.astro.build/en/install-and-setup/) |
| 1つの言語だけのサイトは、`locales` の `root` に `label` と `lang` を書く。`root` の言語のページの URL には、言語の接頭辞が付かない。`root` が既定の言語なら `defaultLocale` は要らない。Starlight は、日本語を含む多くの言語の UI の訳を持つ | Starlight [Internationalization (i18n)](https://starlight.astro.build/guides/i18n/) |
| `locales` のキーは、その言語のファイルを置くディレクトリである | Starlight [Configuration Reference](https://starlight.astro.build/reference/configuration/) の `locales` |
| Starlight 0.42.6 は、日本語の UI の訳 `packages/starlight/src/translations/ja.json` を持つ | Starlight のリポジトリの、タグ `@astrojs/starlight@0.42.6` のファイル |
| 既定のサイドバーは、ファイルの構成から作り、ファイルの名前の順に並ぶ。`sidebar` には、ページの `slug` (文字だけの短い書き方も可)、`label` と `items` の組 (入れ子にできる) を並べる。`slug` の項目の表示は、既定でそのページの `title` である | Starlight [Sidebar Navigation](https://starlight.astro.build/guides/sidebar/) |
| `title` は、サイトの名前で、文字か、言語ごとの文字の表である。メタデータとブラウザのタブのタイトルにも使う | Starlight [Configuration Reference](https://starlight.astro.build/reference/configuration/) の `title` |
| 本文の幅は、CSS の変数 `--sl-content-width` で決まる。`customCss` に、CSS のファイルのパスを足す | Starlight [CSS & Styling](https://starlight.astro.build/guides/css-and-tailwind/) |
| 0.42.6 では、`--sl-content-width` の既定は `45rem` である。`.sl-container` の `max-width` と、`.main-pane` の幅の計算が、この変数を使う。どちらの規則も `@layer starlight.core` の中にある。角の丸みの変数はない | Starlight のリポジトリの、タグ `@astrojs/starlight@0.42.6` の `src/style/props.css`、`src/components/ContentPanel.astro`、`src/components/TwoColumnContent.astro` |

上の表にない Astro と Starlight の振る舞いは、この文書では推測として書き、「推測」と記す。実装の Issue で、確認のコマンド (`npm run check-site`) で確かめる。表の、1つの言語のサイトの行から下の7行は、2026-10-19 に読んだ。

## 決めたこと

### テンプレートは `init` が `site/` に書く

`cumin-trace init` が、ほかのテンプレートに続けて、`site/` の下のファイルを書く。書き方は、ほかのテンプレートと同じく、足りないファイルだけを書き、上書きしない ([テンプレートの生成](../templates/index.md))。サイトのためのコマンドは足さない。

- 理由: `CTR-SITE-001` はテンプレートの生成を求め、`init` はすでに「足りないファイルだけを書く」仕組みを持つ。コマンドを足さなければ、`CTR-CMD-001` の表のコマンドは増えない。表の `init` の行に `CTR-SITE-001` を足すだけである。
- 採らなかった案: `cumin-trace site init` などの新しいコマンド。同じ書き方の仕組みが2つになる。
- 採らなかった案: `npm create astro` の Starlight のテンプレートを使わせ、手で直させる。ID のアンカーとリンクの部品を、プロダクトごとに書き写すことになる。
- 細部: [ファイルと設定](files.md)

### 設定は `.cumin-trace.toml` の `[site]` に置く

文書のディレクトリ (`docs_dir`、既定 `docs/ja`) と、サイトの base path (`base`、既定 `/`) を、`.cumin-trace.toml` の `[site]` の表に置く。サイトは、同じファイルの `prefix` と `requirements_dir` も読む。`internal/core` は、`[site]` を知っているキーとして受け付け、値を確かめる。

- 理由: `CTR-SITE-002` は「リポジトリの設定」を求める。接頭辞と要件のディレクトリはすでにこのファイルにある。base path をここに置くと、`site/` のファイルは、どのリポジトリでもテンプレートと同じになる。
- 採らなかった案: `site/astro.config.mjs` に定数で書く。このリポジトリの `site/` がテンプレートと違うファイルになり、ずれを確かめられない。
- 採らなかった案: 環境変数。手元のプレビューと配信で、値を2か所に書くことになる。
- 細部: [ファイルと設定](files.md)

### 文書は `src/content/docs/` に写さず、ローダーで直接読む

`site/src/content.config.mjs` の `docs` のコレクションは、Astro の `glob()` を包む小さなローダーで、`docs_dir` の下の `**/*.md` を読む。包むローダーは、`parseData()` に渡す前に、frontmatter に `title` がなければ、文書の最初の見出しの文字を `title` にする。スキーマは `docsSchema()` のままである。

- 理由: `CTR-SITE-002` は元のファイルを書き換えないことを求める。`parseData()` はスキーマを確かめる前に呼ばれるので、そこで `title` を足せば、Starlight の求める `title` が満たされる。写しを作らないので、プレビュー中の文書の変更がそのまま映る (推測)。
- 採らなかった案: ビルドの前に、frontmatter を足した写しを `src/content/docs/` に作る。写しを作る手順がビルドとプレビューの前に要り、写しと元がずれる。
- 採らなかった案: remark のプラグインで `data.astro.frontmatter.title` を足す。コレクションのスキーマはレンダリングより前にデータを確かめるので、`title` がないエラーが先に出る (推測)。
- 細部: [ビルド](build.md)

### ID のアンカー、参照のリンク、要件の枠は、Sätteri の1つのプラグインで作る

`site/src/requirements.mjs` が、Sätteri の hast のプラグインとして、要件の見出しに ID のアンカーを付け、要件を `<section>` で囲み、ID の参照をリンクにする。文書の間のリンク (`CTR-SITE-010`) は、同じファイルの2つ目のプラグイン (`linksPlugin`) が、このプラグインより先に書き換える。参照を読む場所と4つの決まりは、書式検査と同じにする。参照は、テキストとコードスパンのノードの値 (表示される文字) で読む。Markdown の処理系は、既定の Sätteri のままにする。

- 理由: 値で読むのが最も簡単である。書式検査はエスケープの前の文字で読む ([文書の読み方](../check/reading.md)) ので、エスケープした ID だけは検査と結果が違いうる。このリポジトリの要件文書は ID をエスケープして書かないので、その違いを受け入れる。Sätteri は Astro 7 と Starlight 0.42.6 の既定の処理系で、両方が最初に支える道である。Sätteri の構文木は remark と同じ mdast と hast である。Astro の見出しの `id` を付けるプラグインは、利用者の hast のプラグインのあとに動き、すでにある `id` を残す (`@astrojs/markdown-satteri` 0.4.4 のコードで確かめた)。
- 採らなかった案: ノードの元の文字の位置 (unist の `offset`) から元の文字を取り出し、検査と同じ文字で読んで、値の位置に写す。エスケープ、文字参照、行頭の空白を数える写し方が要るが、このリポジトリにはそれを要する文書がない。
- 採らなかった案: `@astrojs/markdown-remark` の `unified()` と remark のプラグイン。依存が1つ増え、Astro の既定の処理系から外れる。
- 細部: [ビルド](build.md)

### 図は、文書の隣の SVG をそのまま使い、クリックで SVG のファイルを開く

画像のパスは、文書のファイルからの相対パスとして Astro が解決する。`./` も `../` もない相対パス (`site-build.svg`) も、そのまま解決される。本文の画像をクリックすると、1行のインラインのスクリプトが、その SVG のファイルを開く。ブラウザが SVG だけを元の大きさで表示する。npm のパッケージは足さない。

- 理由: `CTR-SITE-004` と `CTR-SITE-007` を、静的なファイルだけで満たす (`CTR-SITE-008`)。SVG は拡大しても文字が崩れない。
- 採らなかった案: プラグインで、`./` のない相対パスに `./` を足す。Astro は Markdown の画像のパスを mdast の段階で集め、hast の `src` がその文字と同じ `<img>` だけを画像として扱う (`@astrojs/markdown-satteri` 0.4.4 のコードで確かめた)。hast で `src` を変えると、画像が出力に置かれない。
- 採らなかった案: `starlight-image-zoom` (0.16.2)。依存が1つ増え、画面の中に重ねて表示するので、横に長い図は画面の幅より大きくならない。
- 採らなかった案: 画像を、プラグインで SVG へのリンクで包む。Astro が出力に置いた画像の URL を、プラグインの段階では知らない (推測)。
- 採らなかった案: 画面に重ねて表示する、自分で書いた拡大表示の JavaScript。保守するコードが増える。
- 細部: [ビルド](build.md)

### バージョンは正確な値で固定し、cumin-trace のリリースと一緒に上げる

`site/package.json` は、`astro` を `7.4.0`、`@astrojs/starlight` を `0.42.6`、`@astrojs/markdown-satteri` を `0.4.4`、`satteri` を `0.10.5`、`smol-toml` を `1.9.1` に、範囲 (`^`) なしで書く。Node.js は、`site/.node-version` に `24` を書き、`engines` に `>=22.22.0` を書く。`site/package-lock.json` もテンプレートに含め、`npm ci` が同じ依存を入れる。上げるときは、Maintainer が、このリポジトリのテンプレートと `site/` を同じ Pull Request で上げ、`npm run check-site` を通す。すでにテンプレートを書いたプロダクトは、`init` が上書きしないので、自分で上げる。

- 理由: `technology.md` は、固定したバージョンの公式文書を読むことを求める。範囲で書くと、`npm install` のたびに読むべき文書が変わる。Node.js の `24` は偶数のバージョンで、Astro の求める `v22.22.0` 以上に入る。`smol-toml` は `.cumin-trace.toml` を読むためで、依存のない小さな TOML のパーサである。
- 採らなかった案: `^` の範囲。プロダクトごとに違うバージョンが入る。
- 採らなかった案: Dependabot。設定は `.github/` に置き、Agent が書けない。必要になれば Maintainer が足す。
- 採らなかった案 (Docker): サイトを Docker のコンテナでビルドする。要求Issueの制約 (Node と npm でビルドする) に合わない。

### このリポジトリの `site/` は `init` で書き、テンプレートとの一致をテストで確かめる

このリポジトリも、`go run ./cmd/cumin-trace init CTR` で書いた `site/` をコミットする。`.cumin-trace.toml` に `[site]` の `base = "/cumin-trace/"` を書く。Go のテストが、`site/` の各ファイルが `templates/site/` と同じであることを確かめる。

- 理由: このリポジトリのサイトが、プロダクトに書くものと同じテンプレートで動くことを、毎回確かめられる。
- 採らなかった案: `templates/site/` から直接ビルドする。リポジトリの最上位が `../../` になり、プロダクトの形と違う。
- 細部: [ファイルと設定](files.md)

### サイトの振る舞いは、1つのコマンドで確かめる

`npm --prefix site run check-site` が、前の出力と Astro のキャッシュを消し、文書のハッシュを取り、サイトをビルドし、もう一度ハッシュを取り、出力の HTML を調べる。調べるのは、タイトル、アンカー、リンクの行き先、図のファイル、要件の枠、元の文書が変わらないことである。Node.js の標準の機能だけで書く。

- 理由: `CTR-SITE-001` から `CTR-SITE-013` の検証は `demo` で、毎回ブラウザで見るのは繰り返せない。HTML の形で確かめられる部分を、どの手元でも同じ結果になるコマンドにする。
- 採らなかった案: Playwright でブラウザを動かす。大きな依存が増える。クリックの見た目は `demo` で Maintainer が見る。
- 細部: [ファイルと設定](files.md)

### サイドバーは、`docs_dir` の `index.md` の見出しとリンクの順に、設定で並べる

`astro.config.mjs` が、Starlight の `sidebar` を、`requirements.mjs` の `sidebar()` で作る。`sidebar()` は、`docs_dir` の `index.md` の本文のリンクを順に読む。宛先がサイトのページのリンクだけを使う。`docs_dir` の `index.md` の `##` 見出しは、最上位のグループになり、その下のリンクがグループの中に並ぶ。ディレクトリの `index.md` へのリンクは、そのディレクトリのグループになり、中も同じ決まりで並ぶ。項目の表示は、リンクの文字である。`index.md` のページしか持たないグループは、そのページの項目1つになる。`docs_dir` の `index.md` のページは、サイトの左上の名前から開くので、サイドバーには出さない。どこからもリンクしていない文書は、そのあとにパスの順で並ぶ (`CTR-SITE-011`、`CTR-SITE-015`)。

- 理由: Starlight の既定のサイドバーは、ファイルの名前の順である。読む順は、`docs_dir` の `index.md` だけが知っている。`sidebar` の設定は、文書を書き換えずに順を決められる (`CTR-SITE-002`)。グループを作らないと、40 ほどのページが1列に並ぶ。
- 理由: 要件のページは、カテゴリごとのディレクトリの `index.md` だけのことが多い。ディレクトリだけでグループを作ると、同じ名前のページを1つ持つグループが並ぶ。見出しのグループなら、ディレクトリの構成を変えずに、「概要」「要件」「設計」のように最上位を分けられる。`docs_dir` の `index.md` を書き換えれば、サイドバーも変わる。
- 理由: ページのタイトルは、そのページだけで読める名前 (「サイトの設計」など) である。サイドバーでは、グループの名前があるので短くしたい (「設計」の下の「サイト」)。リンクの文字は、`docs_dir` の `index.md` を書く人が、その場所に合わせて決められる。
- 理由: `docs_dir` の `index.md` は、サイドバーと同じリンクを並べた文書になる。サイドバーにも置くと、同じ並びを2回見せることになる。
- 採らなかった案: 各文書の frontmatter の `sidebar.order` で順を決める。文書を書き換えることになる。
- 採らなかった案: `docs_dir` の `index.md` の箇条書きの入れ子を、グループの入れ子にする。リンクのない項目 (このリポジトリの以前の「設計文書」) の扱いの決まりが増える。見出しなら、グループの名前の行がはっきりする。
- 採らなかった案: グループのために、文書のディレクトリを `requirements/areas/` のように分け直す。URL とすべての相対リンクが変わり、カテゴリの一覧の文書が2つになる。
- 採らなかった案: リンクした文書だけを1列に並べる。リンクしていない文書がグループに入らず、長い1列になる。
- 細部: [ビルド](build.md) の「サイドバー」

### UI は `root` の言語の日本語にし、URL は文書のディレクトリの中から始める

Starlight の `locales` に `root` の1つだけを置き、`lang` を `ja`、`label` を `日本語` にする。ページの ID は、今のとおり `docs_dir` からの相対パスで決める (`pageID()`)。サイドバーは上の `sidebar()` だけで作る。どちらにも、`docs_dir` より上のディレクトリと `ja` は出ない (`CTR-SITE-012`)。言語は、今はテンプレートに固定する。設定のキーは足さない。

- 理由: `root` の言語は、URL に言語の接頭辞を付けずに、UI の言語だけを変える。Starlight は日本語の UI の訳を持つ。英語のサイトは v0.3 の候補 ([バックログ](../../requirements/backlog.md)) で、今は決めない。
- 英語のサイトの余地: Starlight の言語は、ページの ID の最初のディレクトリで決まる (推測)。英語の文書を足すときは、ローダーが英語の文書に `en/` で始まる ID を付け、`locales` に `en` を足せば、日本語の URL は変わらない (推測)。言語を選ぶ設定のキーは、そのときに足す。
- 採らなかった案: `docs/` を読み、`ja` を `locales` のキーにする。URL とサイドバーに `ja` が出る。
- 採らなかった案: `docs_dir` の最後のディレクトリの名前 (`ja`) から言語を決める。名前が言語のコードでないリポジトリで、決まりが働かない。
- 細部: [ビルド](build.md) の「言語」

### サイトの名前は、`[site]` の `name`、`origin` のリポジトリの名前、接頭辞の順に決める

`settings.mjs` の `siteName()` が、Starlight の `title` を決める (`CTR-SITE-013`)。

1. `.cumin-trace.toml` の `[site]` の `name`
2. `git config --get remote.origin.url` の URL から、末尾の `/` と `.git` を除き、最後の部分を取った名前
3. `prefix`

`git` がない、リポジトリでない、`origin` がない、名前が空のときは、3 に進む。エラーにしない。このリポジトリは `name = "cumin-trace"` を書くので、GitHub Pages のビルドも、手元と同じ名前になる。

- 理由: キーの名前は、要件の言葉「名前」に合わせ、ページのタイトル (frontmatter の `title`) と分ける。`origin` の URL は、手元の clone にも CI の checkout にもある (CI は推測)。ネットワークも token も使わない。
- 採らなかった案: `GITHUB_REPOSITORY` の環境変数。GitHub Actions の中にしかなく、手元のプレビューと名前が変わる。
- 採らなかった案: リポジトリの最上位のディレクトリの名前。worktree や clone の名前が、リポジトリの名前と違うことがある。
- 採らなかった案: GitHub の API で名前を読む。ネットワークと token が要り、`CTR-SITE-009` と `CTR-SITE-005` に合わない。
- 採らなかった案: キーを `title` にする。文書のタイトルと同じ言葉になる。
- 細部: [ファイルと設定](files.md) の「設定」

### すべてのページで、CSS だけで本文の幅と画像の角を変える

`requirements.css` が、すべてのページで、`:root` の `--sl-content-width` を `60rem` (既定は `45rem`) にする。本文の画像 (`.sl-markdown-content img`) の角を、`border-radius: 0.5rem` で丸める。要件の枠と同じ値である (`CTR-SITE-014`)。

- 理由: `CTR-SITE-014` は、サイトのすべてのページの本文を広げ、画像の角を丸めることを求める。ページを選ぶ決まりは要らない。Starlight は、本文の枠 (`.sl-container`) と本文の欄 (`.main-pane`) の両方の幅を `--sl-content-width` から決める。変数を変えれば、両方が同じに広がる。Starlight には角の丸みの変数がないので、値を書く。
- 採らなかった案: 要件の枠 (`section.requirement`) を持つページだけを、CSS の `:has()` で選んで広げる。設計文書のページが狭いまま残り、`CTR-SITE-014` に合わない。
- 採らなかった案: `.sl-container` の `max-width` だけを広げる。`.main-pane` の幅は変わらないので、画面の幅が 120rem ほどになるまで、本文はほとんど広がらない。
- 採らなかった案: `--sl-content-width` を、自分自身の値から計算し直す (`calc(var(--sl-content-width) * 4 / 3)`)。CSS の変数の循環になり、無効になる。
- 細部: [ファイルと設定](files.md) の「テンプレートのファイル」

### workflow は変えない

`.github/workflows/` のファイルは、どれも変えない。

- 理由: `site.yml` は、`docs/ja/`、`site/`、`.cumin-trace.toml` の変更でサイトをビルドし、`check-site` を動かす。新しいキー `name` は `.cumin-trace.toml` にあり、サイドバーと言語と CSS は `site/` にある。このリポジトリの名前は設定から来るので、checkout の `origin` に頼らない。
