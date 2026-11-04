package main

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
)

// contentPage is the page beside the sidebar for the selected repository.
type contentPage int

const (
	contentGit contentPage = iota
	contentTerminal
)

// terminalPreset is a repository terminal's launch shortcut: the name on the
// list, the command typed into its shell, and the working directory.
type terminalPreset struct {
	Name    string `yaml:"name,omitempty"`
	Command string `yaml:"command,omitempty"`
	Dir     string `yaml:"dir,omitempty"`
}

// terminalTab is one running terminal of a repository. The session keeps
// running while the Git page shows, so switching pages never interrupts its
// programs, such as a development server or a coding agent.
type terminalTab struct {
	id       int
	preset   terminalPreset
	term     *terminal.Terminal
	pristine bool // an untouched default shell, which a preset replaces
}

// terminalDialog edits a launch shortcut, or makes one when its index is -1.
type terminalDialog struct {
	open    bool
	key     string
	index   int
	name    string
	command string
	dir     string
}

// terminalDir is the directory a repository's terminals open in by default:
// the main worktree, the repository's own path.
func (s *settings) terminalDir() string {
	if s.selectedProject == nil || !s.selectedProject.Repository {
		return ""
	}
	return filepath.Clean(s.selectedProject.Path)
}

// terminalKey normalizes a repository path for the per-repository maps.
// Windows paths are case-insensitive, as samePath treats them.
func terminalKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func sameTerminalPreset(a, b terminalPreset) bool {
	return a.Name == b.Name && a.Command == b.Command && a.Dir == b.Dir
}

// terminalPresets lists a repository's saved launch shortcuts.
func (s *settings) terminalPresets(key string) []*terminalPreset {
	return s.state.TerminalPresets[key]
}

// terminalTabsOf lists a repository's open terminals.
func (s *settings) terminalTabsOf(key string) []*terminalTab {
	return s.terminalTabs[key]
}

// ensureTerminalMaps makes the per-repository terminal maps writable. Settings
// built before any terminal existed, or from tests, leave them nil.
func (s *settings) ensureTerminalMaps() {
	if s.terminalTabs == nil {
		s.terminalTabs = make(map[string][]*terminalTab)
	}
	if s.terminalActive == nil {
		s.terminalActive = make(map[string]int)
	}
	if s.terminalDrag == nil {
		s.terminalDrag = make(map[int]*ui.Point)
	}
	if s.terminalTabBounds == nil {
		s.terminalTabBounds = make(map[int]ui.Rect)
	}
}

// terminalActiveIndex clamps the remembered active tab of a repository.
func (s *settings) terminalActiveIndex(key string) int {
	tabs := s.terminalTabsOf(key)
	if len(tabs) == 0 {
		return -1
	}
	index := s.terminalActive[key]
	if index < 0 || index >= len(tabs) {
		return 0
	}
	return index
}

// activeTerminal returns the shown terminal of a repository, if any.
func (s *settings) activeTerminal(key string) *terminalTab {
	index := s.terminalActiveIndex(key)
	if index < 0 {
		return nil
	}
	return s.terminalTabsOf(key)[index]
}

// newTerminalTab starts one terminal of a preset. The command is typed into a
// login shell, so a finished command leaves an interactive shell in place; a
// preset without a command is just that shell.
func (s *settings) newTerminalTab(preset terminalPreset, dir string) *terminalTab {
	if preset.Dir != "" {
		dir = preset.Dir
	}
	dir = filepath.Clean(dir)
	s.terminalNextTabID++
	tab := &terminalTab{id: s.terminalNextTabID, preset: preset, pristine: preset.Name == "" && preset.Command == ""}
	onMain := s.repositoryDispatch()
	term, err := terminal.New(terminal.Options{
		Dir: dir,
		OnExit: func(int) {
			onMain(func() { s.invalidate() })
		},
		OnKey: func(ui.Modifiers, ui.Key) bool {
			tab.pristine = false
			return false
		},
	})
	if err != nil {
		s.toast = &ui.Toast{Title: s.text("无法打开终端"), Description: err.Error(), Type: components.ToastError}
		s.invalidate()
		return nil
	}
	tab.term = term
	if preset.Command != "" {
		term.Send([]byte(preset.Command + "\r"))
	}
	return tab
}

