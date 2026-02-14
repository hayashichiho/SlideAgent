package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type geminiResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func callGeminiJSON(userText string) (map[string]any, error) {
	/* Gemini API を呼び出して、ユーザの入力に基づいてスライドの内容を JSON 形式で生成する関数 */
	// GEMINI_API_KEY 環境変数から API キーを取得
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return nil, errors.New("GEMINI_API_KEY is empty")
	}

	// モデルは環境変数 GEMINI_MODEL から取得（指定がなければ "gemini-2.5-flash" を使用）
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.5-flash"
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)

	// リクエストボディの構築
	body := map[string]any{
		"contents": []any{
			map[string]any{
				"parts": []any{
					map[string]any{"text": SystemPrompt + "\n\nUSER_INPUT:\n" + userText},
				},
			},
		},
	}

	// HTTP POST リクエストの送信
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", key)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// レスポンスの処理
	var gr geminiResp
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, err
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("empty gemini response")
	}

	// レスポンスから JSON を抽出して返す
	raw := gr.Candidates[0].Content.Parts[0].Text
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w\nraw:\n%s", err, raw)
	}
	return out, nil
}
