package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"hp/internal/model"
	"hp/internal/ollama"
	"hp/internal/store"
)

type PriorityChange struct {
	TaskID      string `json:"task_id"`
	NewPriority int    `json:"new_priority"`
	Rationale   string `json:"rationale"`
}

type PrioritizeResult struct {
	Changes []PriorityChange `json:"changes"`
}

var prioritizeSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "changes": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "task_id": {"type": "string"},
          "new_priority": {"type": "integer"},
          "rationale": {"type": "string"}
        },
        "required": ["task_id", "new_priority", "rationale"]
      }
    }
  },
  "required": ["changes"]
}`)

const prioritizeSystemPrompt = `You are a careful prioritization assistant. You will be given a list of open tasks, each with an exact id, plus background context and deadlines. Decide which tasks deserve a different priority (1 highest - 5 lowest) given time constraints and goal importance, and explain why in one sentence. Only include tasks whose priority should actually change - do not restate tasks that are already correctly prioritized. Always reference tasks using the exact id given, copied verbatim.`

// Prioritize asks the model to review open tasks (optionally scoped to one
// goal) and propose priority changes. Returns the result plus the exact
// list of tasks that were sent, so callers can render before/after diffs.
func Prioritize(ctx context.Context, client *ollama.Client, s *store.Store, goalID string) (*PrioritizeResult, []*model.Task, error) {
	bundle, err := BuildContextBundle(s)
	if err != nil {
		return nil, nil, err
	}
	allTasks, err := s.ListTasks()
	if err != nil {
		return nil, nil, err
	}
	filter := store.TaskFilter{GoalID: goalID}
	open := make([]*model.Task, 0)
	for _, t := range store.FilterTasks(allTasks, filter) {
		if t.Status != model.StatusDone && t.Status != model.StatusCancelled {
			open = append(open, t)
		}
	}
	if len(open) == 0 {
		return &PrioritizeResult{}, open, nil
	}

	user := fmt.Sprintf(
		"BACKGROUND CONTEXT:\n%s\n\nOPEN TASKS:\n%s\n\nSuggest priority changes for the tasks above.",
		bundle, describeTasksWithID(open),
	)

	var result PrioritizeResult
	if err := client.ChatJSON(ctx, prioritizeSystemPrompt, user, prioritizeSchema, &result); err != nil {
		return nil, open, err
	}
	return &result, open, nil
}