// openTerminal shows the built-in terminal of dir, starting a default shell
// the first time a repository has none.
func (s *settings) openTerminal(dir string) {
	if dir == "" {
		dir = s.terminalDir()
	}
	if dir == "" {
		return
	}
	dir = filepath.Clean(dir)
	key := terminalKey(dir)
	s.ensureTerminalMaps()
	if len(s.terminalTabs[key]) == 0 {
		tab := s.newTerminalTab(terminalPreset{Dir: dir}, dir)
		if tab == nil {
			return
		}
		s.terminalTabs[key] = []*terminalTab{tab}
		s.terminalActive[key] = 0
	}
	s.content = contentTerminal
	s.invalidate()
}

// openTerminalPreset shows a saved launch shortcut, replacing an untouched
// default shell instead of opening another tab beside it.
func (s *settings) openTerminalPreset(key string, preset terminalPreset) {
	dir := s.terminalDir()
	if dir == "" {
		dir = preset.Dir
	}
	s.ensureTerminalMaps()
	tabs := s.terminalTabs[key]
	for index, tab := range tabs {
		if sameTerminalPreset(tab.preset, preset) {
			s.terminalActive[key] = index
			s.content = contentTerminal
			s.invalidate()
			return
		}
	}
	index := s.terminalActiveIndex(key)
	if index >= 0 && tabs[index].pristine {
		if tab := s.newTerminalTab(preset, dir); tab != nil {
			s.replaceTerminalTab(key, index, tab)
		}
	} else if tab := s.newTerminalTab(preset, dir); tab != nil {
		s.addTerminalTab(key, tab)
	}
	s.content = contentTerminal
	s.invalidate()
}

// addTerminalTab appends an open terminal and makes it the shown one.
func (s *settings) addTerminalTab(key string, tab *terminalTab) {
	s.ensureTerminalMaps()
	tabs := append(s.terminalTabs[key], tab)
	s.terminalTabs[key] = tabs
	s.terminalActive[key] = len(tabs) - 1
}

// replaceTerminalTab swaps one open terminal for another in place, closing the
// first, and keeps it the shown one.
func (s *settings) replaceTerminalTab(key string, index int, tab *terminalTab) {
	s.ensureTerminalMaps()
	tabs := s.terminalTabs[key]
	if index < 0 || index >= len(tabs) {
		return
	}
	if old := tabs[index].term; old != nil {
		_ = old.Close()
	}
	tabs[index] = tab
	s.terminalActive[key] = index
}

// closeTerminalTab closes one terminal of a repository and forgets it. The
// saved launch shortcut stays in the list.
func (s *settings) closeTerminalTab(key string, index int) {
	s.ensureTerminalMaps()
	tabs := s.terminalTabs[key]
	if index < 0 || index >= len(tabs) {
		return
	}
	closed := tabs[index]
	if closed.term != nil {
		_ = closed.term.Close()
	}
	delete(s.terminalDrag, closed.id)
	delete(s.terminalTabBounds, closed.id)
	active := s.terminalActive[key]
	tabs = append(tabs[:index], tabs[index+1:]...)
	if len(tabs) == 0 {
		delete(s.terminalTabs, key)
		delete(s.terminalActive, key)
	} else {
		if active > index {
			active--
		}
		if active >= len(tabs) {
			active = len(tabs) - 1
		}
		if active < 0 {
			active = 0
		}
		s.terminalTabs[key] = tabs
		s.terminalActive[key] = active
	}
	s.invalidate()
}

