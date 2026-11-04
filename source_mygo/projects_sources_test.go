package main

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRepositorySourceParsing(t *testing.T) {
	cases := []struct {
		remote string
		source string
	}{
		{"https://github.com/owner/repo.git", "github"},
		{"https://www.github.com/owner/repo", "github"},
		{"git@github.com:owner/repo.git", "github"},
		{"ssh://git@github.com/owner/repo.git", "github"},
		{"https://gitee.com/owner/repo.git", "gitee"},
		{"git@gitee.com:owner/repo.git", "gitee"},
		{"https://gitcode.com/owner/repo.git", "gitcode"},
		{"https://gitcode.net/owner/repo.git", "gitcode"},
		{"git@gitcode.com:owner/repo.git", "gitcode"},
		{"https://gitlab.com/owner/repo.git", ""},
		{"https://github.com/owner/repo", "github"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := repositorySource(tc.remote); got != tc.source {
			t.Errorf("repositorySource(%q) = %q, want %q", tc.remote, got, tc.source)
		}
	}
}

func TestSourceIconKnownHosts(t *testing.T) {
	for _, source := range []string{"github", "gitee", "gitcode"} {
		icon := sourceIcon(source)
		if icon == nil {
			t.Fatalf("sourceIcon(%q) = nil", source)
		}
		// Parsing here surfaces a malformed embedded SVG before it reaches the UI.
		if icon() == nil {
			t.Errorf("sourceIcon(%q) did not parse", source)
		}
	}
	if sourceIcon("gitlab") != nil {
		t.Error("sourceIcon of an unknown host must be nil")
	}
}

func TestScanProjectsRecordsSource(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	remotes := map[string]string{
		"github-repo":  "https://github.com/owner/repo.git",
		"gitee-repo":   "git@gitee.com:owner/repo.git",
		"gitcode-repo": "https://gitcode.com/owner/repo.git",
		"mirror-repo":  "https://gitlab.com/owner/repo.git",
	}
	want := map[string]string{
		"github-repo":  "github",
		"gitee-repo":   "gitee",
		"gitcode-repo": "gitcode",
		"mirror-repo":  "",
	}
	paths := map[string]string{}
	for name, remote := range remotes {
		path := filepath.Join(root, name)
		testGit(t, "init", path)
		testGit(t, "-C", path, "remote", "add", "origin", remote)
		paths[name] = path
	}
	plain := filepath.Join(root, "plain-repo")
	testGit(t, "init", plain)
	paths["plain-repo"] = plain
	want["plain-repo"] = ""

	tree, err := scanProjects(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	for name, path := range paths {
		node := findProject(tree, path)
		if node == nil {
			t.Fatalf("repository %q not found", name)
		}
		if node.Source != want[name] {
			t.Errorf("%q source = %q, want %q", name, node.Source, want[name])
		}
	}
}
