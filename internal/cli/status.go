package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"hp/internal/model"
	"hp/internal/store"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Summary: overdue tasks, tasks due soon, and open tasks by goal",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd)
		},
	}
}

func runStatus(cmd *cobra.Command) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	tasks, err := s.ListTasks()
	if err != nil {
		return err
	}
	goals, err := s.ListGoals()
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()

	open := func(t *model.Task) bool {
		return t.Status != model.StatusDone && t.Status != model.StatusCancelled
	}

	var overdue, dueSoon []*model.Task
	soonCutoff := time.Now().Truncate(24*time.Hour).AddDate(0, 0, 7)
	for _, t := range tasks {
		if !open(t) || t.Due == "" {
			continue
		}
		d, err := time.Parse("2006-01-02", t.Due)
		if err != nil {
			continue
		}
		switch {
		case d.Before(time.Now().Truncate(24 * time.Hour)):
			overdue = append(overdue, t)
		case !d.After(soonCutoff):
			dueSoon = append(dueSoon, t)
		}
	}

	printSection(out, color.New(color.FgRed, color.Bold), "OVERDUE", overdue)
	printSection(out, color.New(color.FgYellow, color.Bold), "DUE SOON (7 days)", dueSoon)

	fmt.Fprintln(out, "\nGOALS")
	if len(goals) == 0 {
		fmt.Fprintln(out, "  no goals yet — add one with `hp goal add \"...\"`")
	}
	for _, g := range goals {
		openTasks := store.FilterTasks(tasks, store.TaskFilter{GoalID: g.ID})
		n := 0
		for _, t := range openTasks {
			if open(t) {
				n++
			}
		}
		fmt.Fprintf(out, "  %-8s %-10s %2d open   %s\n", store.ShortID(g.ID), g.Status, n, g.Title)
	}
	return nil
}

func printSection(out io.Writer, c *color.Color, title string, tasks []*model.Task) {
	if len(tasks) == 0 {
		return
	}
	c.Fprintf(out, "%s (%d)\n", title, len(tasks))
	for _, t := range tasks {
		fmt.Fprintf(out, "  %s  p%d  due:%s  %s\n", store.ShortID(t.ID), t.Priority, t.Due, t.Title)
	}
	fmt.Fprintln(out)
}
