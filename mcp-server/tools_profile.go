package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func ensureOutputsDir() (string, error) {
	dir := filepath.Join("..", "outputs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func runNodeJSON(script string, in map[string]any) (map[string]any, error) {
	inBytes, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("node", script)
	cmd.Stdin = bytes.NewReader(inBytes)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("node tool failed: %s", msg)
	}
	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("invalid node output: %w", err)
	}
	return out, nil
}

func toolPptxExtractStyleDef() map[string]any {
	return map[string]any{
		"name":        "pptx_extract_style",
		"description": "Extract textRuns/shapes/images metrics from a .pptx (OOXML) and save JSON under outputs/.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pptx_path":   map[string]any{"type": "string"},
				"output_path": map[string]any{"type": "string"},
			},
			"required": []string{"pptx_path"},
		},
	}
}

func toolPptxExtractStyleCall(args map[string]any) (map[string]any, error) {
	pptxPath, ok := args["pptx_path"].(string)
	if !ok || strings.TrimSpace(pptxPath) == "" {
		return nil, errors.New("missing pptx_path")
	}
	in := map[string]any{"pptx_path": pptxPath}
	if outPath, ok := args["output_path"].(string); ok && strings.TrimSpace(outPath) != "" {
		in["output_path"] = outPath
	} else {
		outDir, err := ensureOutputsDir()
		if err != nil {
			return nil, err
		}
		in["output_path"] = filepath.Join(outDir, fmt.Sprintf("metrics_%d.json", time.Now().Unix()))
	}

	out, err := runNodeJSON("../pptx-engine/src/pptx_extract_style.mjs", in)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": "pptx metrics extracted"},
			map[string]any{"type": "text", "text": out["metrics_path"]},
		},
		"structuredContent": out,
	}, nil
}

func toolProfileBuildDef() map[string]any {
	return map[string]any{
		"name":        "profile_build",
		"description": "Build style profile from template metrics and manual rules text.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"template_metrics":      map[string]any{"type": "object"},
				"template_metrics_path": map[string]any{"type": "string"},
				"rules_text":            map[string]any{"type": "string"},
				"output_path":           map[string]any{"type": "string"},
			},
		},
	}
}

func toolProfileBuildCall(args map[string]any) (map[string]any, error) {
	in := map[string]any{}
	if v, ok := args["template_metrics"]; ok {
		in["template_metrics"] = v
	}
	if v, ok := args["template_metrics_path"]; ok {
		in["template_metrics_path"] = v
	}
	if _, ok := in["template_metrics"]; !ok {
		if _, ok := in["template_metrics_path"]; !ok {
			return nil, errors.New("missing template_metrics or template_metrics_path")
		}
	}
	if v, ok := args["rules_text"]; ok {
		in["rules_text"] = v
	}

	if outPath, ok := args["output_path"].(string); ok && strings.TrimSpace(outPath) != "" {
		in["output_path"] = outPath
	} else {
		outDir, err := ensureOutputsDir()
		if err != nil {
			return nil, err
		}
		in["output_path"] = filepath.Join(outDir, fmt.Sprintf("profile_%d.json", time.Now().Unix()))
	}

	out, err := runNodeJSON("../pptx-engine/src/profile_build.mjs", in)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": "profile built"},
			map[string]any{"type": "text", "text": out["profile_path"]},
		},
		"structuredContent": out,
	}, nil
}

func toolProfileCheckDef() map[string]any {
	return map[string]any{
		"name":        "profile_check",
		"description": "Check deck metrics against profile and emit violations with score.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"profile":           map[string]any{"type": "object"},
				"profile_path":      map[string]any{"type": "string"},
				"deck_metrics":      map[string]any{"type": "object"},
				"deck_metrics_path": map[string]any{"type": "string"},
				"output_path":       map[string]any{"type": "string"},
			},
		},
	}
}

