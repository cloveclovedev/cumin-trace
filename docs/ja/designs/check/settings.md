# 設定ファイル

[書式検査の設計](index.md) の細部。

## キー

`.cumin-trace.toml` は、リポジトリの最上位に置く。書式検査が読むキーは次の3つである (`CTR-FMT-011`、`CTR-FMT-001`)。

```toml
# プロダクトの接頭辞
prefix = "CTR"

# 使ってよいカテゴリ。書いた順が、出力の順になる (CTR-EXP-002)
areas = ["FMT", "TPL", "CHK", "EXP", "TRC", "SITE", "DASH"]

# 要件文書のディレクトリ。省略できる
requirements_dir = "docs/ja/requirements"
```

| キー | 型 | 必須 | 決まり |
|---|---|---|---|
| `prefix` | 文字列 | 必須 | 英大文字と数字が1文字以上 |
| `areas` | 文字列の配列 | 必須 | 0個以上。空 (`areas = []`) は、カテゴリがまだないことを表す。それぞれ英大文字と数字が1文字以上。同じ名前を2回書かない |
| `requirements_dir` | 文字列 | 省略できる | リポジトリの最上位からの相対パス。区切りは `/`。絶対パスと `..` は書けない。既定は `docs/ja/requirements` |

- 知らないキーがあれば、設定のエラーにする。ほかの機能 (対応づけ、サイト) がキーを足すときは、同じ構造体に足す。
- カテゴリの名前とディレクトリの名前が合っているかは、確かめない。`CTR-CHK-001` の表にないためである。

## このリポジトリの設定

このリポジトリも、最上位に自分の設定ファイル [.cumin-trace.toml](../../../../.cumin-trace.toml) を持つ。接頭辞は `CTR`、カテゴリは [要件の一覧](../../requirements/index.md) の表の7つで、同じ順に並べる。`requirements_dir` は既定のままなので書かない。`cmd/cumin-trace/main_test.go` の `TestTheDocumentsOfThisRepositoryPassTheCheck` が、このリポジトリの要件文書に書式検査を実行し、エラーがないことを確かめる。

## 確かめたこと

- 版: `github.com/BurntSushi/toml` v1.6.0。cumin-works の `go.mod` と同じ版である。
- 公式ページ: <https://pkg.go.dev/github.com/BurntSushi/toml> (v1.6.0、2025-12-18 公開)。`Decode`、`MetaData.Undecoded`、`MetaData.IsDefined`、`ParseError` の定義を読んだ。
- 対応する TOML の版は、README では v1.1.0、概要では v1.0.0 と書いてある。上の3つのキーは、どちらの版でも同じ書き方である。
- 標準ライブラリ以外に依存せず (`go list -deps`)、MIT ライセンスである。Go 1.27.2 の `go list std` に、TOML のパーサはない。
- 手元の実行で: 知らないキーは `MetaData.Undecoded`、構文エラーの行番号は `ParseError.Position.Line` で取れる。
- 未確認: `github.com/pelletier/go-toml/v2` の公式ページ。cumin-works と同じライブラリで足りるので、比べていない。
