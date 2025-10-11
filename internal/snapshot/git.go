package snapshot

import (
	"os/exec"
	"strings"
)

func getGitInfo(providerDir string) *GitInfo {
	if providerDir == "" {
		return nil
	}

	info := &GitInfo{}

	if out, err := runGitCommand(providerDir, "rev-parse", "HEAD"); err == nil {
		info.Commit = strings.TrimSpace(out)
	}

	if out, err := runGitCommand(providerDir, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		info.Branch = strings.TrimSpace(out)
	}

	if out, err := runGitCommand(providerDir, "status", "--porcelain"); err == nil {
		info.IsDirty = strings.TrimSpace(out) != ""
	}

	if out, err := runGitCommand(providerDir, "remote", "get-url", "origin"); err == nil {
		info.Remote = strings.TrimSpace(out)
	}

	if out, err := runGitCommand(providerDir, "log", "-1", "--pretty=%B"); err == nil {
		info.CommitMsg = strings.TrimSpace(out)
	}

	if info.Commit == "" {
		return nil
	}

	return info
}

func runGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
