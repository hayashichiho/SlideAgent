package main

func toolRenderPngDef() map[string]any {
	/* render_png ツールの定義関数 */
	return map[string]any{
		"name":        "render_png",
		"description": "Render pptx into PNGs for visual review. (TODO)",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pptx_path": map[string]any{"type": "string"},
			},
			"required": []string{"pptx_path"},
		},
	}
}
