package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onegit/source_mygo/launchers"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
	"github.com/oneo-me/onevra/source/shells/sidebar"
)

type preferences struct {
	// Roots are the folders the sidebar lists, in the order they were added.
	Roots []string `json:"roots,omitempty" yaml:"roots,omitempty"`
	// ProjectParentDir is the single folder of earlier versions; loading
	// turns it into Roots and drops it.
	ProjectParentDir string           `json:"projectParentDir,omitempty" yaml:"projectParentDir,omitempty"`
	Theme            mygo.ThemeSource `json:"theme,omitempty" yaml:"theme,omitempty"`
	Language         string           `json:"language,omitempty" yaml:"language,omitempty"`
	// Terminal and Editor choose the title bar's launchers by ID, empty for
	// the first installed one or launchers.None to hide the button.
	Terminal string `json:"terminal,omitempty" yaml:"terminal,omitempty"`
	Editor   string `json:"editor,omitempty" yaml:"editor,omitempty"`
}

func loadPreferences(path string) (preferences, error) {
	var p preferences
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	err = yaml.Unmarshal(data, &p)
	if err != nil {
		return preferences{}, err
	}
	if len(p.Roots) == 0 && p.ProjectParentDir != "" {
		p.Roots = []string{p.ProjectParentDir}
	}
	p.ProjectParentDir = ""
	if p.Theme != "" && p.Theme != mygo.ThemeSystem && p.Theme != mygo.ThemeLight && p.Theme != mygo.ThemeDark {
		return preferences{}, fmt.Errorf("unknown theme %q", p.Theme)
	}
	if p.Language != "" && p.Language != "en" && p.Language != "zh-Hans" {
		return preferences{}, fmt.Errorf("unknown language %q", p.Language)
	}
	p.Terminal, p.Editor = launchers.Normalize(p.Terminal, launchers.Terminal), launchers.Normalize(p.Editor, launchers.Editor)
	return p, nil
}

func savePreferences(path string, p preferences) error {
	return saveYAML(path, p)
}

type settings struct {
	commands          commandState
	overview          projectOverview
	content           contentPage
	terminalTabs      map[string][]*terminalTab
	terminalActive    map[string]int
	terminalDialog    terminalDialog
	// terminalDrag and terminalTabBounds keep a dragged tab's offset and the
	// tabs' last laid-out rectangles while the pointer reorders them.
	terminalDrag      map[int]*ui.Point
	terminalTabBounds map[int]ui.Rect
	terminalNextTabID int
	repo              repositoryState
	state             sidebarState
	statePath         string
	stateProblem      string
	projectTree       *projectNode
	selectedProject   *projectNode
	scanning          bool
	scanProblem       string
	scanVersion       uint64
	scanCancel        context.CancelFunc
	open              bool
	choosingDirectory bool
	view              components.SettingsState
	values            preferences
	path              string
	problem           string
	openPath          func(string) error
	openDialog        func(mygo.OpenDialogOptions) ([]string, error)
	choosingIcon      bool
	projectIcons      map[string]ui.ImageSource
	toast             *ui.Toast
	groupDialog       groupDialog
	window            *mygo.Window
	shell             *sidebar.Shell
	systemLanguage    string
	// installed lists this computer's launchers once found, and runLauncher,
	// when set, starts their commands instead of launchers.Run.
	installed   []launchers.Installed
	runLauncher func(*exec.Cmd) error
}

func newSettings(dir string) *settings {
	s := &settings{
		systemLanguage: mygo.App.Locale(),
		terminalTabs:      make(map[string][]*terminalTab),
		terminalActive:    make(map[string]int),
		terminalDrag:      make(map[int]*ui.Point),
		terminalTabBounds: make(map[int]ui.Rect),
	}
	var err error
	if dir != "" {
		s.path = filepath.Join(dir, "config.yaml")
		s.statePath = filepath.Join(dir, "state.yaml")
		legacyDir, legacyErr := os.UserConfigDir()
		if legacyErr == nil {
			s.values, err = loadOrMigratePreferences(s.path, filepath.Join(legacyDir, "Onegit", "settings.json"))
		} else {
			s.values, err = loadPreferences(s.path)
		}
		var stateErr error
		s.state, stateErr = loadSidebarState(s.statePath)
		if stateErr != nil {
			s.stateProblem = s.text("读取状态失败：") + stateErr.Error()
		}
		s.adoptLegacyRoot()
	}
	if err != nil {
		s.problem = s.text("读取设置失败：") + err.Error()
	}
	mygo.Theme.SetSource(s.theme())
	return s
}

