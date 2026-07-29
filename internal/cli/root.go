// Package cli wires up the `hp` command tree.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"hp/internal/config"
	"hp/internal/gitsync"
	"hp/internal/store"
)

var dirFlag string

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "hp",
		Short:         "hp is a local, AI-assisted context and task organizer",
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd)
		},
	}
	root.PersistentFlags().StringVar(&dirFlag, "dir", "", "vault directory (default: $HP_HOME or ~/.hp)")

	root.AddCommand(newInitCmd())
	root.AddCommand(newContextCmd())
	root.AddCommand(newGoalCmd())
	root.AddCommand(newTaskCmd())
	root.AddCommand(newAICmd())
	root.AddCommand(newSyncCmd())
	root.AddCommand(newStatusCmd())
	return root
}

// openStore resolves the vault dir and requires it to already be
// initialized (use for every command except `hp init`).
func openStore() (*store.Store, error) {
	dir, err := store.ResolveVaultDir(dirFlag)
	if err != nil {
		return nil, err
	}
	s := store.New(dir)
	if !s.Exists() {
		return nil, fmt.Errorf("no vault found at %s — run `hp init` first", dir)
	}
	return s, nil
}

// autoCommit commits a mutation to the vault's local git repo. hp sync is
// still the only command that talks to a remote — this only ever commits
// locally, giving every change an undo-able history.
func autoCommit(s *store.Store, message string) error {
	_, err := gitsync.CommitAll(s.Dir, message)
	return err
}

func openStoreAndConfig() (*store.Store, config.Config, error) {
	s, err := openStore()
	if err != nil {
		return nil, config.Config{}, err
	}
	cfg, err := config.Load(s.Dir)
	if err != nil {
		return nil, config.Config{}, err
	}
	return s, cfg, nil
}
