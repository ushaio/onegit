package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"

	"github.com/egoist/mygo/ui"
)

func TestFolderPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Onegit", "settings.json")
	if got, err := loadPreferences(path); err != nil || len(got.Roots) != 0 {
		t.Fatalf("first launch: %+v, %v", got, err)
	}
	s := &settings{path: path, systemLanguage: "en-US"}
	first := filepath.Join(t.TempDir(), "项目 with spaces")
	second := t.TempDir()
	for _, dir := range []string{first, second} {
		if !s.addRoot(dir) {
			t.Fatalf("could not add %s: %s", dir, s.problem)
		}
	}
	got, err := loadPreferences(path)
	if err != nil || len(got.Roots) != 2 || got.Roots[0] != first || got.Roots[1] != second || s.problem != "" {
		t.Fatalf("restore: %+v, %v, %s", got, err, s.problem)
	}
	// A folder already listed, or nested in a listed folder, would show its
	// repositories twice.
	if s.addRoot(first) || s.addRoot(filepath.Join(first, "nested")) {
		t.Fatal("duplicate folder accepted")
	}
	if len(s.values.Roots) != 2 || s.toast == nil {
		t.Fatalf("duplicate folder changed the list: %+v", s.values.Roots)
	}
	s.removeFolder(first)
	got, err = loadPreferences(path)
	if err != nil || len(got.Roots) != 1 || got.Roots[0] != second {
		t.Fatalf("remove: %+v, %v", got, err)
	}
}

func TestSaveFailurePreservesSelection(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "file")
	if err := os.WriteFile(blocked, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &settings{path: filepath.Join(blocked, "settings.json"), values: preferences{Roots: []string{root}}}
	if s.addRoot(filepath.Join(filepath.Dir(root), "other")) || len(s.values.Roots) != 1 || s.values.Roots[0] != root || s.problem == "" {
		t.Fatal("failed save changed folders or hid failure")
	}
	data, err := os.ReadFile(blocked)
	if err != nil || string(data) != "keep" {
		t.Fatal("failed save damaged existing file")
	}
}

// Launcher choices keep known IDs of their kind and reset the rest.
func TestLauncherPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("terminal: zed\neditor: cursor\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := loadPreferences(path); err != nil || got.Terminal != "" || got.Editor != "cursor" {
		t.Fatalf("launchers: %+v, %v", got, err)
	}
}

func TestMalformedPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPreferences(path); err == nil {
		t.Fatal("malformed preferences accepted")
	}
}

func TestSettingsInteraction(t *testing.T) {
	for _, width := range []int{820, 1100} {
		t.Run(strconv.Itoa(width), func(t *testing.T) {
			s := &settings{path: filepath.Join(t.TempDir(), "settings.json"), systemLanguage: "zh-CN"}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() { s.toolbar(c); s.welcome(c); s.render(c) })
			}, width, 720)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			if err := tt.Click("设置"); err != nil {
				t.Fatal(err)
			}
			if !s.open || !tt.HasText("通用") {
				t.Fatal("settings missing", tt.Texts())
			}
			if tt.HasText("项目父目录") || tt.HasText("选择目录…") {
				t.Fatal("the directory setting is still listed", tt.Texts())
			}
			layout := ui.NewTester(s.render, width, 720)
			layout.SetPreferences(ui.Preferences{ReduceMotion: true})
			layout.Frame()
			panel, ok := layout.Find("设置")
			if !ok {
				t.Fatal("missing dialog")
			}
			item, ok := layout.Find("语言偏好")
			if !ok || item.X < panel.X || item.X+item.W > panel.X+panel.W || item.Y+item.H > panel.Y+panel.H {
				t.Fatalf("language setting outside dialog: %+v / %+v", item, panel)
			}
			if err := tt.Click("关闭设置"); err != nil {
				t.Fatal(err)
			}
			if s.open {
				t.Fatal("close did not dismiss settings")
			}
			if err := tt.Click("设置"); err != nil {
				t.Fatal(err)
			}
			tt.Key(0, ui.KeyEscape)
			tt.Frame()
			if s.open {
				t.Fatal("Escape did not dismiss settings")
			}
		})
	}
}