// closeOtherTerminalTabs closes every tab of a repository but keep.
func (s *settings) closeOtherTerminalTabs(key string, keep int) {
	s.ensureTerminalMaps()
	tabs := s.terminalTabs[key]
	if keep < 0 || keep >= len(tabs) {
		return
	}
	kept := tabs[keep]
	for index, tab := range tabs {
		if index == keep {
			continue
		}
		if tab.term != nil {
			_ = tab.term.Close()
		}
		delete(s.terminalDrag, tab.id)
		delete(s.terminalTabBounds, tab.id)
	}
	s.terminalTabs[key] = []*terminalTab{kept}
	s.terminalActive[key] = 0
	s.invalidate()
}

// closeTerminalTabsAfter closes the tabs after index.
func (s *settings) closeTerminalTabsAfter(key string, index int) {
	s.ensureTerminalMaps()
	tabs := s.terminalTabs[key]
	if index < 0 || index >= len(tabs) {
		return
	}
	for _, tab := range tabs[index+1:] {
		if tab.term != nil {
			_ = tab.term.Close()
		}
		delete(s.terminalDrag, tab.id)
		delete(s.terminalTabBounds, tab.id)
	}
	tabs = tabs[:index+1]
	s.terminalTabs[key] = tabs
	if s.terminalActive[key] > index {
		s.terminalActive[key] = index
	}
	s.invalidate()
}

// reorderTerminalTab moves the tab dragged from index by offset to the slot
// its pointer ended over, between the other tabs' last laid-out centers.
func (s *settings) reorderTerminalTab(key string, from int, offset ui.Point) {
	s.ensureTerminalMaps()
	tabs := s.terminalTabs[key]
	if from < 0 || from >= len(tabs) {
		return
	}
	moved := tabs[from]
	fromBounds, ok := s.terminalTabBounds[moved.id]
	if !ok {
		return
	}
	center := fromBounds.X + offset.X + fromBounds.W/2
	to := 0
	for index, tab := range tabs {
		if index == from {
			continue
		}
		if bounds, ok := s.terminalTabBounds[tab.id]; ok && bounds.X+bounds.W/2 < center {
			to++
		}
	}
	if to < 0 {
		to = 0
	}
	if to >= len(tabs) {
		to = len(tabs) - 1
	}
	if to == from {
		if pt := s.terminalDrag[moved.id]; pt != nil {
			*pt = ui.Point{}
		}
		return
	}
	next := make([]*terminalTab, 0, len(tabs))
	next = append(next, tabs[:from]...)
	next = append(next, tabs[from+1:]...)
	next = append(next, nil)
	copy(next[to+1:], next[to:])
	next[to] = moved
	s.terminalTabs[key] = next
	s.terminalActive[key] = to
	if pt := s.terminalDrag[moved.id]; pt != nil {
		*pt = ui.Point{}
	}
	s.invalidate()
}

// closeTerminals closes every terminal on the way out.
func (s *settings) closeTerminals() {
	for _, tabs := range s.terminalTabs {
		for _, tab := range tabs {
			if tab.term != nil {
				_ = tab.term.Close()
			}
		}
	}
	s.terminalTabs = nil
	s.terminalActive = nil
	s.terminalDrag = nil
	s.terminalTabBounds = nil
}

// enterTerminal opens a repository's built-in terminal, selecting the
// repository first when its sidebar row is not the shown one.
func (s *settings) enterTerminal(node *projectNode) {
	if node == nil || !node.Repository {
		return
	}
	onMain := s.repositoryDispatch()
	if node != s.selectedProject {
		s.selectProject(node, onMain)
	}
	s.openTerminal(node.Path)
}

