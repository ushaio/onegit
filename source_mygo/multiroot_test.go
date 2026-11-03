package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// everyFolder scans plain folders that each hold one repository.
func everyFolder(t *testing.T) (roots []string, names []string) {
	t.Helper()
	for _, name := range []string{"first", "second", "third"} {
		parent, err := filepath.EvalSymlinks(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		testGit(t, "init", filepath.Join(parent, name+"-repo"))
		roots = append(roots, parent)
		names = append(names, name+"-repo")
	}
	return roots, names
}

// Every added folder is a top-level row, and each repository under it can be
// selected, not only the first folder's.
func TestMultipleFoldersInSidebar(t *testing.T) {
	roots, names := everyFolder(t)
	s := &settings{statePath: filepath.Join(t.TempDir(), "state.yaml"), systemLanguage: "en-US", values: preferences{Roots: roots}}
	finishScan(t, s)
	if len(s.projectTree.Children) != len(roots) {
		t.Fatalf("folders missing from the tree: %+v", s.projectTree.Children)
	}
	view := ui.NewTester(s.worktreeSidebar, 300, 720)
	view.SetPreferences(ui.Preferences{ReduceMotion: true})
	view.Frame()
	// Every folder row shows its projects without a click; each project is
	// selectable, not only the first folder's.
	for _, node := range s.projectTree.Children {
		if !node.Open {
			t.Fatalf("a new folder should start expanded: %+v", node)
		}
	}
	for _, name := range names {
		if !view.HasText(name) {
			t.Fatalf("project %s of an added folder is not listed: %q", name, view.Texts())
		}
	}
	for _, name := range names {
		if err := view.Click(name); err != nil {
			t.Fatalf("click %s: %v", name, err)
		}
		if s.selectedProject == nil || filepath.Base(s.selectedProject.Path) != name {
			t.Fatalf("click %s selected %+v", name, s.selectedProject)
		}
		view.Frame()
	}
}

// A folder that is itself a repository is selectable like a project, and can
// still be removed from the sidebar as a folder.
func TestRepositoryFolderIsSelectable(t *testing.T) {
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(parent, "solo")
	testGit(t, "init", repo)
	s := &settings{statePath: filepath.Join(t.TempDir(), "state.yaml"), path: filepath.Join(t.TempDir(), "config.yaml"), systemLanguage: "en-US", values: preferences{Roots: []string{repo}}}
	finishScan(t, s)
	if len(s.projectTree.Children) != 1 || !s.projectTree.Children[0].Repository {
		t.Fatalf("repository folder not scanned as a repository: %+v", s.projectTree.Children)
	}
	view := ui.NewTester(s.worktreeSidebar, 300, 720)
	view.SetPreferences(ui.Preferences{ReduceMotion: true})
	view.Frame()
	if err := view.Click("solo"); err != nil {
		t.Fatal(err)
	}
	if s.selectedProject == nil || s.selectedProject.Path != repo {
		t.Fatalf("repository folder did not select: %+v", s.selectedProject)
	}
	if err := view.RightClick("solo"); err != nil {
		t.Fatal(err)
	}
	menu := view.Menu()
	var hasRemoveFolder, hasRemoveProject bool
	for _, item := range menu {
		hasRemoveFolder = hasRemoveFolder || item == "Remove Folder"
		hasRemoveProject = hasRemoveProject || item == "Remove Project"
	}
	if !hasRemoveFolder || hasRemoveProject {
		t.Fatalf("repository folder menu: %q", menu)
	}
	if err := view.ChooseMenuItem("Remove Folder"); err != nil {
		t.Fatal(err)
	}
	if len(s.roots()) != 0 {
		t.Fatalf("folder not removed: %v", s.roots())
	}
}

// A folder that overlaps a listed one is refused with a message naming the
// folder that already covers it, instead of a bare "already in the sidebar".
func TestOverlappingFolderIsExplained(t *testing.T) {
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(parent, "inner")
	if err := os.MkdirAll(inner, 0700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(t.TempDir(), "config.yaml")

	s := &settings{path: config, systemLanguage: "en-US", values: preferences{Roots: []string{parent}}}
	if s.addRoot(inner) {
		t.Fatal("a folder inside a listed folder was accepted")
	}
	if s.toast == nil || !strings.Contains(s.toast.Title, projectName(parent)) {
		t.Fatalf("nested folder toast: %+v", s.toast)
	}

	// Adding a shortcut to the same folder reports the exact path that matched.
	dup := &settings{path: config, systemLanguage: "en-US", values: preferences{Roots: []string{parent}}}
	if dup.addRoot(parent) {
		t.Fatal("the same folder was accepted twice")
	}
	if dup.toast == nil || dup.toast.Description != parent {
		t.Fatalf("duplicate folder toast: %+v", dup.toast)
	}

	// The reverse overlap names the listed folder the new parent would swallow.
	outer := &settings{path: config, systemLanguage: "en-US", values: preferences{Roots: []string{inner}}}
	if outer.addRoot(parent) {
		t.Fatal("a parent of a listed folder was accepted")
	}
	if outer.toast == nil || !strings.Contains(outer.toast.Title, projectName(inner)) {
		t.Fatalf("parent folder toast: %+v", outer.toast)
	}
}

// A listed folder keeps its row even when every project under it is pinned
// into a collapsed section or removed, so the sidebar never looks empty while
// the folder is still configured.
func TestFolderRowSurvivesHiddenProjects(t *testing.T) {
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(parent, "repo")
	testGit(t, "init", repo)
	closed := false
	s := &settings{statePath: filepath.Join(t.TempDir(), "state.yaml"), systemLanguage: "en-US", values: preferences{Roots: []string{parent}}}
	finishScan(t, s)
	view := ui.NewTester(s.worktreeSidebar, 300, 720)
	view.SetPreferences(ui.Preferences{ReduceMotion: true})

	findProject(s.projectTree, repo).Pinned = true
	s.state.PinnedOpen = &closed
	view.Frame()
	if !view.HasText(projectName(parent)) {
		t.Fatalf("a fully pinned folder lost its row: %q", view.Texts())
	}

	findProject(s.projectTree, repo).Pinned = false
	findProject(s.projectTree, repo).Removed = true
	s.state.PinnedOpen = nil
	view.Frame()
	if !view.HasText(projectName(parent)) {
		t.Fatalf("a folder whose projects were removed lost its row: %q", view.Texts())
	}
}

// The Repositories section lists every repository of every added folder, even
// one pinned or grouped elsewhere, and collapses on its own.
func TestRepositoriesSectionListsAllFolders(t *testing.T) {
	roots, names := everyFolder(t)
	s := &settings{statePath: filepath.Join(t.TempDir(), "state.yaml"), systemLanguage: "en-US", values: preferences{Roots: roots}}
	finishScan(t, s)
	view := ui.NewTester(s.worktreeSidebar, 300, 720)
	view.SetPreferences(ui.Preferences{ReduceMotion: true})
	view.Frame()
	if !view.HasText("Repositories") {
		t.Fatalf("Repositories section missing: %q", view.Texts())
	}
	// It starts collapsed: close the folder rows, so no project could show
	// unless the section were open.
	for _, child := range s.projectTree.Children {
		child.Open = false
	}
	view.Frame()
	if s.state.ReposOpen == nil || *s.state.ReposOpen {
		t.Fatal("the Repositories section should start collapsed")
	}
	for _, name := range names {
		if view.HasText(name) {
			t.Fatalf("%s shows while the section is collapsed: %q", name, view.Texts())
		}
	}
	if err := view.Click("Repositories"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if s.state.ReposOpen == nil || !*s.state.ReposOpen {
		t.Fatal("the Repositories section did not expand")
	}
	for _, name := range names {
		if !view.HasText(name) {
			t.Fatalf("%s is not listed: %q", name, view.Texts())
		}
	}
	// A pinned project leaves the folder tree but stays in the flat list.
	findProject(s.projectTree, filepath.Join(roots[1], names[1])).Pinned = true
	if got := len(allProjects(s.projectTree)); got != len(names) {
		t.Fatalf("allProjects = %d, want %d", got, len(names))
	}
}

func TestRemoveThenReaddFolder(t *testing.T) {
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	testGit(t, "init", filepath.Join(parent, "repo"))
	s := &settings{statePath: filepath.Join(t.TempDir(), "state.yaml"), path: filepath.Join(t.TempDir(), "config.yaml"), systemLanguage: "en-US", values: preferences{Roots: []string{parent}}}
	finishScan(t, s)
	s.removeFolder(s.projectTree.Children[0].Path)
	if len(s.roots()) != 0 {
		t.Fatalf("folder not removed: %v", s.roots())
	}
	if !s.addRoot(parent) {
		t.Fatalf("re-adding the removed folder was refused: %s", s.problem)
	}
	if len(s.roots()) != 1 || !samePath(s.roots()[0], parent) {
		t.Fatalf("re-added folders: %v", s.roots())
	}
}
