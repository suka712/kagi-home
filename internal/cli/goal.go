package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"hp/internal/model"
	"hp/internal/store"
)

func newGoalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "goal",
		Short: "Manage long-horizon goals",
	}
	cmd.AddCommand(newGoalAddCmd())
	cmd.AddCommand(newGoalListCmd())
	cmd.AddCommand(newGoalShowCmd())
	return cmd
}

func newGoalAddCmd() *cobra.Command {
	var target string
	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a new goal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			g := &model.Goal{Title: args[0], TargetDate: target}
			if err := s.CreateGoal(g); err != nil {
				return err
			}
			if err := autoCommit(s, "hp: add goal "+g.Title); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created goal %s (%s) -> %s\n", g.ID, g.Title, g.Path)
			return nil
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "target date (YYYY-MM-DD)")
	return cmd
}

func newGoalListCmd() *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List goals",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			goals, err := s.ListGoals()
			if err != nil {
				return err
			}
			for _, g := range goals {
				if status != "" && string(g.Status) != status {
					continue
				}
				target := g.TargetDate
				if target == "" {
					target = "-"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s  %-10s  target:%-12s  %s\n", store.ShortID(g.ID), g.Status, target, g.Title)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "filter by status (active|paused|achieved|dropped)")
	return cmd
}

func newGoalShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a goal's details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			g, err := s.GetGoal(args[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "id:      %s\n", g.ID)
			fmt.Fprintf(out, "title:   %s\n", g.Title)
			fmt.Fprintf(out, "status:  %s\n", g.Status)
			fmt.Fprintf(out, "target:  %s\n", g.TargetDate)
			fmt.Fprintf(out, "created: %s\n", g.Created.Format("2006-01-02"))
			if g.Body != "" {
				fmt.Fprintf(out, "\n%s\n", g.Body)
			}
			return nil
		},
	}
}
