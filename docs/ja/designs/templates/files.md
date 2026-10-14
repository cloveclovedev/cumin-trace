# ファイルと埋め込み

[テンプレートの生成の設計](index.md) の細部。

## ファイルの一覧

`init` は、次の順に書く。「埋め込み」は `templates/` の下のパスである。権限は、スクリプトだけ `0755`、ほかは `0644` で、ディレクトリは `0755` で作る。

| # | 書く先 | 埋め込み | 差し込む値 | 要件 |
|---|---|---|---|---|
| 1 | `.cumin-trace.toml` | `settings.toml` | `{{prefix}}`、`{{area}}` | `CTR-TPL-001` |
| 2 | `docs/ja/requirements/index.md` | `docs/entry.md` | `{{prefix}}`、`{{area}}`、`{{area_dir}}` | `CTR-TPL-001` |
| 3 | `docs/ja/requirements/overview.md` | `docs/overview.md` | なし | `CTR-TPL-002` |
| 4 | `docs/ja/requirements/technology.md` | `docs/technology.md` | なし | `CTR-TPL-004` |
| 5 | `docs/ja/requirements/glossary.md` | `docs/glossary.md` | なし | `CTR-TPL-007` |
| 6 | `docs/ja/requirements/backlog.md` | `docs/backlog.md` | なし | `CTR-TPL-001` |
| 7 | `docs/ja/requirements/{{area_dir}}/index.md` | `docs/area.md` | `{{prefix}}`、`{{area}}` | `CTR-TPL-003` |
| 8 | `docs/ja/architecture/overview.md` | `docs/architecture.md` | なし | `CTR-TPL-009` |
| 9 | `.claude/skills/cumin-trace-docs/SKILL.md` | `skill/SKILL.md` | なし | `CTR-TPL-005` |
| 10 | `.claude/skills/cumin-trace-docs/templates/<名前>` | `docs/<名前>` の9つ (下の表) | 2 から 8 と同じ | `CTR-TPL-005`、`CTR-TPL-008` |
| 11 | `scripts/render-diagrams.sh` | `scripts/render-diagrams.sh` | なし | `CTR-TPL-006` |
| 12 | `tools/plantuml/Dockerfile` | `tools/plantuml/Dockerfile` | なし | `CTR-TPL-006` |
| 13 | `.github/workflows/cumin-trace-check.yml` | `workflows/cumin-trace-check.yml` | `{{version}}` | `CTR-CHK-005` |

10 は、次の9つを、この順に書く。

| 名前 | 種類 | `docs/` にも書くか |
|---|---|---|
| `entry.md` | 入口のページ | 書く (2) |
| `overview.md` | 概要 | 書く (3) |
| `technology.md` | 技術要素 | 書く (4) |
| `glossary.md` | 用語集 | 書く (5) |
| `backlog.md` | backlog | 書く (6) |
| `area.md` | カテゴリごとの要件 | 書く (7) |
| `architecture.md` | ソースコード全体構成 | 書く (8) |
| `design-index.md` | 設計文書の入口 | skill だけ |
| `design-detail.md` | 設計文書の細部 | skill だけ |

- 埋め込むパスに、`.claude`、`CLAUDE.md`、`.github/workflows` は現れない。書く先のパスは、`internal/scaffold/domain.go` の表だけが持つ。
- 1 から 8 は #34 と #35、9 と 10 は #36、11 と 12 は #37、13 は #38 で足す。足すまでは、一覧にその行がない。

## 差し込む値

| 置き換える文字列 | 値 | 例 (`init PC TASK`) |
|---|---|---|
| `{{prefix}}` | 引数の接頭辞 | `PC` |
| `{{area}}` | 引数のカテゴリ | `TASK` |
| `{{area_dir}}` | カテゴリの小文字 (`strings.ToLower`) | `task` |
| `{{version}}` | バイナリの版 | `v0.1.0` |

`.cumin-trace.toml` の中身:

```toml
# The settings of cumin-trace for the requirement documents of this repository
prefix = "{{prefix}}"

# The allowed areas. Add an area here before you write its first requirement.
areas = ["{{area}}"]
```

カテゴリのページの記入例は、`{{prefix}}-{{area}}-001` の要件を1つ持ち、`検証` の行と区切り線を持つ。生成した直後のファイルは、書式検査をエラーなしで通る (`CTR-TPL-001`)。

## skill

`SKILL.md` は英語で書き、`CTR-TPL-005` の6つの指示を持つ。テンプレートは `templates/` の下にあり、`SKILL.md` は種類ごとにその名前を示す。

## 図のスクリプト

`scripts/render-diagrams.sh` と `tools/plantuml/Dockerfile` は、cumin-works の同じ名前のファイルを写す。書いた時点の cumin-works の main のコミットを、スクリプトの先頭のコメントに書く。

| 使い方 | 動き |
|---|---|
| `scripts/render-diagrams.sh` | `docs/` の下のすべての `.puml` を、隣の SVG にする |
| `scripts/render-diagrams.sh <in.puml> <out.svg>` | 1つの図を、指定した SVG にする |
| `scripts/render-diagrams.sh --check <base>` | `<base>` と違う図を描画し、SVG と比べる。ファイルは変えない |

## 確かめたこと

- `embed` の `//go:embed` は、`.` か `_` で始まるファイルを、ディレクトリの指定からは外す: <https://pkg.go.dev/embed> (Go 1.27.2 の `go doc embed` で読んだ)。埋め込むファイルの名前は、どれも `.` で始めない。
- cumin-works の `scripts/render-diagrams.sh` と `tools/plantuml/Dockerfile` を、GitHub の API (`gh api repos/cloveclovedev/cumin-works/contents/...`) で読んだ。
- 手元の実行: この文書の SVG は、cumin-works の Dockerfile と同じイメージで描画した。既存の `check-packages.svg` を描画し直して、同じバイト列になった。
