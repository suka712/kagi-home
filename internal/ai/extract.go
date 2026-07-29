package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"hp/internal/ollama"
	"hp/internal/store"
)

type ExtractResult struct {
	SuggestedTaskTitle string   `json:"suggested_task_title"`
	ActionItems        []string `json:"action_items"`
}

var extractSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "suggested_task_title": {"type": "string"},
    "action_items": {"type": "array", "items": {"type": "string"}}
  },
  "required": ["suggested_task_title", "action_items"]
}`)

const extractSystemPrompt = `You extract concrete, actionable to-do items from a document (notes, an email, meeting minutes, etc.), relevant to the person's goals and background. Each action item should be a short, specific, actionable line in imperative mood (e.g. "Email professor's assistant to confirm meeting time"). Do not include vague or non-actionable statements, and do not invent items not supported by the document. Also suggest a short title for a task that would hold these action items.`

// Extract asks the model to pull actionable to-do items out of docText.
func Extract(ctx context.Context, client *ollama.Client, s *store.Store, docText string) (*ExtractResult, error) {
	bundle, err := BuildContextBundle(s)
	if err != nil {
		return nil, err
	}
	user := fmt.Sprintf("BACKGROUND CONTEXT:\n%s\n\nDOCUMENT:\n%s\n\nExtract action items from the document above.", bundle, docText)

	var result ExtractResult
	if err := client.ChatJSON(ctx, extractSystemPrompt, user, extractSchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
