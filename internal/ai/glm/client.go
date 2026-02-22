package glm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Lin-Jiong-HDU/tada/internal/ai"
)

const (
	// GLM API uses a different endpoint
	defaultAPIBaseURL = "https://open.bigmodel.cn/api"

	defaultSystemPrompt = `You are tada, a terminal AI assistant. Your job is to understand user requests and convert them into shell commands.

Rules:
1. Return ONLY valid JSON
2. For simple requests, return a single command
3. Explain your reasoning in the "reason" field
4. Mark dangerous commands (rm, chmod, etc.) with needs_confirm: true

Response format:
{
  "commands": [{"cmd": "command", "args": ["arg1", "arg2"]}],
  "reason": "explanation",
  "needs_confirm": false
}`
)

// Client implements AIProvider for GLM (Zhipu AI)
type Client struct {
	apiKey  string
	model   string
	baseURL string
}

// NewClient creates a new GLM client
func NewClient(apiKey, model, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultAPIBaseURL
	}
	return &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

// ParseIntent parses user input and returns intent
func (c *Client) ParseIntent(ctx context.Context, input string, systemPrompt string) (*ai.Intent, error) {
	if systemPrompt == "" {
		systemPrompt = defaultSystemPrompt
	}

	prompt := fmt.Sprintf("User request: %s\n\nConvert this to shell commands. Return JSON only.", input)

	response, err := c.callAPI(ctx, []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, err
	}

	return c.parseIntentResponse(response)
}

// AnalyzeOutput analyzes command output
func (c *Client) AnalyzeOutput(ctx context.Context, cmd string, output string) (string, error) {
	prompt := fmt.Sprintf("Command: %s\nOutput:\n%s\n\nBriefly explain what happened (max 2 sentences).", cmd, output)

	response, err := c.callAPI(ctx, []ai.Message{
		{Role: "system", Content: "You are a helpful assistant. Be brief and clear."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return "", err
	}

	return response, nil
}

// Chat handles general conversation
func (c *Client) Chat(ctx context.Context, messages []ai.Message) (string, error) {
	return c.callAPI(ctx, messages)
}

// callAPI makes the actual API call to GLM
func (c *Client) callAPI(ctx context.Context, messages []ai.Message) (string, error) {
	// Build request body for GLM API
	reqBody := map[string]interface{}{
		"model":    c.model,
		"messages": messages,
		"stream":   false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// GLM uses /paas/v4/chat/completions endpoint
	url := c.baseURL + "/paas/v4/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse GLM response format
	var respData struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(respData.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	// Check for sensitive content filter
	if respData.Choices[0].FinishReason == "sensitive" {
		return "", fmt.Errorf("content was filtered by safety check")
	}

	return respData.Choices[0].Message.Content, nil
}

// parseIntentResponse parses JSON response into Intent
func (c *Client) parseIntentResponse(response string) (*ai.Intent, error) {
	// Clean response - remove markdown code blocks if present
	cleaned := cleanJSONResponse(response)

	var intent ai.Intent
	if err := json.Unmarshal([]byte(cleaned), &intent); err != nil {
		return nil, fmt.Errorf("failed to parse intent: %w", err)
	}
	return &intent, nil
}

// cleanJSONResponse removes markdown code blocks from AI responses
func cleanJSONResponse(response string) string {
	// Remove ```json and ``` markers
	trimmed := response

	// Check for ```json at start
	if len(trimmed) > 7 && trimmed[:7] == "```json" {
		trimmed = trimmed[7:]
	} else if len(trimmed) > 3 && trimmed[:3] == "```" {
		trimmed = trimmed[3:]
	}

	// Check for ``` at end
	if len(trimmed) > 3 && trimmed[len(trimmed)-3:] == "```" {
		trimmed = trimmed[:len(trimmed)-3]
	}

	return trimmed
}

// ChatStream 流式对话
func (c *Client) ChatStream(ctx context.Context, messages []ai.Message) (<-chan string, error) {
	reqBody := map[string]interface{}{
		"model":    c.model,
		"messages": messages,
		"stream":   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/paas/v4/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	ch := make(chan string)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// SSE 格式: "data: {...}"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) > 0 {
				content := chunk.Choices[0].Delta.Content
				ch <- content
			}
		}
	}()

	return ch, nil
}
