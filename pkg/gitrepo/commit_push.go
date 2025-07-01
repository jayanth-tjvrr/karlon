package gitrepo

import (
	"fmt"
	"os"
	"path/filepath"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"time"
)

// CommitAndPushFile clones the repo, creates folder(s) if needed, writes the file, commits, and pushes.
func CommitAndPushFile(repoUrl, branch, filePath string, content []byte, commitMsg string) error {
	dir, err := os.MkdirTemp("", "kkarlon-git-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	repo, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:           repoUrl,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repo: %w", err)
	}

	fullPath := filepath.Join(dir, filePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}
	_, err = wt.Add(filePath)
	if err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}
	_, err = wt.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kkarlonctl",
			Email: "kkarlonctl@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Use GIT_USERNAME and GIT_TOKEN env vars for authentication
	username := os.Getenv("GIT_USERNAME")
	password := os.Getenv("GIT_TOKEN")
	if username == "" || password == "" {
		return fmt.Errorf("GIT_USERNAME and GIT_TOKEN environment variables must be set for push authentication")
	}
	pushErr := repo.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: username,
			Password: password,
		},
	})
	if pushErr != nil {
		return fmt.Errorf("failed to push: %w", pushErr)
	}
	return nil
}
