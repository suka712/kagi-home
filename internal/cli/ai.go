package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"hp/internal/ai"
	"hp/internal/apply"
	"hp/internal/model"
	"hp/internal/ollama"
	"hp/internal/store"
)

const aiTimeout = 3 * time.Minute

func newAICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "AI-assisted planning, prioritization, and extraction (via local Ollama)",
	}
	cmd.AddCommand(newAIPlanCmd())
	cmd.AddCommand(newAIPrioritizeCmd())
	cmd.AddCommand(newAIExtractCmd())
	cmd.AddCommand(newAIAskCmd())
	return cmd
}

// newAIClient opens the store+config and builds an Ollama client, pinging
// it first so failures are a clear message instead of a raw dial error.
func newAIClient(ctx context.Context) (*store.Store, *ollama.Client, error) {
	s, cfg, err := openStoreAndConfig()
	if err != nil {
		return nil, nil, err
	}
	client := ollama.NewClient(cfg.Ollama.Host, cfg.Ollama.Model, cfg.Ollama.Temperature)
	if err := client.Ping(ctx); err != nil {
		return nil, nil, err
	}
	return s, client, nil
}

func newAIPlanCmd() *cobra.Command {
	var goalRef string
	var yes bool
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Propose subtasks for a goal",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), aiTimeout)
			defer cancel()

			s, client, err := newAIClient(ctx)
			if err != nil {
				return err
			}
			if goalRef == "" {
				return fmt.Errorf("--goal is required")
			}
			goal, err := s.GetGoal(goalRef)
			if err != nil {
				return fmt.Errorf("--goal: %w", err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "asking %s for subtasks toward %q...\n\n", client.Model, goal.Title)

			result, err := ai.Plan(ctx, client, s, goal)
			if err != nil {
				return err
			}

			if len(result.ClarifyingQuestions) > 0 {
				fmt.Fprintln(out, "The model had clarifying questions before it could plan confidently:")
				for _, q := range result.ClarifyingQuestions {
					fmt.Fprintln(out, " -", q)
				}
				fmt.Fprintln(out, "(consider `hp ai ask --goal", goal.ID, "` or `hp context add` to fill these in)")
				fmt.Fprintln(out)
			}

			if len(result.Tasks) == 0 {
				fmt.Fprintln(out, "no subtasks proposed")
				return nil
			}

			confirmer := apply.NewConfirmer(cmd.InOrStdin(), out, yes)
			accepted, skipped := 0, 0
			for _, p := range result.Tasks {
				accept, quit, err := confirmer.Confirm(apply.RenderNewTask(p))
				if err != nil {
					return err
				}
				if quit {
					skipped += len(result.Tasks) - accepted - skipped
					break
				}
				if !accept {
					skipped++
					continue
				}
				t := ai.NewTaskFromProposal(p, goal.ID, "")
				if err := s.CreateTask(t); err != nil {
					return err
				}
				accepted++
			}
			if accepted > 0 {
				if err := autoCommit(s, fmt.Sprintf("hp ai plan: %d subtask(s) for %s", accepted, goal.Title)); err != nil {
					return err
				}
			}
			fmt.Fprintf(out, "\n%d accepted, %d skipped\n", accepted, skipped)
			return nil
		},
	}
	cmd.Flags().StringVar(&goalRef, "goal", "", "goal id (or prefix) to plan for (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "accept all proposals without prompting")
	return cmd
}

func newAIPrioritizeCmd() *cobra.Command {
	var goalRef string
	var yes bool
	cmd := &cobra.Command{
		Use:   "prioritize",
		Short: "Propose priority changes across open tasks (optionally scoped to one goal)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), aiTimeout)
			defer cancel()

			s, client, err := newAIClient(ctx)
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
			fmt.Fprintf(out, "asking %s to review priorities...\n\n", client.Model)

			result, _, err := ai.Prioritize(ctx, client, s, goalID)
			if err != nil {
				return err
			}
			if len(result.Changes) == 0 {
				fmt.Fprintln(out, "no priority changes proposed")
				return nil
			}

			confirmer := apply.NewConfirmer(cmd.InOrStdin(), out, yes)
			accepted, skipped := 0, 0
			for _, c := range result.Changes {
				t, err := s.GetTask(c.TaskID)
				if err != nil {
					fmt.Fprintf(out, "(skipping unresolvable task_id %q: %v)\n", c.TaskID, err)
					continue
				}
				if c.NewPriority < model.PriorityHighest || c.NewPriority > model.PriorityLowest {
					fmt.Fprintf(out, "(skipping %s: invalid priority %d)\n", t.Title, c.NewPriority)
					continue
				}
				if c.NewPriority == t.Priority {
					// Small models sometimes "propose" a change back to the same
					// value despite being told not to; nothing to confirm here.
					continue
				}
				accept, quit, err := confirmer.Confirm(apply.RenderPriorityChange(t, c))
				if err != nil {
					return err
				}
				if quit {
					break
				}
				if !accept {
					skipped++
					continue
				}
				t.Priority = c.NewPriority
				if err := s.UpdateTask(t); err != nil {
					return err
				}
				accepted++
			}
			if accepted > 0 {
				if err := autoCommit(s, fmt.Sprintf("hp ai prioritize: %d task(s) reprioritized", accepted)); err != nil {
					return err
				}
			}
			fmt.Fprintf(out, "\n%d accepted, %d skipped\n", accepted, skipped)
			return nil
		},
	}
	cmd.Flags().StringVar(&goalRef, "goal", "", "limit to tasks under this goal id (or prefix)")
	cmd.Flags().BoolVar(&yes, "yes", false, "accept all proposals without prompting")
	return cmd
}

