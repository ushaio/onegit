package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestTerminalDir(t *testing.T) {
	s := &settings{}
	if got := s.terminalDir(); got != "" {
		t.Fatalf("no repository: %q", got)
	}
	repo := filepath.Join(t.TempDir(), "repo")
	s.selectedProject = &projectNode{Path: repo, Repository: true}
	if got := s.terminalDir(); got != filepath.Clean(repo) {
		t.Fatalf("terminal dir: %q", got)
	}
	s.selectedProject = &projectNode{Path: repo}
	if got := s.terminalDir(); got != "" {
		t.Fatalf("a folder is not a repository: %q", got)
	}
}

func TestTerminalKey(t *testing.T) {
	got := terminalKey(filepath.Join("a", "b", "..", "c"))
	want := filepath.Clean(filepath.Join("a", "c"))
	if runtime.GOOS == "windows" {
		want = strings.ToLower(want)
	}
	if got != want {
		t.Fatalf("terminal key: %q want %q", got, want)
	}
}

// Switching the content tabs changes the page, and the terminal session lives
// outside the page so switching back does not restart it.
func TestPageTabsSwitch(t *testing.T) {
	s := &settings{systemLanguage: "en"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { s.pageTabSwitch(c) })
	}, 400, 80)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	if s.content != contentGit {
		t.Fatalf("initial page: %v", s.content)
	}
	if err := tt.Click("Terminal"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if s.content != contentTerminal {
		t.Fatalf("Terminal tab did not switch the page: %v", s.content)
	}
	if err := tt.Click("Git"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if s.content != contentGit {
		t.Fatalf("Git tab did not switch the page: %v", s.content)
	}
}

// A shortcut made in the dialog is saved to state.yaml and listed again.
func TestTerminalPresetDialogSaves(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	s := &settings{systemLanguage: "en", statePath: filepath.Join(t.TempDir(), "state.yaml")}
	s.selectedProject = &projectNode{Path: repo, Repository: true}
	key := terminalKey(repo)
	s.openTerminalDialog(key, -1)
	if s.terminalDialog.dir != filepath.Clean(repo) {
		t.Fatalf("new shortcut starts at the repository: %q", s.terminalDialog.dir)
	}
	s.terminalDialog.name = " dev "
	s.terminalDialog.command = " npm run dev "
	preset, created := s.submitTerminalDialog()
	if !created || preset == nil {
		t.Fatalf("the first shortcut should be new: %v %v", created, preset)
	}
	list := s.terminalPresets(key)
	if len(list) != 1 || list[0].Name != "dev" || list[0].Command != "npm run dev" {
		t.Fatalf("saved shortcut: %+v", list)
	}
	data, err := os.ReadFile(s.statePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "npm run dev") {
		t.Fatalf("state.yaml does not hold the shortcut:\n%s", data)
	}
}

// Editing a shortcut replaces it, and deleting one empties the list.
func TestTerminalPresetEditAndDelete(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	s := &settings{systemLanguage: "en", statePath: filepath.Join(t.TempDir(), "state.yaml")}
	s.selectedProject = &projectNode{Path: repo, Repository: true}
	key := terminalKey(repo)
	s.state.TerminalPresets = map[string][]*terminalPreset{key: {
		{Name: "dev", Command: "npm run dev", Dir: repo},
	}}
	s.openTerminalDialog(key, 0)
	s.terminalDialog.command = "npm run build"
	if preset, created := s.submitTerminalDialog(); created || preset == nil {
		t.Fatalf("editing should not create: %v %v", created, preset)
	}
	if list := s.terminalPresets(key); len(list) != 1 || list[0].Command != "npm run build" {
		t.Fatalf("edited shortcut: %+v", list)
	}
	s.deleteTerminalPreset(key, 0)
	if list := s.terminalPresets(key); len(list) != 0 {
		t.Fatalf("deleted shortcut remains: %+v", list)
	}
}

// The active tab is clamped to the open terminals.
func TestTerminalActiveTab(t *testing.T) {
	s := &settings{}
	key := terminalKey(t.TempDir())
	s.terminalTabs = map[string][]*terminalTab{key: {{}, {}}}
	s.terminalActive = map[string]int{key: 5}
	if got := s.terminalActiveIndex(key); got != 0 {
		t.Fatalf("active index: %d", got)
	}
	s.terminalActive[key] = 1
	if tab := s.activeTerminal(key); tab != s.terminalTabs[key][1] {
		t.Fatalf("active tab: %+v", tab)
	}
}

// The terminal list shows saved shortcuts, and its New Terminal button opens
// the form.
func TestTerminalListOpensDialog(t *testing.T) {
	s := &settings{systemLanguage: "en"}
	key := terminalKey(t.TempDir())
	s.state.TerminalPresets = map[string][]*terminalPreset{key: {
		{Name: "dev", Command: "npm run dev"},
	}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() { s.terminalList(c, key) })
	}, 320, 200)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	if !tt.HasText("dev") || !tt.HasText("npm run dev") {
		t.Fatal("the list does not show the shortcut")
	}
	if err := tt.Click("New Terminal"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !s.terminalDialog.open {
		t.Fatal("New Terminal did not open the form")
	}
}

// Adding or replacing a terminal from settings that never opened one
// initializes the maps instead of panicking on a nil one.
func TestTerminalTabMapsInitialize(t *testing.T) {
	s := &settings{}
	key := terminalKey(t.TempDir())
	first := &terminalTab{}
	s.addTerminalTab(key, first)
	if got := s.terminalActive[key]; got != 0 || len(s.terminalTabs[key]) != 1 {
		t.Fatalf("after add: active %d tabs %d", got, len(s.terminalTabs[key]))
	}
	s.addTerminalTab(key, &terminalTab{})
	replacement := &terminalTab{}
	s.replaceTerminalTab(key, 0, replacement)
	if len(s.terminalTabs[key]) != 2 || s.terminalTabs[key][0] != replacement {
		t.Fatalf("after replace: %+v", s.terminalTabs[key])
	}
}