// chooseTerminalDirectory asks for the launch shortcut's working directory,
// starting at the one the field shows, and fills the read-only field with the
// folder chosen.
func (s *settings) chooseTerminalDirectory() {
	open := s.openDialog
	if open == nil {
		open = mygo.Dialog.Open
	}
	d := &s.terminalDialog
	key := d.key
	opts := mygo.OpenDialogOptions{
		Parent: s.window, Title: s.text("选择工作目录"), Message: s.text("选择终端启动时所在的文件夹。"),
		ButtonLabel: s.text("选择"), Directory: true,
	}
	opts.DefaultPath = d.dir
	if opts.DefaultPath == "" {
		opts.DefaultPath = s.terminalDir()
	}
	onMain := s.repositoryDispatch()
	// Dialog.Open waits for dismissal. Never wait inside a UI frame.
	go func() {
		paths, err := open(opts)
		if err != nil || len(paths) == 0 {
			return
		}
		onMain(func() {
			if !s.terminalDialog.open || s.terminalDialog.key != key {
				return
			}
			s.terminalDialog.dir = filepath.Clean(paths[0])
			s.invalidate()
		})
	}()
}

// openTerminalDialog edits the shortcut at index, or makes a new one when it
// is -1.
func (s *settings) openTerminalDialog(key string, index int) {
	d := terminalDialog{open: true, key: key, index: index}
	if presets := s.terminalPresets(key); index >= 0 && index < len(presets) {
		d.name, d.command, d.dir = presets[index].Name, presets[index].Command, presets[index].Dir
		if d.dir == "" {
			d.dir = s.terminalDir()
		}
	} else {
		d.dir = s.terminalDir()
	}
	s.terminalDialog = d
	s.invalidate()
}

// submitTerminalDialog saves the shortcut the dialog filled in. It reports
// whether the shortcut is new, so the caller can show it run.
func (s *settings) submitTerminalDialog() (*terminalPreset, bool) {
	d := &s.terminalDialog
	name := strings.TrimSpace(d.name)
	if name == "" {
		return nil, false
	}
	preset := &terminalPreset{
		Name:    name,
		Command: strings.TrimSpace(d.command),
		Dir:     strings.TrimSpace(d.dir),
	}
	list := s.state.TerminalPresets[d.key]
	created := d.index < 0 || d.index >= len(list)
	if created {
		list = append(list, preset)
		d.index = len(list) - 1
	} else {
		*list[d.index] = *preset
	}
	if s.state.TerminalPresets == nil {
		s.state.TerminalPresets = make(map[string][]*terminalPreset)
	}
	s.state.TerminalPresets[d.key] = list
	d.open = false
	s.saveSidebarState()
	s.invalidate()
	return preset, created
}

// deleteTerminalPreset removes a shortcut from a repository's list.
func (s *settings) deleteTerminalPreset(key string, index int) {
	list := s.state.TerminalPresets[key]
	if index < 0 || index >= len(list) {
		return
	}
	list = append(list[:index], list[index+1:]...)
	if len(list) == 0 {
		delete(s.state.TerminalPresets, key)
	} else {
		s.state.TerminalPresets[key] = list
	}
	s.saveSidebarState()
	s.invalidate()
}

// terminalName labels a tab: a shortcut's name, or the default shell.
func (s *settings) terminalName(preset terminalPreset) string {
	if preset.Name == "" {
		return s.text("终端")
	}
	return preset.Name
}

// pageTabSwitch switches the content between the Git page and the terminal,
// whose sessions stay alive.
func (s *settings) pageTabSwitch(c *ui.Context) {
	selected := 0
	if s.content == contentTerminal {
		selected = 1
	}
	tabs := components.Tabs(c, &selected, components.TabsSegmented, s.text("Git"), s.text("终端"))
	if tabs.List.Changed() {
		if selected == 1 {
			// A repository's first look at the terminal page starts a shell, so
			// the page never opens empty.
			dir := s.terminalDir()
			if dir != "" {
				key := terminalKey(dir)
				if len(s.terminalTabsOf(key)) == 0 {
					s.openTerminal(dir)
					return
				}
			}
			s.content = contentTerminal
		} else {
			s.content = contentGit
		}
		c.Invalidate()
	}
}

