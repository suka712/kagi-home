package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage background context docs fed to the AI (bio, CV, constraints, notes)",
	}
	cmd.AddCommand(newContextAddCmd())
	cmd.AddCommand(newContextListCmd())
	return cmd
}

func newContextAddCmd() *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:   "add <file|- >",
		Short: "Add a context doc from a text/markdown file, or from stdin with -",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			var body string
			src := args[0]
			if src == "-" {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				body = string(data)
				if title == "" {
					return fmt.Errorf("--title is required when reading from stdin")
				}
			} else {
				ext := strings.ToLower(filepath.Ext(src))
				if ext != ".md" && ext != ".txt" {
					return fmt.Errorf("unsupported file type %q — add .md/.txt content directly; for other files (PDFs, etc.) convert to text first, or keep the original alongside your notes", ext)
				}
				data, err := os.ReadFile(src)
				if err != nil {
					return err
				}
				body = string(data)
				if title == "" {
					title = strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
				}
			}
			doc, err := s.CreateContextDoc(title, body)
			if err != nil {
				return err
			}
			if err := autoCommit(s, "hp: add context doc "+doc.Title); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "added context doc %q -> %s\n", doc.Title, doc.Path)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "title for the context doc (required when reading from stdin)")
	return cmd
}

func newContextListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List context docs",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			docs, err := s.ListContextDocs()
			if err != nil {
				return err
			}
			if len(docs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no context docs yet — add one with `hp context add <file>`")
				return nil
			}
			for _, d := range docs {
				fmt.Fprintf(cmd.OutOrStdout(), "%-30s %s\n", d.Title, filepath.Base(d.Path))
			}
			return nil
		},
	}
}
