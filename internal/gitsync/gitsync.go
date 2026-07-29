// Package gitsync shells out to the system git binary so that auth,
// credentials, and remotes work exactly as they do for the user's normal
// git workflow, on any git host.
package gitsync

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, out.String())
	}
	return out.String(), nil
}

// Init creates a git repo in dir if one doesn't already exist.
func Init(dir string) error {
	if exists(dir) {
		return nil
	}
	_, err := run(dir, "init")
	return err
}

func exists(dir string) bool {
	_, err := run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

// AddRemote sets (or replaces) the "origin" remote.
func AddRemote(dir, url string) error {
	if HasRemote(dir) {
		_, err := run(dir, "remote", "set-url", "origin", url)
		return err
	}
	_, err := run(dir, "remote", "add", "origin", url)
	return err
}

func HasRemote(dir string) bool {
	out, err := run(dir, "remote")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "origin" {
			return true
		}
	}
	return false
}

// HasChanges reports whether the working tree has uncommitted changes.
func HasChanges(dir string) (bool, error) {
	out, err := run(dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// CommitAll stages everything and commits with message, if there is
// anything to commit. Returns false if there was nothing to commit.
func CommitAll(dir, message string) (bool, error) {
	changed, err := HasChanges(dir)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	if _, err := run(dir, "add", "-A"); err != nil {
		return false, err
	}
	if _, err := run(dir, "commit", "-m", message); err != nil {
		return false, err
	}
	return true, nil
}

// SyncResult summarizes what a Sync call did.
type SyncResult struct {
	Committed bool
	Pulled    bool
	Pushed    bool
	Message   string
}

// Sync commits any local changes, then (if a remote is configured) pulls
// with rebase and pushes. This is the only operation that talks to a
// remote; everything else stays local-only.
func Sync(dir, commitMessage string) (SyncResult, error) {
	var res SyncResult
	committed, err := CommitAll(dir, commitMessage)
	if err != nil {
		return res, fmt.Errorf("commit: %w", err)
	}
	res.Committed = committed

	if !HasRemote(dir) {
		res.Message = "no remote configured; changes committed locally only"
		return res, nil
	}

	branch, err := currentBranch(dir)
	if err != nil {
		return res, fmt.Errorf("determining current branch: %w", err)
	}

	remoteHasBranch, err := remoteBranchExists(dir, branch)
	if err != nil {
		return res, fmt.Errorf("checking remote branch: %w", err)
	}

	if remoteHasBranch {
		if _, err := run(dir, "pull", "--rebase", "origin", branch); err != nil {
			return res, fmt.Errorf("pull --rebase: %w", err)
		}
		res.Pulled = true
	}

	if _, err := run(dir, "push", "-u", "origin", branch); err != nil {
		return res, fmt.Errorf("push: %w", err)
	}
	res.Pushed = true
	return res, nil
}

func currentBranch(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// remoteBranchExists reports whether origin already has branch, so Sync
// knows whether to pull before pushing (a brand new remote has no branches
// to pull from, and `git pull` on one fails outright).
func remoteBranchExists(dir, branch string) (bool, error) {
	out, err := run(dir, "ls-remote", "--heads", "origin", branch)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}