// terminalPage shows the terminals of the selected repository beside its list
// of saved launch shortcuts.
func (s *settings) terminalPage(c *ui.Context) {
	dir := s.terminalDir()
	if dir == "" {
		return
	}
	key := terminalKey(dir)
	t := c.Theme()
	ui.Row(c).Grow(1).FillWidth().MinHeight(0).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).FillHeight().Children(func() {
			s.terminalTabStrip(c, key)
			if tab := s.activeTerminal(key); tab != nil {
				terminal.View(c, tab.term).Fill().AutoFocus()
				return
			}
			ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				components.Text(c, s.text("选择或新建终端")).TextColor(t.TextMuted)
			})
		})
		ui.Box(c).Width(1).Shrink(0).FillHeight().Background(t.Border)
		s.terminalList(c, key)
	})
}

// terminalTabStrip switches between a repository's open terminals, reorders
// them by dragging, and closes them from each tab's menu or the active tab's
// close button.
func (s *settings) terminalTabStrip(c *ui.Context, key string) {
	s.ensureTerminalMaps()
	tabs := s.terminalTabsOf(key)
	if len(tabs) == 0 {
		return
	}
	selected := s.terminalActiveIndex(key)
	t := c.Theme()
	buttons := make([]ui.Element, len(tabs))
	offsets := make([]*ui.Point, len(tabs))
	closed := false
	ui.Row(c).FillWidth().Shrink(0).AlignItems(ui.Center).Gap(t.Space(1)).Padding(t.Space(1), 0, t.Space(1), 0).Children(func() {
		ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(t.Space(0.5)).Padding(t.Space(0.5)).Radius(t.Space(20)).Background(t.Surface).Children(func() {
			for i, tab := range tabs {
				i, tab := i, tab
				offset := s.terminalDrag[tab.id]
				if offset == nil {
					offset = &ui.Point{}
					s.terminalDrag[tab.id] = offset
				}
				offsets[i] = offset
				variant := components.ButtonGhost
				if i == selected {
					variant = components.ButtonToolbar
				}
				button := components.Button(c, s.terminalName(tab.preset), variant)
				button.Key(tab.id)
				button.Draggable(offset, ui.DraggableOptions{Axis: ui.Horizontal})
				if button.Clicked() && i != selected {
					s.terminalActive[key] = i
					c.Invalidate()
				}
				button.ContextMenu(func(m *ui.Menu) {
					if m.Item(s.text("关闭")).Chosen() {
						s.closeTerminalTab(key, i)
					}
					if m.Item(s.text("关闭其他")).Chosen() {
						s.closeOtherTerminalTabs(key, i)
					}
					if m.Item(s.text("关闭右侧")).Chosen() {
						s.closeTerminalTabsAfter(key, i)
					}
				})
				buttons[i] = button
			}
		})
		ui.Box(c).Grow(1)
		if components.IconButton(c, lucide.X(), s.text("关闭终端"), components.IconButtonCompact).Clicked() {
			s.closeTerminalTab(key, selected)
			closed = true
		}
	})
	if closed {
		return
	}
	for i := range buttons {
		if _, _, ended := buttons[i].DragEnded(); ended {
			s.reorderTerminalTab(key, i, *offsets[i])
			return
		}
		if bounds := buttons[i].LayoutBounds(); bounds.W > 0 || bounds.H > 0 {
			s.terminalTabBounds[tabs[i].id] = bounds
		}
	}
}

