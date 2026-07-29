package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"hp/internal/model"
	"hp/internal/ollama"
	"hp/internal/store"
)

type ProposedTask struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    int      `json:"priority"`
	DueDate     string   `json:"due_date"`
	ActionItems []string `json:"action_items"`
	Rationale   string   `json:"rationale"`
}

type PlanResult struct {
	Tasks               []ProposedTask `json:"tasks"`
	ClarifyingQuestions []string       `json:"clarifying_questions"`
}

var planSchema = json.RawMessage(`{
  "type": "object",
  "properties": {
    "tasks": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "title": {"type": "string"},
          "description": {"type": "string"},
          "priority": {"type": "integer"},
          "due_date": {"type": "string"},
          "action_items": {"type": "array", "items": {"type": "string"}},
          "rationale": {"type": "string"}
        },
        "required": ["title", "priority", "rationale"]
      }
    },
    "clarifying_questions": {"type": "array", "items": {"type": "string"}}
  },
  "required": ["tasks"]
}`)

const planSystemPrompt = `You are a careful planning assistant helping a person break a long-horizon goal into concrete subtasks. Only propose subtasks that are genuinely useful and specific to their situation and background - do not pad the list with generic filler. Priority is 1 (highest) to 5 (lowest). Use YYYY-MM-DD for due_date only when there's a clear reason to set one, otherwise leave it empty. If missing information would change your plan, list it in clarifying_questions instead of guessing.`

// Plan asks the model to propose subtasks for goal, given the vault's
// context docs and the goal's existing tasks (so it doesn't propose
// duplicates).
func Plan(ctx context.Context, client *ollama.Client, s *store.Store, goal *model.Goal) (*PlanResult, error) {
	bundle, err := BuildContextBundle(s)
	if err != nil {
		return nil, err
	}
	allTasks, err := s.ListTasks()
	if err != nil {
		return nil, err
	}
	existing := store.FilterTasks(allTasks, store.TaskFilter{GoalID: goal.ID})

	user := fmt.Sprintf(
		"BACKGROUND CONTEXT:\n%s\n\n%s\n\nEXISTING TASKS FOR THIS GOAL (do not duplicate these):\n%s\n\nPropose new subtasks to make progress on this goal.",
		bundle, describeGoal(goal), describeTasks(existing),
	)

	var result PlanResult
	if err := client.ChatJSON(ctx, planSystemPrompt, user, planSchema, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// NewTaskFromProposal converts a proposal into a model.Task ready for
// store.CreateTask, linked to goal (and parent, if this proposal becomes a
// subtask of an existing task).
func NewTaskFromProposal(p ProposedTask, goalID, parentID string) *model.Task {
	body := p.Description
	if len(p.ActionItems) > 0 {
		if body != "" {
			body += "\n\n"
		}
		body += "Action items:\n"
		for _, item := range p.ActionItems {
			body += "- [ ] " + item + "\n"
		}
	}
	if p.Rationale != "" {
		body += "\n(AI rationale: " + p.Rationale + ")\n"
	}
	priority := p.Priority
	if priority < model.PriorityHighest || priority > model.PriorityLowest {
		priority = 3
	}
	return &model.Task{
		Title:    p.Title,
		GoalID:   goalID,
		ParentID: parentID,
		Priority: priority,
		Due:      p.DueDate,
		Source:   model.SourceAI,
		Body:     body,
	}
}
