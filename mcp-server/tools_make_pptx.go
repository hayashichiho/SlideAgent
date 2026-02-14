package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
)

func toolMakePptxDef() map[string]any {
	/* make_pptx ツールの定義関数 */
	return map[string]any{
		"name":        "make_pptx",
		"description": "Build a .pptx deck from slides_json using the local pptx-engine (PptxGenJS). Output is saved under ./outputs.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"slides_json": map[string]any{"type": "object", "description": "Deck spec: {title, subtitle, slides:[{title, bullets, imagePath?}], outputName?}"},
			},
			"required": []string{"slides_json"},
		},
	}
}

func toolMakePptxCall(args map[string]any) (map[string]any, error) {
	/* pptx-engine を呼び出して .pptx を生成する関数 */
	// 引数から slides_json を取得して JSON にシリアライズ
	raw, ok := args["slides_json"]
	if !ok {
		return nil, errors.New("missing slides_json")
	}
	inBytes, _ := json.Marshal(raw)

	// Node.js の pptx-engine スクリプトを呼び出す
	cmd := exec.Command("node", "pptx-engine/src/build_pptx.mjs")
	cmd.Stdin = bytes.NewReader(inBytes)

	// コマンドの標準出力と標準エラーをキャプチャ
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, errors.New(stderr.String())
	}

	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, err
	}

	// MCP tool result format: content[] を返すのが一般的
	return map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": "pptx generated"},
			map[string]any{"type": "text", "text": out["pptx_path"]},
		},
		"structuredContent": out,
	}, nil
}