func toolProfileCheckCall(args map[string]any) (map[string]any, error) {
	in := map[string]any{}
	if v, ok := args["profile"]; ok {
		in["profile"] = v
	}
	if v, ok := args["profile_path"]; ok {
		in["profile_path"] = v
	}
	if v, ok := args["deck_metrics"]; ok {
		in["deck_metrics"] = v
	}
	if v, ok := args["deck_metrics_path"]; ok {
		in["deck_metrics_path"] = v
	}
	if _, ok := in["profile"]; !ok {
		if _, ok := in["profile_path"]; !ok {
			return nil, errors.New("missing profile or profile_path")
		}
	}
	if _, ok := in["deck_metrics"]; !ok {
		if _, ok := in["deck_metrics_path"]; !ok {
			return nil, errors.New("missing deck_metrics or deck_metrics_path")
		}
	}

	if outPath, ok := args["output_path"].(string); ok && strings.TrimSpace(outPath) != "" {
		in["output_path"] = outPath
	} else {
		outDir, err := ensureOutputsDir()
		if err != nil {
			return nil, err
		}
		in["output_path"] = filepath.Join(outDir, fmt.Sprintf("violations_%d.json", time.Now().Unix()))
	}

	out, err := runNodeJSON("../pptx-engine/src/profile_check.mjs", in)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": "profile check completed"},
			map[string]any{"type": "text", "text": out["violations_path"]},
		},
		"structuredContent": out,
	}, nil
}

func toolGenerateCoachingTipsDef() map[string]any {
	return map[string]any{
		"name":        "generate_coaching_tips",
		"description": "Generate Japanese coaching tips from profile and violations via Gemini.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"template_profile": map[string]any{"type": "object"},
				"violations":       map[string]any{"type": "object"},
				"output_path":      map[string]any{"type": "string"},
			},
			"required": []string{"template_profile", "violations"},
		},
	}
}

func toolGenerateCoachingTipsCall(args map[string]any) (map[string]any, error) {
	templateProfile, ok := args["template_profile"]
	if !ok {
		return nil, errors.New("missing template_profile")
	}
	violations, ok := args["violations"]
	if !ok {
		return nil, errors.New("missing violations")
	}

	promptHeader := `あなたは「スライド設計コーチ」です。ユーザーはハッカソン/研究発表のスライドを改善したいが、単なる修正案ではなく“原理（コツ）”を学びたい。

入力:
- template_profile: 型のルールと統計
- violations: 型からのズレ一覧（evidence付き）

出力（日本語）:
1) 総合スコア（100点満点）と一言講評（15〜25字）
2) 重要度A（最大3件）：各項目を「なぜ重要か（原理）」→「今の状態（evidence）」→「直し方（具体）」の順で説明
3) 重要度B（最大3件）：同様に短く
4) “型の要点まとめ”を5行で（暗記できる形）
5) 次に見るべきチェックリスト（7項目）

制約:
- 断定しすぎず、evidenceに基づく
- “直し方”はユーザーが手で直せる粒度（位置/フォント/行数など）
- 箇条書き中心。必要な専門用語は一言で補足。`

	tpBytes, _ := json.MarshalIndent(templateProfile, "", "  ")
	vBytes, _ := json.MarshalIndent(violations, "", "  ")
	prompt := promptHeader + "\n\n" + "template_profile:\n" + string(tpBytes) + "\n\nviolations:\n" + string(vBytes)

	key := os.Getenv("GEMINI_API_KEY")
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("GEMINI_API_KEY is empty")
	}
	model := os.Getenv("GEMINI_MODEL")
	if strings.TrimSpace(model) == "" {
		model = "gemini-2.5-flash"
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)

	body := map[string]any{
		"contents": []any{map[string]any{"parts": []any{map[string]any{"text": prompt}}}},
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", key)
	resp, err := (&http.Client{Timeout: 45 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini api error: status=%d body=%s", resp.StatusCode, string(raw))
	}

	var gr struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, err
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("empty gemini response")
	}
	tipsText := strings.TrimSpace(gr.Candidates[0].Content.Parts[0].Text)

	outPath, _ := args["output_path"].(string)
	if strings.TrimSpace(outPath) == "" {
		outDir, err := ensureOutputsDir()
		if err != nil {
			return nil, err
		}
		outPath = filepath.Join(outDir, fmt.Sprintf("tips_%d.json", time.Now().Unix()))
	}
	payload := map[string]any{
		"generatedAt": time.Now().UTC().Format(time.RFC3339),
		"tips":        tipsText,
	}
	ob, _ := json.MarshalIndent(payload, "", "  ")
	if err := os.WriteFile(outPath, ob, 0o644); err != nil {
		return nil, err
	}

	return map[string]any{
		"content": []any{
			map[string]any{"type": "text", "text": "coaching tips generated"},
			map[string]any{"type": "text", "text": outPath},
		},
		"structuredContent": map[string]any{
			"tips_path": outPath,
			"tips":      tipsText,
		},
	}, nil
}