func (s *settings) show() {
	s.commands.open = false
	s.open = true
	if s.window != nil {
		s.window.Show()
		s.window.Focus()
		s.window.Invalidate()
	}
}

func (s *settings) toolbar(c *ui.Context) {
	if sidebar.IconButton(c, lucide.Plus(), s.text("添加文件夹")).Label(s.text("添加文件夹")).Clicked() {
		s.addFolder()
	}
	if sidebar.IconButton(c, lucide.Settings(), s.text("设置")+" (⌘,)").Label(s.text("设置")).Clicked() {
		s.show()
	}
	ui.Spacer(c)
	icon := lucide.Monitor()
	switch s.theme() {
	case mygo.ThemeLight:
		icon = lucide.Sun()
	case mygo.ThemeDark:
		icon = lucide.Moon()
	}
	sidebar.IconButton(c, icon, s.text("主题")).Menu(func(m *ui.Menu) {
		for _, choice := range []mygo.ThemeSource{mygo.ThemeSystem, mygo.ThemeLight, mygo.ThemeDark} {
			if m.Item(s.themeLabel(choice)).Checked(s.theme() == choice).Chosen() {
				s.setTheme(choice)
			}
		}
	})
}

// addFolder asks for a folder to add to the sidebar.
func (s *settings) addFolder() {
	s.startDirectorySelection(mygo.Dialog.Open, mygo.RunOnMain)
}

func (s *settings) startDirectorySelection(open func(mygo.OpenDialogOptions) ([]string, error), onMain func(func())) {
	if s.choosingDirectory {
		return
	}
	s.choosingDirectory = true
	opts := mygo.OpenDialogOptions{
		Parent: s.window, Title: s.text("添加文件夹"), Message: s.text("选择要加入侧栏的文件夹。"),
		ButtonLabel: s.text("添加"), Directory: true,
	}
	if roots := s.roots(); len(roots) > 0 {
		opts.DefaultPath = filepath.Dir(roots[len(roots)-1])
	}
	// Dialog.Open waits for dismissal. Never wait inside a UI frame: native
	// events must be able to finish that frame while the sheet is open.
	go func() {
		paths, err := open(opts)
		onMain(func() {
			s.choosingDirectory = false
			s.applyFolderSelection(paths, err)
			if s.window != nil {
				s.window.Invalidate()
			}
		})
	}()
}

func (s *settings) applyFolderSelection(paths []string, err error) {
	if err != nil {
		s.problem = s.text("选择目录失败：") + err.Error()
		return
	}
	if len(paths) == 0 {
		return
	}
	info, err := os.Stat(paths[0])
	if err != nil {
		s.problem = s.text("无法访问所选目录：") + err.Error()
		return
	}
	if !info.IsDir() {
		s.problem = s.text("请选择文件夹。")
		return
	}
	s.addRoot(filepath.Clean(paths[0]))
}

// roots lists the added folders in sidebar order.
func (s *settings) roots() []string {
	return s.values.Roots
}

// addRoot adds a folder to the sidebar and scans it. A folder already listed,
// or one that contains or sits inside a listed folder, is refused so that no
// repository appears twice.
func (s *settings) addRoot(path string) bool {
	path = folderPath(path)
	for _, root := range s.roots() {
		resolved := folderPath(root)
		switch {
		case samePath(resolved, path):
			s.toast = &ui.Toast{
				ID: "folder-listed", Title: s.text("该文件夹已在侧栏中"), Description: root,
				Type: components.ToastError,
			}
			s.invalidate()
			return false
		case withinFolder(resolved, path):
			s.toast = &ui.Toast{
				ID: "folder-listed", Title: fmt.Sprintf(s.text("该文件夹已包含在「%s」中"), projectName(root)),
				Type: components.ToastError,
			}
			s.invalidate()
			return false
		case withinFolder(path, resolved):
			s.toast = &ui.Toast{
				ID: "folder-listed", Title: fmt.Sprintf(s.text("「%s」已包含在此文件夹中"), projectName(root)),
				Type: components.ToastError,
			}
			s.invalidate()
			return false
		}
	}
	next := s.values
	next.Roots = append(append([]string{}, s.roots()...), path)
	if !s.save(next) {
		return false
	}
	// Without a window no one shows the result; tests drive the scan.
	if s.window != nil {
		s.scanProjects()
	}
	return true
}

