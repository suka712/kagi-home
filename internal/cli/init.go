package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"hp/internal/config"
	"hp/internal/gitsync"
	"hp/internal/store"
)

func newInitCmd() *cobra.Command {
	var remote string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new vault (goals/tasks/context store) and git repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := store.ResolveVaultDir(dirFlag)
			if err != nil {
				return err
			}
			s := store.New(dir)
			alreadyInitialized := s.Exists()
			if err := s.Init(); err != nil {
				return err
			}

			gitignorePath := filepath.Join(dir, ".gitignore")
			if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
				if err := os.WriteFile(gitignorePath, []byte(".DS_Store\n"), 0o644); err != nil {
					return err
				}
			}

			configPath := filepath.Join(dir, "config.yaml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				if err := config.Save(dir, config.Default()); err != nil {
					return err
				}
			}

			if err := gitsync.Init(dir); err != nil {
				return err
			}
			if _, err := gitsync.CommitAll(dir, "hp: initialize vault"); err != nil {
				return err
			}

			if remote != "" {
				if err := gitsync.AddRemote(dir, remote); err != nil {
					return err
				}
				if res, err := gitsync.Sync(dir, "hp: initialize vault"); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "remote added, but initial push failed: %v\n(run `hp sync` once you've resolved this)\n", err)
				} else if res.Pushed {
					fmt.Fprintln(cmd.OutOrStdout(), "pushed initial commit to", remote)
				}
			}

			if alreadyInitialized {
				fmt.Fprintf(cmd.OutOrStdout(), "vault already existed at %s (left contents in place)\n", dir)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "initialized vault at %s\n", dir)
				fmt.Fprintln(cmd.OutOrStdout(), "edit", configPath, "to change the ollama host/model if needed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&remote, "remote", "", "git remote URL to push the vault to (any git host)")
	return cmd
}
