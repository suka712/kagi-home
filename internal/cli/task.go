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

func newTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks and subtasks",
	}
	cmd.AddCommand(newTaskAddCmd())
	cmd.AddCommand(newTaskListCmd())
	cmd.AddCommand(newTaskShowCmd())
	cmd.AddCommand(newTaskEditCmd())
	cmd.AddCommand(newTaskDoneCmd())
	return cmd
}

func newTaskAddCmd() *cobra.Command {
	var goalRef, parentRef, due string
	var priority int
	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a task (optionally under a goal and/or as a subtask of another task)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			t := &model.Task{Title: args[0], Due: due, Priority: priority}
			if goalRef != "" {
				g, err := s.GetGoal(goalRef)
				if err != nil {
					return fmt.Errorf("--goal: %w", err)
				}
				t.GoalID = g.ID
			}
			if parentRef != "" {
				p, err := s.GetTask(parentRef)
				if err != nil {
					return fmt.Errorf("--parent: %w", err)
				}
				t.ParentID = p.ID
				if t.GoalID == "" {
					t.GoalID = p.GoalID
				}
			}
			if err := s.CreateTask(t); err != nil {
				return err
			}
			if err := autoCommit(s, "hp: add task "+t.Title); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created task %s (%s) -> %s\n", t.ID, t.Title, t.Path)
			return nil
		},
	}
	cmd.Flags().StringVar(&goalRef, "goal", "", "goal id (or prefix) this task belongs to")
	cmd.Flags().StringVar(&parentRef, "parent", "", "parent task id (or prefix), makes this a subtask")
	cmd.Flags().IntVar(&priority, "priority", 3, "priority 1 (highest) - 5 (lowest)")
	cmd.Flags().StringVar(&due, "due", "", "due date (YYYY-MM-DD)")
	return cmd
}

func newTaskListCmd() *cobra.Command {
	var goalRef, status string
	var priority int
	var tree bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			all, err := s.ListTasks()
			if err != nil {
				return err
			}
			var goalID string
			if goalRef != "" {
				g, err := s.GetGoal(goalRef)
				if err != nil {
					return fmt.Errorf("--goal: %w", err)
				}
				goalID = g.ID
			}

			out := cmd.OutOrStdout()
			if tree {
				roots := store.FilterTasks(all, store.TaskFilter{GoalID: goalID})
				for _, t := range roots {
					if t.ParentID == "" {
						printTaskLine(out, t, 0)
						printChildren(out, all, t.ID, 1)
					}
				}
				return nil
			}

			filtered := store.FilterTasks(all, store.TaskFilter{
				GoalID:   goalID,
				Status:   model.Status(status),
				Priority: priority,
			})
			if len(filtered) == 0 {
				fmt.Fprintln(out, "no matching tasks")
				return nil
			}
			for _, t := range filtered {
				printTaskLine(out, t, 0)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&goalRef, "goal", "", "filter by goal id (or prefix)")
	cmd.Flags().StringVar(&status, "status", "", "filter by status")
	cmd.Flags().IntVar(&priority, "priority", 0, "filter by priority (1-5)")
	cmd.Flags().BoolVar(&tree, "tree", false, "show tasks as a parent/subtask tree")
	return cmd
}

func printChildren(out io.Writer, all []*model.Task, parentID string, depth int) {
	for _, c := range store.Children(all, parentID) {
		printTaskLine(out, c, depth)
		printChildren(out, all, c.ID, depth+1)
	}
}

func printTaskLine(out io.Writer, t *model.Task, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	line := fmt.Sprintf("%s%s  p%d  %-11s  %s", indent, store.ShortID(t.ID), t.Priority, t.Status, t.Title)
	if t.Due != "" {
		line += "  due:" + t.Due
	}
	line += "\n"
	switch {
	case t.Status == model.StatusDone:
		color.New(color.FgGreen).Fprint(out, line)
	case t.Due != "" && isOverdue(t.Due) && t.Status != model.StatusDone && t.Status != model.StatusCancelled:
		color.New(color.FgRed).Fprint(out, line)
	default:
		fmt.Fprint(out, line)
	}
}

func isOverdue(due string) bool {
	d, err := time.Parse("2006-01-02", due)
	if err != nil {
		return false
	}
	return d.Before(time.Now().Truncate(24 * time.Hour))
}

func newTaskShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a task's details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			t, err := s.GetTask(args[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "id:       %s\n", t.ID)
			fmt.Fprintf(out, "title:    %s\n", t.Title)
			fmt.Fprintf(out, "status:   %s\n", t.Status)
			fmt.Fprintf(out, "priority: %d\n", t.Priority)
			fmt.Fprintf(out, "goal_id:  %s\n", t.GoalID)
			fmt.Fprintf(out, "parent_id:%s\n", t.ParentID)
			fmt.Fprintf(out, "due:      %s\n", t.Due)
			fmt.Fprintf(out, "tags:     %v\n", t.Tags)
			fmt.Fprintf(out, "source:   %s\n", t.Source)
			fmt.Fprintf(out, "created:  %s\n", t.Created.Format("2006-01-02"))
			if t.Body != "" {
				fmt.Fprintf(out, "\n%s\n", t.Body)
			}
			return nil
		},
	}
}

func newTaskEditCmd() *cobra.Command {
	var priority int
	var status, due, title string
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Manually edit a task's priority, status, due date, or title",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			t, err := s.GetTask(args[0])
			if err != nil {
				return err
			}
			changed := false
			if cmd.Flags().Changed("priority") {
				t.Priority = priority
				changed = true
			}
			if status != "" {
				t.Status = model.Status(status)
				changed = true
			}
			if cmd.Flags().Changed("due") {
				t.Due = due
				changed = true
			}
			if title != "" {
				t.Title = title
				changed = true
			}
			if !changed {
				return fmt.Errorf("nothing to change — pass at least one of --priority/--status/--due/--title")
			}
			if err := s.UpdateTask(t); err != nil {
				return err
			}
			if err := autoCommit(s, "hp: edit task "+t.Title); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "updated task %s\n", t.ID)
			return nil
		},
	}
	cmd.Flags().IntVar(&priority, "priority", 0, "priority 1 (highest) - 5 (lowest)")
	cmd.Flags().StringVar(&status, "status", "", "status (todo|in_progress|blocked|done|cancelled)")
	cmd.Flags().StringVar(&due, "due", "", "due date (YYYY-MM-DD, empty clears it)")
	cmd.Flags().StringVar(&title, "title", "", "new title")
	return cmd
}

func newTaskDoneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "done <id>",
		Short: "Mark a task done",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			t, err := s.GetTask(args[0])
			if err != nil {
				return err
			}
			t.Status = model.StatusDone
			if err := s.UpdateTask(t); err != nil {
				return err
			}
			if err := autoCommit(s, "hp: complete task "+t.Title); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "done: %s\n", t.Title)
			return nil
		},
	}
}
