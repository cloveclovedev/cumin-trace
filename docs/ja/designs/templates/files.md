# ファイルと埋め込み

[テンプレートの生成の設計](index.md) の細部。

## ファイルの一覧

`init` は、次の順に書く。「埋め込み」は `templates/` の下のパスである。権限は、スクリプトだけ `0755`、ほかは `0644` で、ディレクトリは `0755` で作る。

| # | 書く先 | 埋め込み | 差し込む値 | 要件 |
|---|---|---|---|---|
| 1 | `.cumin-trace.toml` | `settings.toml` | `{{prefix}}`、`{{areas}}` | `CTR-TPL-001` |
| 2 | `docs/ja/requirements/index.md` | `docs/entry.md` | `{{prefix}}`、`{{area_rows}}` | `CTR-TPL-001` |
| 3 | `docs/ja/requirements/overview.md` | `docs/overview.md` | なし | `CTR-TPL-002` |
| 4 | `docs/ja/requirements/technology.md` | `docs/technology.md` | なし | `CTR-TPL-004` |
| 5 | `docs/ja/requirements/glossary.md` | `docs/glossary.md` | なし | `CTR-TPL-007` |
| 6 | `docs/ja/requirements/backlog.md` | `docs/backlog.md` | なし | `CTR-TPL-001` |
| 7 | `docs/ja/requirements/{{area_dir}}/index.md` (カテゴリがあるときだけ) | `docs/area.md` | `{{prefix}}`、`{{area}}` | `CTR-TPL-003` |
| 8 | `docs/ja/requirements/{{area_dir}}/overview.puml` (同上) | `docs/area-overview.puml` | なし | `CTR-TPL-003`、`CTR-TPL-006` |
| 9 | `docs/ja/requirements/{{area_dir}}/overview.svg` (同上) | `docs/area-overview.svg` | なし | `CTR-TPL-003`、`CTR-TPL-006` |
| 10 | `docs/ja/architecture/overview.md` | `docs/architecture.md` | なし | `CTR-TPL-009` |
| 11 | `.claude/skills/cumin-trace-docs/SKILL.md` | `skill/SKILL.md` | なし | `CTR-TPL-005` |
| 12 | `.claude/skills/cumin-trace-docs/templates/<名前>` | `docs/<名前>` の9つ (下の表) | 2 から 7、10 と同じ | `CTR-TPL-005`、`CTR-TPL-008` |
| 13 | `scripts/render-diagrams.sh` | `scripts/render-diagrams.sh` | なし | `CTR-TPL-006` |
| 14 | `tools/plantuml/Dockerfile` | `tools/plantuml/Dockerfile` | なし | `CTR-TPL-006` |
| 14a | `site/` の下のファイル ([サイトのファイルと設定](../site/files.md)) | `site/` の下の同じパス (先頭の `.` を除いた名前) | なし | `CTR-SITE-001` |
| 15 | `.github/workflows/cumin-trace-check.yml` | `workflows/cumin-trace-check.yml` | `{{version}}` | `CTR-CHK-005` |
| 16 | `.github/workflows/cumin-trace-site.yml` | `workflows/cumin-trace-site.yml` | `{{version}}` | `CTR-DASH-008` |

12 は、次の9つを、この順に書く。

| 名前 | 種類 | `docs/` にも書くか |
|---|---|---|
| `entry.md` | インデックス | 書く (2) |
| `overview.md` | 概要 | 書く (3) |
| `technology.md` | 技術要素 | 書く (4) |
| `glossary.md` | 用語集 | 書く (5) |
| `backlog.md` | バックログ | 書く (6) |
| `area.md` | カテゴリごとの要件 | 書く (7) |
| `architecture.md` | ソースコード全体構成 | 書く (10) |
| `design-index.md` | 設計文書の入口 | skill だけ |
| `design-detail.md` | 設計文書の細部 | skill だけ |

