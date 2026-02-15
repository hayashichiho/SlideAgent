# SlideAgent

## 追加機能: Template Profile Check (MVP)

`assets/template.pptx` を型として学習し、ユーザーの `.pptx` が型に沿っているかを解析します。

- `pptx_extract_style`: PPTX(OOXML) を解析して metrics JSON を生成
- `profile_build`: template metrics + 手動ルールから profile JSON を生成
- `profile_check`: profile と deck metrics を比較し、score + violations を生成
- `generate_coaching_tips`: violations を元に Gemini で日本語の改善コツを生成

すべての結果JSONは `outputs/` に保存します。

## MCP tools

### 1) `pptx_extract_style`
Input:
```json
{ "pptx_path": "assets/template.pptx" }
```
Output (例):
```json
{ "metrics_path": "../outputs/metrics_....json", "slide_count": 10 }
```

### 2) `profile_build`
Input:
```json
{
  "template_metrics_path": "../outputs/template_metrics.json",
  "rules_text": "SlideAgent Template Rules (v1) ..."
}
```

### 3) `profile_check`
Input:
```json
{
  "profile_path": "../outputs/profile.json",
  "deck_metrics_path": "../outputs/deck_metrics.json"
}
```
Output:
```json
{ "score": 82, "violation_count": 12, "violations_path": "../outputs/violations_....json" }
```

### 4) `generate_coaching_tips`
Input:
```json
{
  "template_profile": {"...": "..."},
  "violations": {"score": 82, "violations": []}
}
```

## 最短実行手順

1. 依存セットアップ
```bash
make setup
```

2. テンプレから profile 作成
```bash
make profile
```

3. ユーザーPPTXをチェック
```bash
make check DECK=decks/review/your_deck.pptx
```

4. Geminiで改善コツを生成
```bash
GEMINI_API_KEY=... make coach
```

生成物:
- `outputs/template_metrics.json`
- `outputs/profile.json`
- `outputs/deck_metrics.json`
- `outputs/violations.json`
- `outputs/tips.json`

## 添削用PPTXの置き場

- 入力用フォルダ: `decks/review/`
- 例: `decks/review/my_slides.pptx`
- 実行: `make check DECK=decks/review/my_slides.pptx`

## Gemini でコツ生成（MCP経由）

環境変数:
- `GEMINI_API_KEY`
- `GEMINI_MODEL` (任意, 既定: `gemini-2.5-flash`)

`generate_coaching_tips` に `template_profile` と `violations` を渡すと `outputs/tips_*.json` が生成されます。

## JSON-RPC 呼び出し例（tools/call）

```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pptx_extract_style","arguments":{"pptx_path":"assets/template.pptx"}}}
```

```json
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"profile_build","arguments":{"template_metrics_path":"../outputs/template_metrics.json","rules_text":"SlideAgent Template Rules (v1) ..."}}}
```

```json
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"profile_check","arguments":{"profile_path":"../outputs/profile.json","deck_metrics_path":"../outputs/deck_metrics.json"}}}
```

## rules_text

`assets/rules_text.txt` を利用:
- Top title bar + bold title
- Keep consistent spacing and alignment across slides
- Structure: bold subheading -> body text
- Purpose highlight in red
- Issues highlighted with blue outline box
- Visual flow: eye moves top-to-bottom
- Font sizes: caption 18pt, body 22pt, subheading 24pt
- Fonts: Japanese Meiryo UI, Latin Segoe UI
- First subheading/body positions are consistent across slides
- Bottom/right alignment consistent across slides when possible
- Body text <= 2 lines
- Caption below images; caption above tables
- Put slide number at top-right in the title bar
- Avoid text-only slides; include figures/tables

## 注意

- `assets/template.pptx` は必須です（学習元）。
- 解析はMVPとして、明示XML情報（`rPr`, `xfrm` など）を優先し、theme/master継承は後回しです。
