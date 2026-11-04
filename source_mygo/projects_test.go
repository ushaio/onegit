package main

import (
	"context"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
)

func testGit(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func TestScanProjects(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(root, "group", "项目 with spaces")
	testGit(t, "init", repo)
	testGit(t, "-C", repo, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "initial")
	linked := filepath.Join(root, "linked")
	testGit(t, "-C", repo, "worktree", "add", "--detach", linked)
	moduleSource := t.TempDir()
	testGit(t, "init", moduleSource)
	testGit(t, "-C", moduleSource, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "initial")
	testGit(t, "-C", repo, "-c", "protocol.file.allow=always", "submodule", "add", moduleSource, "module")
	submodule := filepath.Join(repo, "module")
	nested := filepath.Join(repo, "nested")
	testGit(t, "init", nested)
	bare := filepath.Join(root, "bare.git")
	testGit(t, "init", "--bare", bare)
	empty := filepath.Join(root, "empty")
	testGit(t, "init", empty)
	for _, dir := range []string{filepath.Join(root, "fake", ".git"), filepath.Join(repo, "ordinary"), filepath.Join(root, "unused")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(root, filepath.Join(root, "loop")); err != nil {
		t.Fatal(err)
	}
	// Environment overrides must not cause every directory to become a repo.
	t.Setenv("GIT_DIR", filepath.Join(repo, ".git"))
	t.Setenv("GIT_WORK_TREE", repo)
	tree, err := scanProjects(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]*projectNode{}
	var collect func(*projectNode)
	collect = func(n *projectNode) {
		if n == nil {
			return
		}
		found[n.Path] = n
		for _, child := range n.Children {
			collect(child)
		}
	}
	collect(tree)
	for _, path := range []string{repo, linked, bare, empty} {
		if found[path] == nil || !found[path].Repository {
			t.Fatalf("missing repository %s: %+v", path, found)
		}
	}
	if len(found) != 6 || !found[bare].Bare {
		t.Fatalf("unexpected tree: %+v", found)
	}
	for _, path := range []string{nested, submodule} {
		if found[path] != nil {
			t.Fatalf("scanned inside repository: %s", path)
		}
	}
	// The selected root itself can be a repository, including a submodule.
	for _, path := range []string{repo, submodule} {
		got, err := scanProjects(context.Background(), path)
		if err != nil || got == nil || !got.Repository || len(got.Children) != 0 {
			t.Fatalf("repository root should be a leaf: %+v, %v", got, err)
		}
	}
	// Selecting an ordinary subdirectory must not discover its ancestor repo.
	if got, err := scanProjects(context.Background(), filepath.Join(repo, "ordinary")); err != nil || got != nil {
		t.Fatalf("ordinary directory: %+v, %v", got, err)
	}
}

func TestScanFailuresAndEmpty(t *testing.T) {
	root := t.TempDir()
	if tree, err := scanProjects(context.Background(), root); err != nil || tree != nil {
		t.Fatalf("empty: %+v, %v", tree, err)
	}
	if _, err := scanProjects(context.Background(), filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing root accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scanProjects(ctx, root); err == nil {
		t.Fatal("cancellation ignored")
	}
	t.Setenv("PATH", "")
	if _, err := scanProjects(context.Background(), root); err == nil {
		t.Fatal("missing git ignored")
	}
}

func TestProjectTreeInteraction(t *testing.T) {
	leaf := &projectNode{Path: "/projects/group/repo", Repository: true, Open: true}
	group := &projectNode{Path: "/projects/group", Open: true, Children: []*projectNode{leaf}}
	s := &settings{systemLanguage: "zh-CN", values: preferences{Roots: []string{"/projects"}}, state: sidebarState{Roots: map[string]*projectNode{
		"/projects": {Path: "/projects", Open: true, Children: []*projectNode{group, {Path: "/projects/repo", Repository: true}}},
	}}}
	s.rebuildTree()
	tt := ui.NewTester(s.worktreeSidebar, 300, 720)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame() // Repeated leaf names must have independent identities.
	if err := tt.Click("group"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyLeft)
	tt.Frame()
	if group.Open {
		t.Fatal("keyboard collapse failed")
	}
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if !group.Open {
		t.Fatal("keyboard expansion failed")
	}
	if err := tt.Click("repo"); err != nil {
		t.Fatal(err)
	}
	if s.selectedProject == nil || !s.selectedProject.Repository {
		t.Fatal("project selection failed")
	}
	content := ui.NewTester(s.welcome, 800, 600)
	content.Frame()
	if content.HasText(s.selectedProject.Path) {
		t.Fatal("project path should not appear in the compact worktree list")
	}
}

func TestProjectContextMenu(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	alpha := filepath.Join(root, "alpha")
	beta := filepath.Join(root, "beta")
	group := filepath.Join(root, "group")
	for _, repo := range []string{alpha, beta, filepath.Join(group, "nested")} {
		testGit(t, "init", repo)
	}
	s := &settings{statePath: filepath.Join(t.TempDir(), "state.yaml"), values: preferences{Roots: []string{root}}, systemLanguage: "en-US"}
	callbacks := make(chan func(), 64)
	s.startProjectScan(func(fn func()) { callbacks <- fn })
	awaitNavigation(t, s, callbacks)
	if s.selectedProject == nil || s.selectedProject.Path != alpha {
		t.Fatalf("initial selection: %+v", s.selectedProject)
	}
	opened := make(chan string, 2)
	s.openPath = func(path string) error {
		opened <- path
		return nil
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			s.worktreeSidebar(c)
			components.Toasts(c, components.ToastOptions{})
		})
	}, 500, 720)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()

	if err := tt.RightClick("group"); err != nil {
		t.Fatal(err)
	}
	if menu := tt.Menu(); len(menu) != 1 || menu[0] != "Open Folder" {
		t.Fatalf("folder menu: %q", menu)
	}
	if err := tt.ChooseMenuItem("Open Folder"); err != nil {
		t.Fatal(err)
	}
	select {
	case path := <-opened:
		if path != group {
			t.Fatalf("opened %q", path)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("folder was not opened")
	}

	if err := tt.RightClick("alpha"); err != nil {
		t.Fatal(err)
	}
	menu := tt.Menu()
	if len(menu) != 9 || menu[0] != "Open Folder" || menu[1] != "-" || menu[2] != "Open Terminal" || menu[3] != "Pin Project" || menu[4] != "Move to Group" || menu[5] != "-" || menu[6] != "Choose Icon…" || menu[7] != "-" || menu[8] != "Remove Project" {
		t.Fatalf("project menu: %q", menu)
	}
	if err := tt.ChooseMenuItem("Remove Project"); err != nil {
		t.Fatal(err)
	}
	awaitNavigation(t, s, callbacks)

	removed := findProject(s.projectTree, alpha)
	if removed == nil || !removed.Removed {
		t.Fatalf("project was not removed: %+v", removed)
	}
	if _, _, visible := sidebarProjects(s.projectTree, nil); visible[removed] {
		t.Fatal("removed project is still visible")
	}
	if s.selectedProject == nil || s.selectedProject.Path != beta {
		t.Fatalf("selection after removal: %+v", s.selectedProject)
	}
	if s.state.SelectedProjects[globalRoot] != beta {
		t.Fatalf("saved selection after removal: %q", s.state.SelectedProjects[globalRoot])
	}
	saved, err := loadSidebarState(s.statePath)
	if err != nil {
		t.Fatal(err)
	}
	if node := findProject(saved.Roots[root], alpha); node == nil || !node.Removed {
		t.Fatalf("removal was not saved: %+v", node)
	}
	for _, item := range s.commandItems() {
		if item.ID == "project/"+alpha {
			t.Fatal("command palette lists the removed project")
		}
	}

	// The toast undoes the removal.
	tt.Frame()
	if !tt.HasText("Project removed") {
		t.Fatalf("removal toast is missing: %q", tt.Texts())
	}
	if err := tt.Click("Undo"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if removed.Removed || !tt.HasText("alpha") {
		t.Fatal("undo did not restore the project")
	}

	// The palette restores every project removed from the parent directory.
	if err := tt.RightClick("alpha"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Remove Project"); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range s.commandItems() {
		found = found || item.ID == "projects/restore"
	}
	if !found {
		t.Fatal("restore command is missing")
	}
	s.activateCommand("projects/restore")
	tt.Frame()
	if removed.Removed || !tt.HasText("alpha") {
		t.Fatal("restore command did not restore the project")
	}
}

func TestScanCompletionIgnoresOldDirectory(t *testing.T) {
	first := t.TempDir()
	testGit(t, "init", first)
	s := &settings{values: preferences{Roots: []string{first}}}
	callbacks := make(chan func(), 2)
	s.startProjectScan(func(fn func()) { callbacks <- fn })
	var oldCompletion func()
	select {
	case oldCompletion = <-callbacks:
	case <-time.After(5 * time.Second):
		t.Fatal("first scan did not complete")
	}
	s.values.Roots = []string{t.TempDir()}
	s.startProjectScan(func(fn func()) { callbacks <- fn })
	select {
	case fn := <-callbacks:
		fn()
	case <-time.After(5 * time.Second):
		t.Fatal("scan did not complete")
	}
	oldCompletion()
	found := 0
	walkProjects(s.projectTree, func(node *projectNode) {
		if node.Repository {
			found++
		}
	})
	if s.scanning || found != 0 || s.scanProblem != "" {
		t.Fatalf("unexpected completion: %d repositories, %q", found, s.scanProblem)
	}
}

func TestProjectIcon(t *testing.T) {
	root := t.TempDir()
	repo := &projectNode{Path: filepath.Join(root, "repo"), Repository: true}
	logo := filepath.Join(repo.Path, "logo.png")
	broken := filepath.Join(repo.Path, "broken.png")
	if err := os.MkdirAll(repo.Path, 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(logo)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := os.WriteFile(broken, []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	callbacks := make(chan func(), 4)
	s := &settings{
		statePath: filepath.Join(t.TempDir(), "state.yaml"), systemLanguage: "en-US",
		values: preferences{Roots: []string{root}},
		state:  sidebarState{Roots: map[string]*projectNode{root: {Path: root, Open: true, Children: []*projectNode{repo}}}},
	}
	s.rebuildTree()
	s.repo.dispatch = func(fn func()) { callbacks <- fn }
	opened := make(chan mygo.OpenDialogOptions, 2)
	choice := logo
	s.openDialog = func(opts mygo.OpenDialogOptions) ([]string, error) {
		opened <- opts
		return []string{choice}, nil
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			s.worktreeSidebar(c)
			components.Toasts(c, components.ToastOptions{})
		})
	}, 500, 720)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	choose := func() {
		t.Helper()
		if err := tt.RightClick("repo"); err != nil {
			t.Fatal(err)
		}
		if err := tt.ChooseMenuItem("Choose Icon…"); err != nil {
			t.Fatal(err)
		}
		select {
		case opts := <-opened:
			if opts.DefaultPath != repo.Path || opts.Directory || opts.Multiple || len(opts.Filters) != 1 {
				t.Fatalf("picker options: %+v", opts)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("picker did not open")
		}
		select {
		case fn := <-callbacks:
			fn()
		case <-time.After(5 * time.Second):
			t.Fatal("picker did not finish")
		}
		tt.Frame()
	}

	// An image that does not decode leaves the placeholder.
	choice = broken
	choose()
	if repo.Icon != "" || !tt.HasText("Could not use this image") {
		t.Fatalf("broken image: icon %q, texts %q", repo.Icon, tt.Texts())
	}

	choice = logo
	choose()
	if repo.Icon != logo || s.projectIcons[logo] == nil || s.projectImage(repo) == nil {
		t.Fatalf("icon was not applied: %q", repo.Icon)
	}
	saved, err := loadSidebarState(s.statePath)
	if err != nil {
		t.Fatal(err)
	}
	if node := findProject(saved.Roots[root], repo.Path); node == nil || node.Icon != logo {
		t.Fatalf("icon was not saved: %+v", node)
	}

	// A restart reads the saved icon in the background.
	s.projectIcons = nil
	if s.projectImage(repo) != nil {
		t.Fatal("icon should load in the background")
	}
	select {
	case fn := <-callbacks:
		fn()
	case <-time.After(5 * time.Second):
		t.Fatal("icon did not load")
	}
	if s.projectImage(repo) == nil {
		t.Fatal("saved icon was not loaded")
	}

	if err := tt.RightClick("repo"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Reset Icon"); err != nil {
		t.Fatal(err)
	}
	if repo.Icon != "" || s.projectImage(repo) != nil {
		t.Fatal("icon was not reset")
	}
}