// terminalList is the right side of the terminal page: a repository's saved
// launch shortcuts, with a button making another one.
func (s *settings) terminalList(c *ui.Context, key string) {
	t := c.Theme()
	ui.Column(c).Width(t.Space(60)).FillHeight().Shrink(0).Gap(t.Space(1)).Padding(t.Space(1.5)).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(6).Children(func() {
			components.Text(c, s.text("终端列表")).Bold()
			ui.Box(c).Grow(1)
			if components.Button(c, s.text("新建终端"), components.ButtonGhost).Clicked() {
				s.openTerminalDialog(key, -1)
			}
		})
		presets := s.terminalPresets(key)
		if len(presets) == 0 {
			components.Text(c, s.text("还没有终端")).Wrap().TextColor(t.TextMuted)
		}
		ui.Column(c).FillWidth().Grow(1).Gap(t.Space(0.5)).Children(func() {
			for index, preset := range presets {
				index, preset := index, preset
				row := components.Button(c, s.terminalName(*preset), components.ButtonGhost).FillWidth()
				if row.Clicked() {
					s.openTerminalPreset(key, *preset)
				}
				row.ContextMenu(func(m *ui.Menu) {
					if m.Item(s.text("编辑")).Chosen() {
						s.openTerminalDialog(key, index)
					}
					if m.Item(s.text("删除")).Chosen() {
						s.deleteTerminalPreset(key, index)
					}
				})
				if preset.Command != "" {
					components.Text(c, preset.Command).SingleLine().FontSize(t.Rem(0.85)).TextColor(t.TextMuted)
				}
			}
		})
	})
}

// terminalDialogView is the form making or editing a launch shortcut.
func (s *settings) terminalDialogView(c *ui.Context) {
	d := &s.terminalDialog
	if !d.open {
		return
	}
	title, action := s.text("新建终端"), s.text("创建")
	if d.index >= 0 {
		title, action = s.text("编辑终端"), s.text("保存")
	}
	ui.Box(c).Key("terminal-dialog").Absolute().Children(func() {
		components.Dialog(c, &d.open, components.DialogOptions{Label: title}, func(panel ui.Element) {
			t := c.Theme()
			panel.Width(t.Space(100))
			components.Text(c, title).Bold().FontSize(t.Rem(1.1))
			field := func(label string, value *string, placeholder string, focus bool) {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(2)).Children(func() {
					components.Text(c, label).Width(t.Space(16)).Shrink(0).TextColor(t.TextMuted)
					box := components.TextBox(c, value, components.TextBoxOptions{}).
						Grow(1).Label(label).Placeholder(placeholder)
					if focus {
						box.AutoFocus()
					}
				})
			}
			dirField := func(label string, value *string) {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(t.Space(2)).Children(func() {
					components.Text(c, label).Width(t.Space(16)).Shrink(0).TextColor(t.TextMuted)
					ui.Box(c).Grow(1).MinWidth(0).Children(func() {
						components.TextBox(c, value, components.TextBoxOptions{ReadOnly: true, LeadingIcon: lucide.Folder()}).
							FillWidth().Label(label)
						// The folder icon inside the field opens the picker; the
						// field stays read-only so only the picker sets its path.
						pick := ui.ButtonBase(c).Absolute().Left(t.Space(1.5)).Top(t.Space(1.5)).
							Width(t.Space(7)).Height(t.Space(6)).Cursor(ui.CursorPointer).
							Label(s.text("选择工作目录")).Tooltip(s.text("选择工作目录"))
						if pick.Clicked() {
							s.chooseTerminalDirectory()
						}
					})
				})
			}
			ui.Column(c).FillWidth().Gap(t.Space(1.5)).Children(func() {
				field(s.text("名称"), &d.name, s.text("名称"), true)
				field(s.text("启动命令"), &d.command, "npm run dev", false)
				dirField(s.text("工作目录"), &d.dir)
			})
			ready := strings.TrimSpace(d.name) != ""
			ui.Row(c).FillWidth().Wrap().Justify(ui.End).Gap(t.Space(2)).Children(func() {
				if components.Button(c, s.text("取消"), components.ButtonSecondary).Clicked() {
					d.open = false
				}
				if components.Button(c, action, components.ButtonPrimary).Disabled(!ready).Clicked() {
					if preset, created := s.submitTerminalDialog(); created && preset != nil {
						s.openTerminalPreset(s.terminalDialog.key, *preset)
					}
				}
			})
		})
	})
}
