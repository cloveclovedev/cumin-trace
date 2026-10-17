# 構造化データの出力

要件文書から読み取った要件を、ほかのツールや Agent が使える形で出力するための要件を定める。出力したデータは生成物であり、人は編集しない。

## 全体像

出力は、次の形の JSON になる。例は、この文書群の [対応づけ](../trace/index.md) にある、子を持つ要件とその子の1つである。キーの名前と値の形は、この例のとおりとする。

```json
{
  "requirements": [
    {
      "id": "CTR-TRC-004",
      "parent": null,
      "title": "進捗",
      "text": "子を持たない要件ごとに、進捗を次の3つのうちの1つに決める。廃止した要件には、進捗を付けない。廃止した要件は、未起票の一覧にも出ない。",
      "verifications": [],
      "status": "active",
      "replaced_by": null,
      "file": "docs/ja/requirements/trace/index.md",
      "line": 58
    },
    {
      "id": "CTR-TRC-004.1",
      "parent": "CTR-TRC-004",
      "title": "未起票",
      "text": "対応する要求Issueが1つもない要件は、未起票とする。対応する要求Issueがすべて not planned で閉じている要件も、未起票とする。",
      "verifications": [
        {"method": "test", "how": "要求Issueのない要件と、not planned で閉じた要求Issueだけを持つ要件が、未起票になる"}
      ],
      "status": "active",
      "replaced_by": null,
      "file": "docs/ja/requirements/trace/index.md",
      "line": 62
    }
  ]
}
```

- `status` は `active` か `deprecated` である。廃止した要件は `deprecated` になり、`replaced_by` に置き換え先のIDを持つ。置き換え先がなければ `null` である。
- `line` は、要件の見出しの行番号である。

## 要件

### `CTR-EXP-001` 出力する内容

`cumin-trace export` は、すべての要件を JSON で出力する。要件ごとに、ID、親のID、タイトル、本文、検証の一覧 (方法と確かめ方)、状態、置き換え先、ファイル、行番号を持つ。廃止した要件も出力する。

- 検証: `test` 親子のある要件と廃止した要件を含む文書で、出力を確かめる

---

### `CTR-EXP-002` 同じ文書からは同じ出力

同じ文書からは、常に同じ JSON を出力する。要件の順番は、設定に書いたカテゴリの順、その中では文書に書いた順とする。出力した日時のように、実行のたびに変わる値は含めない。

- 検証: `test` 2回出力して、内容が一致する

---

### `CTR-EXP-003` 書式の誤りがあるとき

書式検査 (`CTR-CHK-001`) でエラーになる文書からは、JSON を出力しない。書式検査と同じエラーを報告し、0 以外の終了コードで終わる。誤った文書から作ったデータを、サイトやほかのツールが使わないためである。

- 検証: `test` エラーを1つ含む文書で、JSON が出ず、エラーが報告され、終了コードが 0 以外になる