// Dragging a tab past its neighbor's center moves it into that slot and makes
// it the shown tab.
func TestTerminalReorderTab(t *testing.T) {
	s := &settings{}
	key := "repo"
	first := &terminalTab{id: 1}
	second := &terminalTab{id: 2}
	third := &terminalTab{id: 3}
	s.terminalTabs = map[string][]*terminalTab{key: {first, second, third}}
	s.terminalActive = map[string]int{key: 0}
	s.terminalTabBounds = map[int]ui.Rect{
		1: {X: 0, W: 100},
		2: {X: 100, W: 100},
		3: {X: 200, W: 100},
	}
	s.terminalDrag = map[int]*ui.Point{1: {}}
	s.reorderTerminalTab(key, 0, ui.Point{X: 150})
	got := s.terminalTabs[key]
	if len(got) != 3 || got[0] != second || got[1] != first || got[2] != third {
		t.Fatalf("reordered tabs: %+v", got)
	}
	if s.terminalActive[key] != 1 {
		t.Fatalf("active tab after reorder: %d", s.terminalActive[key])
	}
	if *s.terminalDrag[1] != (ui.Point{}) {
		t.Fatalf("drag offset not reset: %+v", *s.terminalDrag[1])
	}
}

// The tab menu closes the other tabs, or the ones to the right, and keeps the
// shown tab valid.
func TestTerminalCloseTabMenus(t *testing.T) {
	s := &settings{}
	key := "repo"
	first := &terminalTab{id: 1}
	second := &terminalTab{id: 2}
	third := &terminalTab{id: 3}
	s.terminalTabs = map[string][]*terminalTab{key: {first, second, third}}
	s.terminalActive = map[string]int{key: 1}
	s.closeTerminalTabsAfter(key, 0)
	if got := s.terminalTabs[key]; len(got) != 1 || got[0] != first || s.terminalActive[key] != 0 {
		t.Fatalf("after close to the right: %+v active %d", got, s.terminalActive[key])
	}
	s.terminalTabs[key] = []*terminalTab{first, second, third}
	s.terminalActive[key] = 2
	s.closeOtherTerminalTabs(key, 1)
	if got := s.terminalTabs[key]; len(got) != 1 || got[0] != second || s.terminalActive[key] != 0 {
		t.Fatalf("after close others: %+v active %d", got, s.terminalActive[key])
	}
}

// Dragging a tab in the strip past its neighbor reorders the open terminals.
func TestTerminalTabStripDrag(t *testing.T) {
	s := &settings{systemLanguage: "en"}
	key := "repo"
	first := &terminalTab{id: 1, preset: terminalPreset{Name: "Alpha"}}
	second := &terminalTab{id: 2, preset: terminalPreset{Name: "Beta"}}
	s.terminalTabs = map[string][]*terminalTab{key: {first, second}}
	s.terminalActive = map[string]int{key: 0}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { s.terminalTabStrip(c, key) })
	}, 400, 80)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	rect, ok := tt.Find("Alpha")
	if !ok {
		t.Fatal("the first tab is not shown")
	}
	x, y := rect.X+rect.W/2, rect.Y+rect.H/2
	tt.Press(x, y)
	tt.Frame()
	tt.Move(x+rect.W+20, y)
	tt.Frame()
	tt.Release(x+rect.W+20, y)
	tt.Frame()
	got := s.terminalTabs[key]
	if len(got) != 2 || got[0] != second || got[1] != first {
		t.Fatalf("dragged tabs: %+v", got)
	}
	if s.terminalActive[key] != 1 {
		t.Fatalf("active tab after drag: %d", s.terminalActive[key])
	}
}

// The folder icon picks a working directory at the one already shown, and the
// read-only field takes the folder chosen.
func TestTerminalDialogDirectoryPicker(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	other := filepath.Join(t.TempDir(), "other")
	s := &settings{systemLanguage: "en"}
	s.selectedProject = &projectNode{Path: repo, Repository: true}
	s.repo.dispatch = func(fn func()) { fn() }
	key := terminalKey(repo)
	s.openTerminalDialog(key, -1)
	var got mygo.OpenDialogOptions
	s.openDialog = func(opts mygo.OpenDialogOptions) ([]string, error) {
		got = opts
		return []string{other}, nil
	}
	s.chooseTerminalDirectory()
	want := filepath.Clean(other)
	deadline := time.Now().Add(2 * time.Second)
	for s.terminalDialog.dir != want && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if s.terminalDialog.dir != want {
		t.Fatalf("picked directory: %q want %q", s.terminalDialog.dir, want)
	}
	if !got.Directory || got.DefaultPath != filepath.Clean(repo) {
		t.Fatalf("picker options: %+v", got)
	}
}

// The working-directory field renders beside its folder picker.
func TestTerminalDialogDirectoryField(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	s := &settings{systemLanguage: "en"}
	s.selectedProject = &projectNode{Path: repo, Repository: true}
	s.openTerminalDialog(terminalKey(repo), -1)
	tt := ui.NewTester(func(c *ui.Context) {
		s.terminalDialogView(c)
	}, 640, 400)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	if !tt.HasText("Working directory") {
		t.Fatal("the working-directory field is not shown")
	}
}
