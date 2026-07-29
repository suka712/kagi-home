package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"hp/internal/gitsync"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Commit any pending local changes, pull --rebase, and push (if a remote is configured)",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			msg := fmt.Sprintf("hp sync: %s", time.Now().Format(time.RFC3339))
			res, err := gitsync.Sync(s.Dir, msg)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if res.Committed {
				fmt.Fprintln(out, "committed pending changes")
			} else {
				fmt.Fprintln(out, "no pending changes to commit")
			}
			if res.Pushed {
				fmt.Fprintln(out, "pulled and pushed to remote")
			} else if res.Message != "" {
				fmt.Fprintln(out, res.Message)
			}
			return nil
		},
	}
}
