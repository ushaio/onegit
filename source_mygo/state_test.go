package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func finishScan(t *testing.T, s *settings) {
	t.Helper()
	callbacks := make(chan func(), 64)
	s.startProjectScan(func(fn func()) { callbacks <- fn })
	awaitNavigation(t, s, callbacks)
}

func awaitNavigation(t *testing.T, s *settings, callbacks <-chan func()) {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for s.scanning || s.overview.Loading || s.repositoryLoading() {
		select {
		case fn := <-callbacks:
			fn()
		case <-deadline:
			t.Fatal("navigation timed out")
		}
	}
}

func findProject(root *projectNode, path string) *projectNode {
	var found *projectNode
	walkProjects(root, func(n *projectNode) {
		if n.Path == path {
			found = n
		}
	})
	return found
}

func TestSidebarRestartAndMerge(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(root, "group", "old")
	testGit(t, "init", repo)
	testGit(t, "init", filepath.Join(root, "group", "kept"))
	gone := filepath.Join(root, "gone", "repo")
	testGit(t, "init", gone)
	path := filepath.Join(t.TempDir(), "state.yaml")
	s := &settings{statePath: path, values: preferences{Roots: []string{root}}, systemLanguage: "zh-CN"}
	finishScan(t, s)
	group := findProject(s.projectTree, filepath.Dir(repo))
	if group == nil || group.Open {
		t.Fatal("new group must be collapsed")
	}
	tt := ui.NewTester(s.worktreeSidebar, 300, 720)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if err := tt.Click("group"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	restored, err := loadSidebarState(path)
	if err != nil || !findProject(restored.Roots[root], group.Path).Open {
		t.Fatal("keyboard expansion not persisted", err)
	}
	tt.Key(0, ui.KeyLeft)
	tt.Frame()
	restored, err = loadSidebarState(path)
	if err != nil || findProject(restored.Roots[root], group.Path).Open {
		t.Fatal("keyboard collapse not persisted", err)
	}
	findProject(s.projectTree, repo).Pinned = true
	s.state.SelectedProjects = map[string]string{globalRoot: repo}
	if err := s.saveSidebarState(); err != nil {
		t.Fatal(err)
	}
	if restored, err = loadSidebarState(path); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(repo); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	added := filepath.Join(root, "new-group", "new")
	testGit(t, "init", added)
	restarted := &settings{state: restored, statePath: path, values: s.values, systemLanguage: "zh-CN"}
	callbacks := make(chan func(), 1)
	restarted.startProjectScan(func(fn func()) { callbacks <- fn })
	if findProject(restarted.projectTree, repo) == nil {
		t.Fatal("cached tree not immediately restored")
	}
	// Expand while the worker is running; completion must retain the change.
	findProject(restarted.projectTree, group.Path).Open = true
	awaitNavigation(t, restarted, callbacks)
	if findProject(restarted.projectTree, repo) != nil || findProject(restarted.projectTree, filepath.Dir(gone)) != nil {
		t.Fatal("missing project or its emptied folder kept")
	}
	if findProject(restarted.projectTree, added) == nil {
		t.Fatal("merge lost new project")
	}
	if restarted.selectedProject == nil || restarted.selectedProject.Path == repo {
		t.Fatal("selection did not fall back from the missing project")
	}
	if !findProject(restarted.projectTree, group.Path).Open || findProject(restarted.projectTree, filepath.Dir(added)).Open {
		t.Fatal("merge changed expansion")
	}
	testGit(t, "init", repo)
	finishScan(t, restarted)
	if node := findProject(restarted.projectTree, repo); node == nil || node.Missing || node.Pinned {
		t.Fatal("reappearing project not listed afresh")
	}
	// A missing parent directory, like an unmounted volume, keeps the cache.
	if err := os.Rename(root, root+"-moved"); err != nil {
		t.Fatal(err)
	}
	finishScan(t, restarted)
	if err := os.Rename(root+"-moved", root); err != nil {
		t.Fatal(err)
	}
	if restarted.scanProblem == "" || findProject(restarted.projectTree, repo) == nil {
		t.Fatal("missing parent directory dropped the cache")
	}
	// A scan failure must preserve the cache without marking existing paths missing.
	t.Setenv("PATH", "")
	finishScan(t, restarted)
	if restarted.scanProblem == "" || findProject(restarted.projectTree, repo).Missing {
		t.Fatal("failure lost cached project")
	}
	other := t.TempDir()
	restarted.values.Roots = []string{other}
	finishScan(t, restarted)
	if findProject(restarted.projectTree, repo) != nil {
		t.Fatal("cache leaked across roots")
	}
	restarted.values.Roots = []string{root}
	finishScan(t, restarted)
	if findProject(restarted.projectTree, repo) == nil {
		t.Fatal("switching back lost cache")
	}
}

func TestMergeKeepsRemovedProjects(t *testing.T) {
	old := &projectNode{Path: "/projects", Children: []*projectNode{{Path: "/projects/repo", Repository: true, Removed: true}}}
	fresh := &projectNode{Path: "/projects", Children: []*projectNode{{Path: "/projects/repo", Repository: true}}}
	merged := mergeProjects(old, fresh, map[string]bool{})
	if len(merged.Children) != 1 || !merged.Children[0].Removed {
		t.Fatalf("scan restored a removed project: %+v", merged.Children)
	}
}

func TestStateFilesAndMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	legacy := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(legacy, []byte(`{"projectParentDir":"/projects","theme":"dark","language":"en"}`), 0600); err != nil {
		t.Fatal(err)
	}
	p, err := loadOrMigratePreferences(path, legacy)
	if err != nil || len(p.Roots) != 1 || p.Roots[0] != "/projects" || p.Theme != "dark" || p.Language != "en" {
		t.Fatal("migration failed", p, err)
	}
	p.Language = "zh-Hans"
	if err := savePreferences(path, p); err != nil {
		t.Fatal(err)
	}
	got, err := loadOrMigratePreferences(path, legacy)
	if err != nil || got.Language != "zh-Hans" || len(got.Roots) != 1 || got.Roots[0] != "/projects" {
		t.Fatal("legacy overwrote new preferences", got, err)
	}
	statePath := filepath.Join(dir, "state.yaml")
	if _, err := loadSidebarState(statePath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte("roots: ["), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSidebarState(statePath); err == nil {
		t.Fatal("malformed state accepted")
	}
	blocked := filepath.Join(dir, "file")
	if err := os.WriteFile(blocked, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &settings{statePath: filepath.Join(blocked, "state.yaml")}
	s.saveSidebarState()
	if s.stateProblem == "" {
		t.Fatal("save failure hidden")
	}
	data, err := os.ReadFile(blocked)
	if err != nil || string(data) != "keep" {
		t.Fatal("save failure damaged file")
	}
}

func TestPinnedProjectGroupsAndRestore(t *testing.T) {
	pinned := &projectNode{Path: "/projects/group/repo", Repository: true}
	other := &projectNode{Path: "/projects/other", Repository: true}
	group := &projectNode{Path: "/projects/group", Open: true, savedOpen: true, Children: []*projectNode{pinned}}
	root := &projectNode{Path: "/projects", Open: true, Children: []*projectNode{group, other}}
	s := &settings{statePath: filepath.Join(t.TempDir(), "state.yaml"), values: preferences{Roots: []string{root.Path}}, state: sidebarState{Roots: map[string]*projectNode{root.Path: root}}, systemLanguage: "zh-CN"}
	s.rebuildTree()
	view := ui.NewTester(s.worktreeSidebar, 300, 720)
	view.SetPreferences(ui.Preferences{ReduceMotion: true})
	view.Frame()
	if err := view.Click("repo"); err != nil {
		t.Fatal(err)
	}
	if err := view.RightClick("repo"); err != nil {
		t.Fatal(err)
	}
	if err := view.ChooseMenuItem("置顶项目"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if !pinned.Pinned || s.selectedProject != pinned || view.HasText("group") || !view.HasText("repo") || !view.HasText("other") || !view.HasText("置顶") || !view.HasText("未分组项目") {
		t.Fatal("pinning did not move the project out of its hierarchy", view.Texts())
	}
	pins, _, visible := sidebarProjects(root, nil)
	if len(pins) != 1 || pins[0] != pinned || visible[pinned] || visible[group] || !visible[other] || !group.Open {
		t.Fatal("incorrect projection or expansion changed")
	}
	state, err := loadSidebarState(s.statePath)
	if err != nil {
		t.Fatal(err)
	}
	restored := findProject(state.Roots[root.Path], pinned.Path)
	if restored == nil || !restored.Pinned {
		t.Fatal("pin was not persisted")
	}
	fresh := &projectNode{Path: root.Path, Children: []*projectNode{{Path: group.Path, Children: []*projectNode{{Path: pinned.Path, Repository: true}}}}}
	merged := mergeProjects(state.Roots[root.Path], fresh, nil)
	if !findProject(merged, pinned.Path).Pinned {
		t.Fatal("scan discarded pinned state")
	}
	if err := view.RightClick("repo"); err != nil {
		t.Fatal(err)
	}
	if err := view.ChooseMenuItem("取消置顶"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if pinned.Pinned || !view.HasText("group") || !view.HasText("repo") || s.selectedProject != pinned {
		t.Fatal("unpin did not restore hierarchy and selection", view.Texts())
	}
	state, err = loadSidebarState(s.statePath)
	if err != nil || findProject(state.Roots[root.Path], pinned.Path).Pinned {
		t.Fatal("unpin was not persisted", err)
	}
}

func TestPinnedProjectionIgnoresCollapsedAncestors(t *testing.T) {
	first := &projectNode{Path: "/projects/a/same", Repository: true, Pinned: true, Missing: true}
	second := &projectNode{Path: "/projects/b/same", Repository: true, Pinned: true}
	root := &projectNode{Path: "/projects", Children: []*projectNode{
		{Path: "/projects/a", Children: []*projectNode{first}},
		{Path: "/projects/b", Children: []*projectNode{second}},
	}}
	pins, _, visible := sidebarProjects(root, nil)
	if len(pins) != 2 || pins[0] != first || pins[1] != second || visible[root] {
		t.Fatal("collapsed, missing or same-name projects lost")
	}
	pins, _, visible = sidebarProjects(second, nil)
	if len(pins) != 1 || visible[second] {
		t.Fatal("repository root not pinned")
	}
	second.Pinned = false
	pins, _, visible = sidebarProjects(second, nil)
	if len(pins) != 0 || !visible[second] {
		t.Fatal("repository root not restored")
	}
}

func TestSidebarGroupCollapseAndRestore(t *testing.T) {
	pin := &projectNode{Path: "/projects/pin", Repository: true, Pinned: true}
	repo := &projectNode{Path: "/projects/group/repo", Repository: true}
	group := &projectNode{Path: "/projects/group", Open: true, savedOpen: true, Children: []*projectNode{repo}}
	root := &projectNode{Path: "/projects", Open: true, Children: []*projectNode{pin, group}}
	s := &settings{statePath: filepath.Join(t.TempDir(), "state.yaml"), selectedProject: repo, values: preferences{Roots: []string{root.Path}}, state: sidebarState{Roots: map[string]*projectNode{root.Path: root}}, systemLanguage: "zh-CN"}
	s.rebuildTree()
	view := ui.NewTester(s.worktreeSidebar, 300, 720)
	view.SetPreferences(ui.Preferences{ReduceMotion: true})
	view.Frame()
	if !view.HasText("pin") || !view.HasText("repo") {
		t.Fatal("groups should be expanded by default", view.Texts())
	}
	if err := view.Click("置顶"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if view.HasText("pin") || !view.HasText("repo") || *s.state.PinnedOpen || !*s.state.OtherOpen {
		t.Fatal("pinned collapse affected other group", view.Texts())
	}
	if err := view.Click("未分组项目"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if view.HasText("repo") || view.HasText("group") || *s.state.OtherOpen || !group.Open || s.selectedProject != repo {
		t.Fatal("collapse lost project state", view.Texts())
	}
	state, err := loadSidebarState(s.statePath)
	if err != nil || state.PinnedOpen == nil || *state.PinnedOpen || state.OtherOpen == nil || *state.OtherOpen {
		t.Fatal("group collapse was not persisted", err)
	}
	restarted := &settings{state: state, statePath: s.statePath, values: s.values, systemLanguage: s.systemLanguage}
	restarted.rebuildTree()
	view = ui.NewTester(restarted.worktreeSidebar, 300, 720)
	view.SetPreferences(ui.Preferences{ReduceMotion: true})
	view.Frame()
	if view.HasText("pin") || view.HasText("repo") {
		t.Fatal("restart did not restore collapse", view.Texts())
	}
	if err := view.Click("未分组项目"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if !view.HasText("repo") || view.HasText("pin") {
		t.Fatal("expand did not restore nested projects", view.Texts())
	}
	// The clicked disclosure keeps keyboard focus; Space toggles it again.
	view.Key(0, ui.KeySpace)
	view.Frame()
	if *restarted.state.OtherOpen || view.HasText("repo") {
		t.Fatal("keyboard collapse failed", view.Texts())
	}
}

func TestWindowStateStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.yaml")
	s := &settings{statePath: path, state: sidebarState{SidebarWidth: 260, SidebarCollapsed: true}}
	store := windowStateStore{s}
	if states, err := store.LoadWindowStates(); err != nil || states != nil {
		t.Fatal("empty state restored a window", states, err)
	}
	saved := mygo.WindowState{X: 10, Y: 20, Width: 1200, Height: 800, Maximized: true}
	if err := store.SaveWindowStates(map[string]mygo.WindowState{"other": {Width: 1}, mainWindowStateKey: saved}); err != nil {
		t.Fatal(err)
	}
	state, err := loadSidebarState(path)
	if err != nil || state.Window == nil || *state.Window != saved || state.SidebarWidth != 260 || !state.SidebarCollapsed {
		t.Fatal("window state not saved with the sidebar layout", state, err)
	}
	states, err := windowStateStore{&settings{state: state}}.LoadWindowStates()
	if err != nil || len(states) != 1 || states[mainWindowStateKey] != saved {
		t.Fatal("window state not restored", states, err)
	}
}

func TestDataDirMigration(t *testing.T) {
	home := t.TempDir()
	dir, legacy := filepath.Join(home, ".onegit"), filepath.Join(home, ".oneo", "onegit")
	if err := migrateDataDir(dir, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("created a data directory without legacy data", err)
	}
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "state.yaml"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := migrateDataDir(dir, legacy); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "state.yaml")); err != nil || string(data) != "old" {
		t.Fatal("legacy data not moved", string(data), err)
	}
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "state.yaml"), []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := migrateDataDir(dir, legacy); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "state.yaml")); string(data) != "old" {
		t.Fatal("legacy data overwrote the data directory")
	}
}

func TestProjectGroupsMoveReorderAndRestore(t *testing.T) {
	alpha := &projectNode{Path: "/projects/alpha", Repository: true}
	beta := &projectNode{Path: "/projects/nested/beta", Repository: true}
	nested := &projectNode{Path: "/projects/nested", Open: true, savedOpen: true, Children: []*projectNode{beta}}
	root := &projectNode{Path: "/projects", Open: true, Children: []*projectNode{alpha, nested}}
	s := &settings{statePath: filepath.Join(t.TempDir(), "state.yaml"), values: preferences{Roots: []string{root.Path}}, state: sidebarState{Roots: map[string]*projectNode{root.Path: root}}, systemLanguage: "en-US"}
	s.rebuildTree()
	view := ui.NewTester(s.worktreeSidebar, 300, 720)
	view.SetPreferences(ui.Preferences{ReduceMotion: true})
	view.Frame()
	if !view.HasText("Ungrouped Projects") || !view.HasText("alpha") || !view.HasText("beta") {
		t.Fatal("projects should start ungrouped", view.Texts())
	}
	// Headings count their projects, not folders, and empty groups show none.
	if !view.HasText("2") || view.HasText("0") {
		t.Fatal("ungrouped count wrong", view.Texts())
	}

	// A group made from a project's menu takes the project.
	if err := view.RightClick("beta"); err != nil {
		t.Fatal(err)
	}
	if err := view.ChooseMenuItem("Move to Group", "New Group…"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	view.Type("Work")
	view.Key(0, ui.KeyEnter)
	view.Frame()
	if s.groupDialog.open || beta.Group == "" || !view.HasText("Work") || view.HasText("nested") {
		t.Fatal("new group did not take the project", beta.Group, view.Texts())
	}
	if view.HasText("2") || !view.HasText("1") {
		t.Fatal("group counts not updated", view.Texts())
	}
	if err := view.RightClick("Ungrouped Projects"); err != nil {
		t.Fatal(err)
	}
	if err := view.ChooseMenuItem("New Group…"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	view.Type("Later")
	view.Key(0, ui.KeyEnter)
	view.Frame()
	if err := view.RightClick("alpha"); err != nil {
		t.Fatal(err)
	}
	if err := view.ChooseMenuItem("Move to Group", "Later"); err != nil {
		t.Fatal(err)
	}
	order := func(groups []*projectGroup) string {
		var names []string
		for _, group := range groups {
			names = append(names, group.ID+"="+group.Name)
		}
		return strings.Join(names, ",")
	}
	if got := order(s.projectGroups()); got != "g1=Work,g2=Later,ungrouped=" || alpha.Group != "g2" {
		t.Fatal("unexpected groups", got, alpha.Group)
	}

	// Dragging a header onto another group moves it there, in either direction.
	drag := func(from, to string) {
		t.Helper()
		view.Frame()
		src, ok := view.Find(from)
		target, found := view.Find(to)
		if !ok || !found {
			t.Fatal("missing group header", from, to)
		}
		view.Press(src.X+5, src.Y+src.H/2)
		for k := 1; k <= 4; k++ {
			f := float32(k) / 4
			view.Move(src.X+5, src.Y+src.H/2+(target.Y-src.Y)*f)
		}
		view.Release(src.X+5, target.Y+target.H/2)
		view.Frame()
	}
	drag("Ungrouped Projects", "Work")
	if got := order(s.projectGroups()); got != "ungrouped=,g1=Work,g2=Later" {
		t.Fatal("drag up did not reorder", got)
	}
	drag("Work", "Later")
	if got := order(s.projectGroups()); got != "ungrouped=,g2=Later,g1=Work" {
		t.Fatal("drag down did not reorder", got)
	}
	drag("Pinned", "Work")
	if got := order(s.projectGroups()); got != "ungrouped=,g2=Later,g1=Work" {
		t.Fatal("pinned group moved", got)
	}

	if err := view.RightClick("Work"); err != nil {
		t.Fatal(err)
	}
	if err := view.ChooseMenuItem("Rename Group…"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if s.groupDialog.name != "Work" {
		t.Fatalf("rename starts from %q", s.groupDialog.name)
	}
	s.groupDialog.name = "Client"
	s.submitGroupDialog()
	view.Frame()
	if !view.HasText("Client") || view.HasText("Work") || beta.Group != "g1" {
		t.Fatal("rename lost the group's projects", view.Texts())
	}

	// A restart restores the order, names and members.
	state, err := loadSidebarState(s.statePath)
	if err != nil {
		t.Fatal(err)
	}
	restarted := &settings{state: state, statePath: s.statePath, values: s.values, systemLanguage: s.systemLanguage}
	restarted.rebuildTree()
	if got := order(restarted.projectGroups()); got != "ungrouped=,g2=Later,g1=Client" || findProject(restarted.projectTree, beta.Path).Group != "g1" {
		t.Fatal("groups were not persisted", got)
	}

	// Deleting a group returns its projects to Ungrouped Projects.
	if err := view.RightClick("Client"); err != nil {
		t.Fatal(err)
	}
	if err := view.ChooseMenuItem("Delete Group"); err != nil {
		t.Fatal(err)
	}
	view.Frame()
	if beta.Group != "" || view.HasText("Client") || !view.HasText("nested") || !view.HasText("beta") {
		t.Fatal("deleted group kept its projects", view.Texts())
	}
	if err := view.RightClick("Ungrouped Projects"); err != nil {
		t.Fatal(err)
	}
	if menu := view.Menu(); len(menu) != 1 || menu[0] != "New Group…" {
		t.Fatalf("ungrouped menu: %q", menu)
	}
}