// removeFolder drops a folder from the sidebar without touching its files. Its
// cached tree stays, so adding the folder again restores pins and groups.
func (s *settings) removeFolder(path string) {
	path = folderPath(path)
	roots := s.roots()
	kept := make([]string, 0, len(roots))
	for _, root := range roots {
		if !samePath(folderPath(root), path) {
			kept = append(kept, root)
		}
	}
	if len(kept) == len(roots) {
		return
	}
	next := s.values
	next.Roots = kept
	if !s.save(next) {
		return
	}
	if s.window != nil {
		s.scanProjects()
	}
	s.invalidate()
}

// folderPath normalizes a folder the way scanning does: absolute and clean,
// keeping the spelling the user picked.
func folderPath(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

// withinFolder reports whether path is folder or lies inside it.
func withinFolder(folder, path string) bool {
	if samePath(folder, path) {
		return true
	}
	rel, err := filepath.Rel(filepath.Clean(folder), filepath.Clean(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// samePath compares two folders the way their filesystem does.
func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	return runtime.GOOS == "windows" && strings.EqualFold(a, b)
}

func (s *settings) save(next preferences) bool {
	if s.path == "" {
		s.problem = s.text("无法确定设置保存位置。")
		return false
	}
	if err := savePreferences(s.path, next); err != nil {
		s.problem = fmt.Sprintf(s.text("保存失败，设置未更改：%v"), err)
		return false
	}
	s.values, s.problem = next, ""
	return true
}

func (s *settings) render(c *ui.Context) {
	if !s.open {
		return
	}
	t := c.Theme()
	w, h := c.Size()
	inset := sidebar.HeaderHeight(c)
	width := min(t.Space(200), w*.88, max(0, w-2*inset))
	components.Dialog(c, &s.open, components.DialogOptions{Label: s.text("设置"), Inset: inset}, func(panel ui.Element) {
		panel.Width(width).Height(min(t.Space(180), h*.85)).Padding(0).Gap(0).Clip()
		if s.problem != "" {
			ui.Column(c).FillWidth().Shrink(0).Padding(16).Children(func() { components.Text(c, s.problem).Wrap() })
		}
		components.Settings(c, &s.view, s.categories(), components.SettingsOptions{
			Narrow: width < t.Space(170), Title: s.text("设置"), SearchLabel: s.text("搜索设置"), SearchPlaceholder: s.text("搜索设置…"), CategoryLabel: s.text("设置分类"), ResultsTitle: s.text("搜索结果"), EmptyText: s.text("没有匹配的设置"),
			HeaderActions: func(c *ui.Context) {
				if sidebar.IconButton(c, lucide.X(), s.text("关闭设置")).Clicked() {
					s.open = false
				}
			},
		})
	})
}

func (s *settings) menu() *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu, Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleAbout}, mygo.Separator(),
			{Label: s.text("设置…"), Accelerator: "Cmd+,", Click: func(*mygo.MenuItem, *mygo.Window) { s.show() }},
			mygo.Separator(), {Role: mygo.RoleServices}, mygo.Separator(),
			{Role: mygo.RoleHide}, {Role: mygo.RoleHideOthers}, {Role: mygo.RoleUnhide}, mygo.Separator(), {Role: mygo.RoleQuit},
		}},
		{Role: mygo.RoleFileMenu}, {Role: mygo.RoleEditMenu}, {Role: mygo.RoleViewMenu}, {Role: mygo.RoleWindowMenu},
	})
}

// Import the old preferences only when the new config does not exist.
func loadOrMigratePreferences(path, legacy string) (preferences, error) {
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		return loadPreferences(path)
	}
	if _, err := os.Stat(legacy); errors.Is(err, os.ErrNotExist) {
		return preferences{}, nil
	}
	p, err := loadPreferences(legacy)
	if err != nil {
		return p, err
	}
	return p, savePreferences(path, p)
}

// Move the earlier ~/.oneo/onegit directory once, when the data directory does not exist.
func migrateDataDir(dir, legacy string) error {
	if _, err := os.Lstat(dir); !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(legacy); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return os.Rename(legacy, dir)
}
