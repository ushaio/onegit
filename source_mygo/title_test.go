package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onegit/source_mygo/launchers"
)

// The repository's actions live in the title bar beside its name, not in a
// strip of their own.
func TestRepositoryTitleActions(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "demo")
	testGit(t, "init", repo)
	s := &settings{systemLanguage: "zh-CN", selectedProject: &projectNode{Path: repo, Repository: true}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).FillWidth().Height(52).Padding(0, 8).Children(func() { s.repositoryTitle(c) })
	}, 800, 52)
	tt.Frame()
	if !tt.HasText("demo") {
		t.Fatal("project name missing from the title bar")
	}
	for _, label := range []string{"获取", "拉取", "推送", "贮藏"} {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("%s missing from the title bar", label)
		}
		if r.Y < 0 || r.Y+r.H > 52 {
			t.Fatalf("%s outside the title bar: %+v", label, r)
		}
	}

	// The toolbar sits in the middle, and the name stays with several
	// worktrees, whose tabs belong to the page.
	fetch, _ := tt.Find("获取")
	stash, _ := tt.Find("贮藏")
	if mid := (fetch.X + stash.X + stash.W) / 2; mid < 380 || mid > 420 {
		t.Fatalf("toolbar not centered: %v to %v", fetch, stash)
	}
	s.overview.Worktrees = []worktree{{Path: repo}, {Path: filepath.Join(root, "feature")}}
	tt.Frame()
	if !tt.HasText("demo") || tt.HasText("feature") {
		t.Fatalf("worktrees in the title bar: %v", tt.Texts())
	}

	page := ui.NewTester(s.welcome, 800, 600)
	page.Frame()
	if page.HasText("获取") {
		t.Fatal("the repository page still has a toolbar strip")
	}
}

// The Git actions are followed by the chosen launchers for the selected
// worktree and the repository's GitHub page; a launcher that fails shows a
// toast.
func TestRepositoryTitleLaunchers(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(root, "demo")
	testGit(t, "init", repo)
	testGit(t, "-C", repo, "remote", "add", "origin", "git@github.com:o/demo.git")
	s := &settings{statePath: filepath.Join(t.TempDir(), "state.yaml"), systemLanguage: "en",
		values: preferences{Roots: []string{root}, Editor: launchers.None}}
	for _, id := range []string{"finder", "ghostty", "vscode"} {
		l, _ := launchers.ByID(id)
		s.installed = append(s.installed, launchers.Installed{Launcher: l, Path: "/Applications/" + l.Name + ".app"})
	}
	ran := make(chan *exec.Cmd, 1)
	s.runLauncher = func(cmd *exec.Cmd) error { ran <- cmd; return errors.New("denied") }
	callbacks := make(chan func(), 64)
	s.startProjectScan(func(fn func()) { callbacks <- fn })
	awaitNavigation(t, s, callbacks)

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).FillWidth().Height(52).Padding(0, 8).Children(func() { s.repositoryTitle(c) })
	}, 1000, 52)
	tt.Frame()
	stash, _ := tt.Find("Stash")
	for _, label := range []string{"Open in Finder", "Open in Ghostty", "Open on GitHub"} {
		if r, ok := tt.Find(label); !ok || r.X < stash.X || r.X+r.W > 1000 {
			t.Fatalf("%s missing or before the Git actions: %v", label, tt.Texts())
		}
	}
	if _, ok := tt.Find("Open in Visual Studio Code"); ok {
		t.Fatal("hidden editor shown")
	}
	if err := tt.Click("Open on GitHub"); err != nil {
		t.Fatal(err)
	}
	if opened := tt.OpenedURLs(); len(opened) != 1 || opened[0] != "https://github.com/o/demo" {
		t.Fatalf("opened: %v", opened)
	}
	if err := tt.Click("Open in Ghostty"); err != nil {
		t.Fatal(err)
	}
	select {
	case cmd := <-ran:
		if !strings.HasSuffix(cmd.Args[len(cmd.Args)-1], repo) {
			t.Fatalf("launched %v", cmd.Args)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("launcher did not run")
	}
	select {
	case fn := <-callbacks:
		fn()
	case <-time.After(10 * time.Second):
		t.Fatal("failure not reported")
	}
	if s.toast == nil || s.toast.Title != "Could not open Ghostty" || s.toast.Description != "denied" {
		t.Fatalf("toast: %+v", s.toast)
	}
}
