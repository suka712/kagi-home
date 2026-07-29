package apply

import (
	"fmt"
	"strings"

	"github.com/fatih/color"

	"hp/internal/ai"
	"hp/internal/model"
)

var (
	addColor    = color.New(color.FgGreen)
	changeColor = color.New(color.FgYellow)
)

func RenderNewTask(p ai.ProposedTask) string {
	var b strings.Builder
	addColor.Fprintf(&b, "+ new task: %s  (priority %d", p.Title, p.Priority)
	if p.DueDate != "" {
		fmt.Fprintf(&b, ", due %s", p.DueDate)
	}
	fmt.Fprint(&b, ")\n")
	if p.Description != "" {
		fmt.Fprintf(&b, "  %s\n", p.Description)
	}
	for _, item := range p.ActionItems {
		fmt.Fprintf(&b, "    - [ ] %s\n", item)
	}
	if p.Rationale != "" {
		fmt.Fprintf(&b, "  rationale: %s\n", p.Rationale)
	}
	return b.String()
}

func RenderPriorityChange(t *model.Task, c ai.PriorityChange) string {
	var b strings.Builder
	changeColor.Fprintf(&b, "~ %s: priority %d -> %d\n", t.Title, t.Priority, c.NewPriority)
	if c.Rationale != "" {
		fmt.Fprintf(&b, "  rationale: %s\n", c.Rationale)
	}
	return b.String()
}

func RenderActionItem(item string) string {
	return addColor.Sprintf("+ action item: %s\n", item)
}