- 埋め込むパスに、`.claude`、`CLAUDE.md`、`.github/workflows` は現れない。書く先のパスは、`internal/scaffold/domain.go` の表だけが持つ。
- 15 と 16 は、バージョンを持つバイナリのときだけ書く ([リリースと workflow](release.md))。`internal/scaffold/domain.go` の表では、`NeedsVersion` で示す。

## カテゴリの図

8 と 9 は、カテゴリのページ (7) の「全体像」が示す図の例である。`CTR-TPL-003` の図を置く場所を埋め、`CTR-TPL-006` の demo (生成したスクリプトで、テンプレートの図を SVG にする) の対象になる。

- `overview.puml` は `@startuml overview` で始まり、英語のラベルだけを持つ。差し込む値を持たないので、どのプロダクトでも同じ文になる。
- `overview.svg` は、`overview.puml` を cumin-works と同じ PlantUML のイメージで描画したものを、そのまま埋め込む。生成した直後から、ページの図が見える。
- 生成したスクリプト (13) で描画し直すと、同じ SVG になる。埋め込んだ2つのファイルが同じ描画であることは、`init` のテストではなく、このリポジトリで描画し直して確かめる。
- カテゴリのページは、`![カテゴリの全体像](overview.svg)` と `図の元ファイル: [overview.puml](overview.puml)` で図を示す。

## 差し込む値

| 置き換える文字列 | 値 | 例 (`init PC TASK`) |
|---|---|---|
| `{{prefix}}` | 引数の接頭辞 | `PC` |
| `{{area}}` | 引数のカテゴリ | `TASK` |
| `{{area_dir}}` | カテゴリの小文字 (`strings.ToLower`) | `task` |
| `{{areas}}` | `areas` の配列。カテゴリがなければ空 | `["TASK"]`。カテゴリがなければ `[]` |
| `{{area_rows}}` | インデックスの、カテゴリの表の行。カテゴリがなければ空の文字列 | `TASK` の1行 (略号と、`task/index.md` へのリンク) |
| `{{version}}` | バイナリのバージョン | `v0.1.0` |

カテゴリがないとき、`{{area}}` は `<AREA>` になる。これが文に現れるのは skill のテンプレート (12 の `area.md`) だけである。カテゴリがあるとき、skill のテンプレートは最初のカテゴリ (例 `TASK`) を持つ。どちらの場合も、skill は Agent に、カテゴリを決めてから、すべての ID のカテゴリを置き換えるように指示する。見出しは、テンプレートのコメントに従ってカテゴリの名前に書き換える。入口のページのカテゴリの表には、新しいカテゴリの行を足す。新しいカテゴリのページは `<requirements_dir>/<カテゴリの小文字>/index.md` に書き、同じディレクトリに `overview.puml` と `overview.svg` を描く。

`.cumin-trace.toml` の中身:

```toml
# The settings of cumin-trace for the requirement documents of this repository
prefix = "{{prefix}}"

# The allowed areas. Add an area here before you write its first requirement.
areas = {{areas}}
```

入口のページ (2) は、「全体」の節で、概要 (3)、技術要素 (4)、用語集 (5)、バックログ (6)、ソースコード全体構成 (10) にリンクする。3 から 6 と 10 は、それぞれの要件 (`CTR-TPL-002`、`CTR-TPL-004`、`CTR-TPL-007`、`CTR-TPL-009`) の見出しと表の列を持つ。見出しと表のほかに見える文は、バックログの「項目ごとに、内容、後回しにした理由、見直すきっかけを書く。」の1行だけである。ほかは、何を書くかを説明するコメントである。技術要素と用語集は、IDを持たない。ソースコード全体構成は、図を描くまで画像を示さず、図を置く行をコメントに持つ。

カテゴリがないとき、生成した直後の文書には要件がなく、書式検査はエラーなしで通る。カテゴリのページの記入例は、`{{prefix}}-{{area}}-001` の要件を1つ持ち、`検証` の行と区切り線を持つ。生成した直後のファイルは、書式検査をエラーなしで通る (`CTR-TPL-001`)。

## skill

