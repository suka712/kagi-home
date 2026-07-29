package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"hp/internal/model"
	"hp/internal/ollama"
	"hp/internal/store"
)

type AskResult struct {
	Questions []string `json:"questions"`
}

var askSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "questions": {"type": "array", "items": {"type": "string"}}
  },
  "required": ["questions"]
}`)

const askSystemPrompt = `You help a person plan toward their goals by identifying missing information. Given their background context and a goal, ask up to 5 short, specific clarifying questions whose answers would materially change how you'd plan or prioritize their tasks (e.g. exact deadlines, constraints, which goal matters more if they conflict). Do not ask about anything already answered in the context. Keep each question to one sentence. If nothing important is missing, return an empty list.`

// Ask asks the model what clarifying questions it would want answered
// before planning/prioritizing for goal.
func Ask(ctx context.Context, client *ollama.Client, s *store.Store, goal *model.Goal) (*AskResult, error) {
	bundle, err := BuildContextBundle(s)
	if err != nil {
		return nil, err
	}
	user := fmt.Sprintf("BACKGROUND CONTEXT:\n%s\n\n%s\n\nWhat clarifying questions would help you plan for this goal?", bundle, describeGoal(goal))

	var result AskResult
	if err := client.ChatJSON(ctx, askSystemPrompt, user, askSchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
