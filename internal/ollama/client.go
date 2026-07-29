// Package ollama is a minimal client for the parts of the Ollama HTTP API
// that hp needs: chat completions with a JSON-schema response format, so
// results parse directly into typed Go structs.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	Host        string
	Model       string
	Temperature float64
	HTTPClient  *http.Client
}

func NewClient(host, model string, temperature float64) *Client {
	return &Client{
		Host:        host,
		Model:       model,
		Temperature: temperature,
		HTTPClient:  &http.Client{Timeout: 3 * time.Minute},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string          `json:"model"`
	Messages []chatMessage   `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   json.RawMessage `json:"format,omitempty"`
	Options  map[string]any  `json:"options,omitempty"`
}

type chatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Error string `json:"error"`
}

// Chat sends a system+user message pair, optionally constrained to a JSON
// schema (pass nil for freeform text), and returns the assistant's raw
// response content.
func (c *Client) Chat(ctx context.Context, system, user string, schema json.RawMessage) (string, error) {
	reqBody := chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Stream:  false,
		Format:  schema,
		Options: map[string]any{"temperature": c.Temperature},
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Host+"/api/chat", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama not reachable at %s (is `ollama serve` running?): %w", c.Host, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned %s: %s", resp.Status, string(body))
	}
	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("parsing ollama response: %w", err)
	}
	if cr.Error != "" {
		return "", fmt.Errorf("ollama error: %s", cr.Error)
	}
	return cr.Message.Content, nil
}

// ChatJSON is Chat plus unmarshaling the assistant's response into out,
// using schema both to constrain generation and describe out's shape.
func (c *Client) ChatJSON(ctx context.Context, system, user string, schema json.RawMessage, out any) error {
	content, err := c.Chat(ctx, system, user, schema)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(content), out); err != nil {
		return fmt.Errorf("model returned invalid JSON: %w\nraw response:\n%s", err, content)
	}
	return nil
}

// Ping checks that the Ollama server is reachable.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Host+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama not reachable at %s (is `ollama serve` running?): %w", c.Host, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned %s", resp.Status)
	}
	return nil
}
