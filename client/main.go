package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type rpcMsg map[string]any

func writeLine(w *bufio.Writer, v any) error {
	/* RPC メッセージを JSON 形式でシリアライズして書き込む関数 */
	b, _ := json.Marshal(v)
	_, err := w.WriteString(string(b) + "\n")
	if err != nil {
		return err
	}
	return w.Flush()
}

func readLine(r *bufio.Reader) (rpcMsg, error) {
	/* RPC メッセージを JSON 形式で読み込む関数 */
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	var m rpcMsg
	if err := json.Unmarshal(line, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func main() {
	// ユーザからピッチの入力を受け取る
	fmt.Println("SlideAgent (MVP) — paste your pitch in one paragraph, then Enter:")
	in := bufio.NewReader(os.Stdin)
	userText, _ := in.ReadString('\n')

	// Gemini API 呼び出し
	slidesJSON, err := callGeminiJSON(userText)
	if err != nil {
		fmt.Println("Gemini error:", err)
		return
	}

	// MCPサーバーの起動
	cmd := exec.Command("go", "run", ".", "../mcp-server")
	cmd = exec.Command("go", "run", "./mcp-server")

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Println("failed to start mcp-server:", err)
		return
	}
	defer cmd.Process.Kill()

	w := bufio.NewWriter(stdin)
	r := bufio.NewReader(stdout)

	// JSON-RPCの初期化
	initReq := rpcMsg{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "slideagent-client",
				"version": "0.1.0",
			},
		},
	}

	_ = writeLine(w, initReq)
	_, _ = readLine(r)

	// 初期化完了通知
	_ = writeLine(w, rpcMsg{"jsonrpc": "2.0", "method": "notifications/initialized"})

	outName := fmt.Sprintf("pitch_%d.pptx", time.Now().Unix())
	slidesJSON["outputName"] = outName

	callReq := rpcMsg{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "make_pptx",
			"arguments": map[string]any{
				"slides_json": slidesJSON,
			},
		},
	}
	_ = writeLine(w, callReq)
	resp, err := readLine(r)
	if err != nil {
		fmt.Println("tools/call error:", err)
		return
	}

	fmt.Println("MCP response:", resp)
	fmt.Println("Check ./outputs for the generated .pptx")
}