`SKILL.md` は英語で書き、`CTR-TPL-005` の7つの指示を持つ。先頭の front matter に `name: cumin-trace-docs` と `description` を書く。テンプレートは `templates/` の下にあり、`SKILL.md` は表で、種類ごとにその名前と、形が cumin-trace の決めるもの (`Set by cumin-trace`) か推奨 (`Recommended`) かを示す。

| 形 | 種類 | 指示 |
|---|---|---|
| cumin-trace が決める | インデックス、概要、カテゴリごとの要件、用語集、バックログ | 新しく書くとき、テンプレートから書く |
| 推奨 | 技術要素、ソースコード全体構成、設計文書の入口と細部 | リポジトリにその種類の文書があれば、その形に合わせる。なければテンプレートから書く |

書式の文書は、プロダクトのリポジトリにないので、このリポジトリの main の書式の文書の URL で示す (カテゴリのテンプレートと同じ)。

設計文書の入口 (`design-index.md`) は、`CTR-TPL-008` の4つの見出しを持つ。決めたことの記入例は、[テンプレートの生成の設計](index.md) と [書式検査の設計](../check/index.md) と同じ「理由」「採らなかった案」「細部」の行を持つ。細部 (`design-detail.md`) は、正確な値を書く「決まり」と、外部の API やライブラリを書く「確かめたこと」(バージョン、公式ページ、手元の実行、未確認) の2つの節を持つ。[設定ファイル](../check/settings.md) が最初の例である。どちらも、書き換える場所をコメントで示し、差し込む値を持たない。

## 図のスクリプト

`scripts/render-diagrams.sh` と `tools/plantuml/Dockerfile` は、cumin-works の同じ名前のファイルを写す。書いた時点の cumin-works の main のコミットを、スクリプトの先頭のコメントに書く。そのコメントは使い方の行 (2 から 6 行目) のあとに置く。スクリプトは、引数の数が違うとき、その5行を使い方として出す。

- スクリプトのイメージ `cumin-plantuml:<バージョン>` と、Dockerfile の `FROM plantuml/plantuml:<バージョン>` は、同じバージョンを持つ。テストで確かめる。
- 引数の数が 0 でも 2 でもないとき、スクリプトは Docker を使う前に、使い方を標準エラー出力に出し、終了コード 2 で終わる。

| 使い方 | 動き |
|---|---|
| `scripts/render-diagrams.sh` | `docs/` の下のすべての `.puml` を、隣の SVG にする |
| `scripts/render-diagrams.sh <in.puml> <out.svg>` | 1つの図を、指定した SVG にする |
| `scripts/render-diagrams.sh --check <base>` | `<base>` と違う図を描画し、SVG と比べる。ファイルは変えない |

## 確かめたこと

- `embed` の `//go:embed` は、`.` か `_` で始まるファイルを、ディレクトリの指定からは外す: <https://pkg.go.dev/embed> (Go 1.27.2 の `go doc embed` で読んだ)。埋め込むファイルの名前は、どれも `.` で始めない。
- cumin-works の `scripts/render-diagrams.sh` と `tools/plantuml/Dockerfile` を、GitHub の API (`gh api repos/cloveclovedev/cumin-works/contents/...`) で読んだ。main のコミット `f1e4d22f9ea85a299828b1d8ff7f08c7d9eea3b4` のものを写した。
- 手元の実行: 空のディレクトリで `init PC TASK` のあと、生成したスクリプトを引数なしで実行し、カテゴリの図 (8) を描画した。描画した SVG は、埋め込んだ SVG (9) と同じバイト列になった。
- Claude Code の skill は `.claude/skills/<名前>/SKILL.md` に置き、front matter の `name` と `description` を読む。同じディレクトリの別のファイルを、`SKILL.md` から示して使える: <https://code.claude.com/docs/en/skills>
- 手元の実行: この文書の SVG は、cumin-works の Dockerfile と同じイメージで描画した。既存の `check-packages.svg` を描画し直して、同じバイト列になった。