func TestAppearanceAndLanguage(t *testing.T) {
	s := &settings{path: filepath.Join(t.TempDir(), "settings.json"), systemLanguage: "zh-CN"}
	s.values.Roots = []string{"/Users/example/Projects"}
	s.save(s.values)
	s.open = true
	tt := ui.NewTester(s.render, 1100, 720)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if err := tt.Click("外观"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("主题偏好"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("深色"); err != nil {
		t.Fatal(err)
	}
	if s.theme() != "dark" {
		t.Fatal("theme did not change")
	}
	if err := tt.Click("通用"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("语言偏好"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("English"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Language preference") {
		t.Fatal("language did not apply", tt.Texts())
	}
	restored, err := loadPreferences(s.path)
	if err != nil || restored.Language != "en" || restored.Theme != "dark" || len(restored.Roots) != 1 || restored.Roots[0] != "/Users/example/Projects" {
		t.Fatalf("preferences not retained: %+v, %v", restored, err)
	}
	toolbar := ui.NewTester(func(c *ui.Context) { ui.Row(c).FillWidth().Children(func() { s.toolbar(c) }) }, 240, 400)
	toolbar.SetPreferences(ui.Preferences{ReduceMotion: true})
	if err := toolbar.Click("Theme"); err != nil {
		t.Fatal(err)
	}
	if err := toolbar.ChooseMenuItem("Light"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Appearance"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Light") {
		t.Fatal("settings did not reflect toolbar choice", tt.Texts())
	}
	restored, err = loadPreferences(s.path)
	if err != nil || restored.Theme != "light" {
		t.Fatal("toolbar theme not saved")
	}
	s.addRoot("/new/path")
	if s.values.Language != "en" || s.theme() != "light" {
		t.Fatal("adding a folder reset other preferences")
	}
}

func TestSystemLanguage(t *testing.T) {
	for _, locale := range []string{"zh-CN", "zh_Hans_SG", "en-US", "fr-FR"} {
		s := &settings{systemLanguage: locale}
		want := "Settings"
		if strings.HasPrefix(locale, "zh") {
			want = "设置"
		}
		if s.text("设置") != want {
			t.Fatalf("incorrect system language for %s", locale)
		}
	}
}

func TestFolderSelectionDoesNotBlock(t *testing.T) {
	s := &settings{open: true, path: filepath.Join(t.TempDir(), "settings.json"), systemLanguage: "zh-CN"}
	selected := t.TempDir()
	opened := make(chan mygo.OpenDialogOptions, 2)
	release := make(chan struct{})
	defer close(release)
	callbacks := make(chan func(), 2)
	open := func(opts mygo.OpenDialogOptions) ([]string, error) {
		opened <- opts
		<-release
		return []string{selected}, nil
	}
	onMain := func(fn func()) { callbacks <- fn }
	s.startDirectorySelection(open, onMain)
	select {
	case opts := <-opened:
		if !opts.Directory || opts.Multiple {
			t.Fatal("expected single directory picker")
		}
	case <-time.After(time.Second):
		t.Fatal("picker did not start")
	}
	s.startDirectorySelection(open, onMain)
	tt := ui.NewTester(s.render, 1100, 720)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	tt.Frame()
	if !tt.HasText("通用") {
		t.Fatal("UI did not render while picker was waiting")
	}
	// Other preferences may change before the native picker returns.
	s.setLanguage("en")
	release <- struct{}{}
	select {
	case fn := <-callbacks:
		if !s.choosingDirectory || len(s.values.Roots) != 0 {
			t.Fatal("worker mutated UI state")
		}
		fn()
	case <-time.After(time.Second):
		t.Fatal("picker did not finish")
	}
	select {
	case <-opened:
		t.Fatal("duplicate picker opened")
	default:
	}
	if s.choosingDirectory || len(s.values.Roots) != 1 || s.values.Roots[0] != selected || s.values.Language != "en" {
		t.Fatal("incorrect completion state")
	}
	got, err := loadPreferences(s.path)
	if err != nil || len(got.Roots) != 1 || got.Roots[0] != selected || got.Language != "en" {
		t.Fatalf("selection not persisted: %+v, %v", got, err)
	}
}

func TestFolderSelectionResults(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		paths     []string
		err       error
		wantError bool
	}{
		{name: "cancel"},
		{name: "dialog error", err: errors.New("picker failed"), wantError: true},
		{name: "missing directory", paths: []string{filepath.Join(root, "missing")}, wantError: true},
		{name: "file", paths: []string{file}, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &settings{path: filepath.Join(t.TempDir(), "settings.json"), values: preferences{Roots: []string{root}}}
			s.applyFolderSelection(tc.paths, tc.err)
			if len(s.values.Roots) != 1 || s.values.Roots[0] != root || (s.problem != "") != tc.wantError {
				t.Fatalf("unexpected result: %+v", s)
			}
		})
	}
}