func newAIExtractCmd() *cobra.Command {
	var from, goalRef, taskRef, title string
	var yes bool
	cmd := &cobra.Command{
		Use:   "extract",
		Short: "Extract action items from a document into a task",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), aiTimeout)
			defer cancel()

			s, client, err := newAIClient(ctx)
			if err != nil {
				return err
			}
			if from == "" {
				return fmt.Errorf("--from <file|-> is required")
			}
			var docText string
			if from == "-" {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				docText = string(data)
			} else {
				data, err := os.ReadFile(from)
				if err != nil {
					return err
				}
				docText = string(data)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "asking %s to extract action items...\n\n", client.Model)

			result, err := ai.Extract(ctx, client, s, docText)
			if err != nil {
				return err
			}
			if len(result.ActionItems) == 0 {
				fmt.Fprintln(out, "no action items found")
				return nil
			}

			var target *model.Task
			isNew := false
			if taskRef != "" {
				target, err = s.GetTask(taskRef)
				if err != nil {
					return fmt.Errorf("--task: %w", err)
				}
			} else {
				taskTitle := result.SuggestedTaskTitle
				if title != "" {
					taskTitle = title
				}
				target = &model.Task{Title: taskTitle, Status: model.StatusTodo, Priority: 3, Source: model.SourceAI}
				if goalRef != "" {
					g, err := s.GetGoal(goalRef)
					if err != nil {
						return fmt.Errorf("--goal: %w", err)
					}
					target.GoalID = g.ID
				}
				isNew = true
			}

			confirmer := apply.NewConfirmer(cmd.InOrStdin(), out, yes)
			var accepted []string
			for _, item := range result.ActionItems {
				accept, quit, err := confirmer.Confirm(apply.RenderActionItem(item))
				if err != nil {
					return err
				}
				if quit {
					break
				}
				if accept {
					accepted = append(accepted, item)
				}
			}
			if len(accepted) == 0 {
				fmt.Fprintln(out, "\nnothing accepted, no changes made")
				return nil
			}

			if target.Body != "" {
				target.Body += "\n"
			}
			for _, item := range accepted {
				target.Body += "- [ ] " + item + "\n"
			}
			if isNew {
				err = s.CreateTask(target)
			} else {
				err = s.UpdateTask(target)
			}
			if err != nil {
				return err
			}
			if err := autoCommit(s, fmt.Sprintf("hp ai extract: %d item(s) into %s", len(accepted), target.Title)); err != nil {
				return err
			}
			fmt.Fprintf(out, "\nadded %d action item(s) to %q (%s)\n", len(accepted), target.Title, target.Path)
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "file to extract from, or - for stdin (required)")
	cmd.Flags().StringVar(&goalRef, "goal", "", "goal id (or prefix) for a newly created task")
	cmd.Flags().StringVar(&taskRef, "task", "", "existing task id (or prefix) to append action items to, instead of creating a new task")
	cmd.Flags().StringVar(&title, "title", "", "override the AI-suggested title for a newly created task")
	cmd.Flags().BoolVar(&yes, "yes", false, "accept all proposals without prompting")
	return cmd
}

func newAIAskCmd() *cobra.Command {
	var goalRef string
	cmd := &cobra.Command{
		Use:   "ask",
		Short: "Let the AI ask clarifying questions about a goal; answers are saved to context",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), aiTimeout)
			defer cancel()

			s, client, err := newAIClient(ctx)
			if err != nil {
				return err
			}
			if goalRef == "" {
				return fmt.Errorf("--goal is required")
			}
			goal, err := s.GetGoal(goalRef)
			if err != nil {
				return fmt.Errorf("--goal: %w", err)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "asking %s what it needs to know about %q...\n\n", client.Model, goal.Title)

			result, err := ai.Ask(ctx, client, s, goal)
			if err != nil {
				return err
			}
			if len(result.Questions) == 0 {
				fmt.Fprintln(out, "no clarifying questions right now — context looks sufficient")
				return nil
			}

			reader := bufioReader(cmd.InOrStdin())
			var section string
			answered := 0
			for i, q := range result.Questions {
				fmt.Fprintf(out, "%d. %s\n> ", i+1, q)
				line, _ := reader.ReadString('\n')
				answer := trimNewline(line)
				if answer == "" {
					continue
				}
				section += fmt.Sprintf("### %s — %s\n**Q:** %s\n**A:** %s\n\n", time.Now().Format("2006-01-02"), goal.Title, q, answer)
				answered++
			}
			if answered == 0 {
				fmt.Fprintln(out, "\nno answers given, nothing saved")
				return nil
			}
			if err := s.AppendToContextDoc("qa-log", section); err != nil {
				return err
			}
			if err := autoCommit(s, fmt.Sprintf("hp ai ask: %d answer(s) for %s", answered, goal.Title)); err != nil {
				return err
			}
			fmt.Fprintf(out, "\nsaved %d answer(s) to context/qa-log.md\n", answered)
			return nil
		},
	}
	cmd.Flags().StringVar(&goalRef, "goal", "", "goal id (or prefix) to ask about (required)")
	return cmd
}
