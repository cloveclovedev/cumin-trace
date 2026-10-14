# 全体の構成

cumin-trace のコードの地図である。機能ごとの設計文書は、この図のどこかにあたる。

![cumin-trace のパッケージ](overview.svg)

図の元ファイル: [overview.puml](overview.puml)

| パッケージ | 役割 | 設計 |
|---|---|---|
| `cmd/cumin-trace` | 引数からコマンドを選び、結果を終了コードにする | なし |
| `internal/check` | 書式検査 | [書式検査](../designs/check/index.md) |
| `internal/requirement` | 要件文書を読み、要件のモデルにする。書式検査、出力、対応づけが共有する | [書式検査](../designs/check/index.md) |
| `internal/core` | `.cumin-trace.toml` を読む | [書式検査](../designs/check/index.md) |
