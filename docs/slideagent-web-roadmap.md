# SlideAgent Roadmap (Current)

このドキュメントは現行の最小構成を示します。
旧構成（`client/`, `services/api`, `services/worker`, `services/frontend`, `make_pptx` 系ツール）は削除済みです。

## 現在の目的
- 参考スライド（`assets/template.pptx`）から型を抽出
- 型（profile）を生成
- 対象スライドを型に照らして検査
- 違反点から改善コツを日本語で生成

## 現在の構成
- `mcp-server/`
  - MCP stdio サーバ
  - 利用ツール:
    - `pptx_extract_style`
    - `profile_build`
    - `profile_check`
    - `generate_coaching_tips`
- `pptx-engine/src/`
  - `pptx_extract_style.mjs`
  - `profile_build.mjs`
  - `profile_check.mjs`
  - `generate_tips.mjs`
- `assets/`
  - `template.pptx`
  - `rules_text.txt`
- `outputs/`
  - 生成物JSON（`template_metrics.json`, `profile.json`, `deck_metrics.json`, `violations.json`, `tips.json`）

## 標準フロー
1. `make profile`
2. `make check DECK=outputs/your_deck.pptx`
3. `make coach`

## 備考
- 出力先は `outputs/` に統一。
- 実行方法と入出力の詳細は `README.md` を正とする。
