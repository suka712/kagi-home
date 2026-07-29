// Package ai builds prompts from the vault's contents, calls the local
// Ollama model with a JSON schema, and returns typed proposals. Nothing in
// this package writes to the vault — that's internal/apply's job, once the
// user has reviewed and confirmed a proposal.
package ai

import (
	"fmt"
	"strings"

	"hp/internal/model"
	"hp/internal/store"
)

// BuildContextBundle concatenates all context docs into one text block for
// inclusion in a prompt. Kept as plain concatenation (no chunking or
// embeddings) — fine for a personal-scale set of background docs within
// gemma4's context window; revisit only if that stops being true.
func BuildContextBundle(s *store.Store) (string, error) {
	docs, err := s.ListContextDocs()
	if err != nil {
		return "", err
	}
	if len(docs) == 0 {
		return "(no background context provided yet)", nil
	}
	var b strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&b, "## %s\n%s\n\n", d.Title, d.Body)
	}
	return b.String(), nil
}

func describeGoal(g *model.Goal) string {
	return fmt.Sprintf("Goal: %s\nStatus: %s\nTarget date: %s\nDescription: %s",
		g.Title, g.Status, orNone(g.TargetDate), orNone(g.Body))
}

func describeTasks(tasks []*model.Task) string {
	if len(tasks) == 0 {
		return "(none yet)"
	}
	var b strings.Builder
	for _, t := range tasks {
		fmt.Fprintf(&b, "- [%s] p%d %s (due %s)\n", t.Status, t.Priority, t.Title, orNone(t.Due))
	}
	return b.String()
}

// describeTasksWithID is like describeTasks but includes the full task ID
// so the model can reference a specific task unambiguously.
func describeTasksWithID(tasks []*model.Task) string {
	if len(tasks) == 0 {
		return "(none)"
	}
	var b strings.Builder
	for _, t := range tasks {
		fmt.Fprintf(&b, "- id=%s [%s] p%d %s (due %s)\n", t.ID, t.Status, t.Priority, t.Title, orNone(t.Due))
	}
	return b.String()
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}
