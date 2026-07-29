// Package store reads and writes the vault: a directory of markdown files
// with YAML frontmatter representing goals, tasks, and context docs. The
// vault is the source of truth; nothing is cached or indexed separately.
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var errTooManyCollisions = errors.New("could not find a unique filename")

// ShortID returns the trailing random segment of a ULID for display and
// as something short enough for a human to type. The leading segment of a
// ULID is a millisecond timestamp, so IDs minted close together (e.g.
// several tasks added in one sitting) share a long common prefix — the
// suffix is what actually differs.
func ShortID(id string) string {
	const n = 8
	if len(id) <= n {
		return id
	}
	return id[len(id)-n:]
}

// Store is a handle onto a vault directory.
type Store struct {
	Dir string
}

func New(dir string) *Store {
	return &Store{Dir: dir}
}

// ResolveVaultDir determines the vault directory: --dir flag > HP_HOME env
// var > default ~/.hp.
func ResolveVaultDir(flagDir string) (string, error) {
	if flagDir != "" {
		return expandPath(flagDir)
	}
	if env := os.Getenv("HP_HOME"); env != "" {
		return expandPath(env)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hp"), nil
}

func expandPath(p string) (string, error) {
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, strings.TrimPrefix(p, "~"))
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func (s *Store) contextDir() string { return filepath.Join(s.Dir, "context") }
func (s *Store) goalsDir() string   { return filepath.Join(s.Dir, "goals") }
func (s *Store) tasksDir() string   { return filepath.Join(s.Dir, "tasks") }
func (s *Store) filesDir() string   { return filepath.Join(s.Dir, "context", "files") }

// Init creates the vault directory structure. Safe to call on an existing
// vault (idempotent).
func (s *Store) Init() error {
	dirs := []string{s.contextDir(), s.goalsDir(), s.tasksDir(), s.filesDir()}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", d, err)
		}
	}
	return nil
}

// Exists reports whether the vault directory has already been initialized.
func (s *Store) Exists() bool {
	return exists(s.goalsDir())
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// listMarkdownFiles returns the paths of all *.md files directly inside dir,
// sorted for stable output.
func listMarkdownFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	sort.Strings(paths)
	return paths, nil
}

// resolveID finds the item whose ID exactly matches, or uniquely contains,
// input among ids. ULIDs share a long timestamp prefix for IDs minted close
// together in time, so matching (and the short IDs shown to the user) use
// the trailing random segment rather than a prefix — see ShortID. Contains
// is used (not just suffix) so a pasted short ID always resolves regardless
// of how it was derived.
func resolveID(ids []string, input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("no id given")
	}
	for _, id := range ids {
		if id == input {
			return id, nil
		}
	}
	var matches []string
	for _, id := range ids {
		if strings.Contains(id, input) {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no item found with id %q", input)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("id %q is ambiguous, matches: %s", input, strings.Join(matches, ", "))
	}
}
