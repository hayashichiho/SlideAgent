package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type JsonRpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JsonRpcResp struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      any         `json:"id,omitempty"`
	Result  any         `json:"result,omitempty"`
	Error   *JsonRpcErr `json:"error,omitempty"`
}

type JsonRpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func reply(id any, result any) {
	resp := JsonRpcResp{JSONRPC: "2.0", ID: id, Result: result}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}

func replyErr(id any, code int, msg string) {
	resp := JsonRpcResp{JSONRPC: "2.0", ID: id, Error: &JsonRpcErr{Code: code, Message: msg}}
	b, _ := json.Marshal(resp)
	fmt.Println(string(b))
}

func main() {
	// JSON-RPC メッセージの読み書きのためのバッファリーダーとライターを標準入出力にセットアップ
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		line := in.Bytes()
		var req JsonRpcReq
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		// JSON-RPC のリクエストに応じて処理を分岐
		switch req.Method {

		// "initialize" メソッドの処理
		case "initialize":
			reply(req.ID, map[string]any{
				"protocolVersion": "2025-06-18",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    "slideagent-mcp",
					"version": "0.1.0",
				},
			})

		// "notifications/initialized" メソッドの処理（クライアントからの初期化完了通知）
		case "notifications/initialized":

		// "tools/list" メソッドの処理（利用可能なツールのリストを返す）
		case "tools/list":
			reply(req.ID, map[string]any{
				"tools": []any{
					toolMakePptxDef(),
					toolMakeDiagramDef(),
					toolRenderPngDef(), // 未実装でも定義だけ出してOK
					toolPptxExtractStyleDef(),
					toolProfileBuildDef(),
					toolProfileCheckDef(),
					toolGenerateCoachingTipsDef(),
				},
			})

		// "tools/call" メソッドの処理（ツールの呼び出し）
		case "tools/call":
			var p struct {
				Name  string         `json:"name"`
				Args  map[string]any `json:"arguments"`
				Meta  any            `json:"_meta,omitempty"`
				Extra any            `json:"extra,omitempty"`
			}

			if err := json.Unmarshal(req.Params, &p); err != nil {
				replyErr(req.ID, -32602, "invalid params")
				continue
			}

			switch p.Name {
			case "make_pptx":
				out, err := toolMakePptxCall(p.Args)
				if err != nil {
					replyErr(req.ID, -32000, err.Error())
					continue
				}
				reply(req.ID, out)

			case "make_diagram":
				out, err := toolMakeDiagramCall(p.Args)
				if err != nil {
					replyErr(req.ID, -32000, err.Error())
					continue
				}
				reply(req.ID, out)

			case "render_png":
				replyErr(req.ID, -32000, "render_png not implemented yet")

			case "pptx_extract_style":
				out, err := toolPptxExtractStyleCall(p.Args)
				if err != nil {
					replyErr(req.ID, -32000, err.Error())
					continue
				}
				reply(req.ID, out)

			case "profile_build":
				out, err := toolProfileBuildCall(p.Args)
				if err != nil {
					replyErr(req.ID, -32000, err.Error())
					continue
				}
				reply(req.ID, out)

			case "profile_check":
				out, err := toolProfileCheckCall(p.Args)
				if err != nil {
					replyErr(req.ID, -32000, err.Error())
					continue
				}
				reply(req.ID, out)

			case "generate_coaching_tips":
				out, err := toolGenerateCoachingTipsCall(p.Args)
				if err != nil {
					replyErr(req.ID, -32000, err.Error())
					continue
				}
				reply(req.ID, out)

			default:
				replyErr(req.ID, -32601, "method not found: unknown tool")
			}

		default:
			replyErr(req.ID, -32601, "method not found")
		}
	}
}
