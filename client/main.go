package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

func runPipeline(userText string) (string, rpcMsg, error) {
	// Gemini API 呼び出し
	slidesJSON, err := callGeminiJSON(userText)
	if err != nil {
		return "", nil, fmt.Errorf("gemini error: %w", err)
	}

	// MCP サーバを子プロセスで起動する。
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = "../mcp-server"
	cmd.Env = os.Environ()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", nil, fmt.Errorf("failed to connect mcp stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", nil, fmt.Errorf("failed to connect mcp stdout: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", nil, fmt.Errorf("failed to start mcp-server: %w", err)
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
		return "", nil, fmt.Errorf("initialize write error: %w", err)
	}
	initResp, err := readLine(r)
	if err != nil {
		return "", nil, fmt.Errorf("initialize read error: %w", err)
	}
	if _, hasErr := initResp["error"]; hasErr {
		return "", nil, fmt.Errorf("initialize returned error: %v", initResp["error"])
	}

	// 初期化完了通知
	if err := writeLine(w, rpcMsg{"jsonrpc": "2.0", "method": "notifications/initialized"}); err != nil {
		return "", nil, fmt.Errorf("initialized notification write error: %w", err)
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
		return "", nil, fmt.Errorf("tools/call write error: %w", err)
	}
	resp, err := readLine(r)
	if err != nil {
		return "", nil, fmt.Errorf("tools/call error: %w", err)
	}
	if _, hasErr := resp["error"]; hasErr {
		return "", nil, fmt.Errorf("tools/call returned error: %v", resp["error"])
	}
	if _, ok := resp["result"]; !ok {
		return "", nil, fmt.Errorf("tools/call invalid response: %w", errors.New("missing result"))
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		return "", nil, errors.New("tools/call result is not an object")
	}
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		return "", nil, errors.New("structuredContent is missing")
	}
	pptxPath, ok := structured["pptx_path"].(string)
	if !ok || strings.TrimSpace(pptxPath) == "" {
		return "", nil, errors.New("pptx_path is missing")
	}

	return pptxPath, resp, nil
}

func main() {
	pitch := flag.String("pitch", "", "Pitch text for non-interactive mode")
	jsonOut := flag.Bool("json", false, "Print result as JSON")
	flag.Parse()

	userText := strings.TrimSpace(*pitch)
	if userText == "" {
		fmt.Println("SlideAgent (MVP) — paste your pitch in one paragraph, then Enter:")
		in := bufio.NewReader(os.Stdin)
		line, err := in.ReadString('\n')
		if err != nil {
			fmt.Println("stdin read error:", err)
			return
		}
		userText = strings.TrimSpace(line)
	}

	pptxPath, resp, err := runPipeline(userText)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if *jsonOut {
		out := map[string]any{
			"status":   "succeeded",
			"pptxPath": pptxPath,
		}
		b, _ := json.Marshal(out)
		fmt.Println(string(b))
		return
	}

	fmt.Println("MCP response:", resp)
	fmt.Println("Check ./output for the generated .pptx")
}
