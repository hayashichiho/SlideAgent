package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type rpcMsg map[string]any

func writeLine(w *bufio.Writer, v any) error {
	// RPC メッセージを 1 行 JSON で送信する。
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.WriteString(string(b) + "\n")
	if err != nil {
		return err
	}
	return w.Flush()
}

func readLine(r *bufio.Reader) (rpcMsg, error) {
	// 1 行 JSON の RPC レスポンスを読み込む。
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
	userText, err := in.ReadString('\n')
	if err != nil {
		fmt.Println("stdin read error:", err)
		return
	}

	// Gemini API 呼び出し
	slidesJSON, err := callGeminiJSON(userText)
	if err != nil {
		fmt.Println("Gemini error:", err)
		return
	}

	// MCP サーバを子プロセスで起動する。
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = "../mcp-server"
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("failed to connect mcp stdin:", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("failed to connect mcp stdout:", err)
		return
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Println("failed to start mcp-server:", err)
		return
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

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

	if err := writeLine(w, initReq); err != nil {
		fmt.Println("initialize write error:", err)
		return
	}
	initResp, err := readLine(r)
	if err != nil {
		fmt.Println("initialize read error:", err)
		return
	}
	if _, hasErr := initResp["error"]; hasErr {
		fmt.Println("initialize returned error:", initResp["error"])
		return
	}

	// 初期化完了通知
	if err := writeLine(w, rpcMsg{"jsonrpc": "2.0", "method": "notifications/initialized"}); err != nil {
		fmt.Println("initialized notification write error:", err)
		return
	}

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
	if err := writeLine(w, callReq); err != nil {
		fmt.Println("tools/call write error:", err)
		return
	}
	resp, err := readLine(r)
	if err != nil {
		fmt.Println("tools/call error:", err)
		return
	}
	if _, hasErr := resp["error"]; hasErr {
		fmt.Println("tools/call returned error:", resp["error"])
		return
	}
	if _, ok := resp["result"]; !ok {
		fmt.Println("tools/call invalid response:", errors.New("missing result"))
		return
	}

	fmt.Println("MCP response:", resp)
	fmt.Println("Check ./output for the generated .pptx")
}
