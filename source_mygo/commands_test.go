package main

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components/command"
)

func TestCommandSearchAndNavigation(t *testing.T) {
	repo := &projectNode{Path: "/projects/team/Onegit", Repository: true}
	missing := &projectNode{Path: "/projects/old/Onegit", Repository: true, Missing: true}
	s := &settings{systemLanguage: "zh-CN", path: filepath.Join(t.TempDir(), "config.yaml"), projectTree: &projectNode{Path: "/projects", Children: []*projectNode{repo, missing}}}
	if matches := command.Filter(s.commandItems(), "TEAM onegit"); len(matches) != 1 || matches[0].ID != "project/"+repo.Path {
		t.Fatalf("path search: %+v", matches)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().Children(func() {
			ui.Column(c).Width(240).FillHeight().Children(func() { s.worktreeSidebar(c) })
			ui.Column(c).Fill().Children(func() { s.welcome(c); s.render(c); s.renderCommands(c) })
		})
	}, 1100, 720)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if err := tt.Click("搜索项目与设置"); err != nil {
		t.Fatal(err)
	}
	if !s.commands.open || !tt.Focused("命令面板") {
		t.Fatal("trigger did not focus search")
	}
	tt.Type("team Onegit")
	tt.Key(0, ui.KeyEnter)
	if s.commands.open || s.selectedProject != repo {
		t.Fatal("project command did not select original node")
	}
	tt.Key(ui.Cmd, ui.KeyF)
	if !s.commands.open || s.commands.palette.Query != "" {
		t.Fatal("shortcut did not reset palette")
	}
	tt.Type("nothing-matches")
	if !tt.HasText("没有匹配的项目或设置") {
		t.Fatal("missing empty state")
	}
	tt.Key(0, ui.KeyEnter)
	if !s.commands.open {
		t.Fatal("empty search activated")
	}
	tt.Key(0, ui.KeyEscape)
	if s.commands.open {
		t.Fatal("escape did not dismiss")
	}
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Type("theme")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !s.open || s.commands.open || !tt.HasText("主题偏好") {
		t.Fatal("settings handoff failed", tt.Texts())
	}
	tt.Key(ui.Cmd, ui.KeyF)
	if s.commands.open {
		t.Fatal("shortcut escaped settings modal")
	}
	if err := tt.Click("主题偏好"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("深色"); err != nil {
		t.Fatal(err)
	}
	saved, err := loadPreferences(s.path)
	if err != nil || saved.Theme != "dark" {
		t.Fatalf("command settings did not persist: %+v %v", saved, err)
	}
}

func TestCommandDismissalAndSettingsDestinations(t *testing.T) {
	s := &settings{systemLanguage: "en-US"}
	tt := ui.NewTester(func(c *ui.Context) { s.commandTrigger(c); s.render(c); s.renderCommands(c) }, 820, 600)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if err := tt.Click("Search projects and settings"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if !tt.Focused("Search projects and settings") {
		t.Fatal("trigger focus not restored")
	}
	// A settings destination still opens the matching row.
	tt.Key(ui.Cmd, ui.KeyF)
	tt.Type("language")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if !s.open || !tt.HasText("Language preference") {
		t.Fatalf("language destination missing: %v", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)

	// Folders are added from the palette instead of a settings row.
	found := false
	for _, item := range s.commandItems() {
		found = found || item.ID == "projects/add"
	}
	if !found {
		t.Fatal("add folder command is missing")
	}
}
