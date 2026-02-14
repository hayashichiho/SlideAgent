package main

import "errors"

func toolMakeDiagramDef() map[string]any {
	/* make_diagram ツールの定義関数 */
	return map[string]any{
		"name":        "make_diagram",
		"description": "Generate diagram assets from a spec. (TODO)",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"diagram_spec": map[string]any{"type": "object"},
			},
			"required": []string{"diagram_spec"},
		},
	}
}

func toolMakeDiagramCall(args map[string]any) (map[string]any, error) {
	/* diagram-engine を呼び出してダイアグラムを生成する関数 (TODO) */
	_ = args
	return nil, errors.New("make_diagram not implemented yet")
}
