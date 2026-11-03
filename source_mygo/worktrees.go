package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Git lists the main repository first, even when queried from a linked worktree.
// Keep that order and use NUL delimiters to preserve arbitrary path names.
type worktree struct {
	Path     string
	Branch   string
	HEAD     string
	Main     bool
	Bare     bool
	Detached bool
	Locked   bool
	Prunable bool
}

func listWorktrees(ctx context.Context, path string) ([]worktree, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", path, "worktree", "list", "--porcelain", "-z")
	hideConsole(cmd)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GIT_") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
	output, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s: %s", path, strings.TrimSpace(string(exit.Stderr)))
		}
		return nil, err
	}
	items, err := parseWorktrees(string(output))
	if err != nil {
		return nil, err
	}
	// Git reports forward slashes on Windows; normalize so paths match the
	// filepath-based paths used for projects and selection state.
	for i := range items {
		items[i].Path = filepath.Clean(items[i].Path)
	}
	return items, nil
}

func parseWorktrees(output string) ([]worktree, error) {
	var result []worktree
	for _, field := range strings.Split(output, "\x00") {
		key, value, _ := strings.Cut(field, " ")
		if key == "worktree" {
			if value == "" {
				return nil, fmt.Errorf("empty worktree path")
			}
			result = append(result, worktree{Path: value, Main: len(result) == 0})
			continue
		}
		if len(result) == 0 {
			continue
		}
		item := &result[len(result)-1]
		switch key {
		case "HEAD":
			item.HEAD = value
		case "branch":
			item.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "bare":
			item.Bare = true
		case "detached":
			item.Detached = true
		case "locked":
			item.Locked = true
		case "prunable":
			item.Prunable = true
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no worktrees returned by Git")
	}
	return result, nil
}
