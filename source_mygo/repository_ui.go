package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onegit/source_mygo/gitview"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/shells/sidebar"
)

// repositoryState holds the selected worktree's Git view. The view owns the
// project sidebar, content and Git dialogs; Onegit's title bar starts its
// actions.
type repositoryState struct {
	view     *gitview.View
	dispatch func(func())
}

func (s *settings) repositoryDispatch() func(func()) {
	if s.repo.dispatch != nil {
		return s.repo.dispatch
	}
	return mygo.RunOnMain
}

func (s *settings) invalidate() {
	if s.window != nil {
		s.window.Invalidate()
	}
}

// language is the effective language tag: the preference, else the system's.
func (s *settings) language() string {
	if s.values.Language != "" {
		return s.values.Language
	}
	return s.systemLanguage
}

// gitView returns the repository view, creating it on first use with the
// remembered layout. Avatars are cached beside the configuration.
func (s *settings) gitView() *gitview.View {
	if s.repo.view != nil {
		return s.repo.view
	}
	var state gitview.State
	if s.state.GitView != nil {
		state = *s.state.GitView
	}
	avatars := ""
	if s.path != "" {
		avatars = filepath.Join(filepath.Dir(s.path), "avatars")
	}
	s.repo.view = gitview.New(gitview.Options{
		Dispatch:   func(fn func()) { s.repositoryDispatch()(fn) },
		Invalidate: s.invalidate,
		Git:        gitview.Git{Env: gitEnv()},
		Language:   s.language(),
		AvatarDir:  avatars,
		State:      state,
		OnState: func(state gitview.State) {
			s.state.GitView = &state
			s.saveSidebarState()
		},
		DialogInset: sidebar.HeaderHeight,
		// Files open in their default application or show in the file manager.
		OpenFile:   mygo.Shell.OpenPath,
		RevealFile: mygo.Shell.ShowItemInFolder,
		// A worktree, or a branch another worktree has checked out, switches
		// the page to that worktree.
		OpenWorktree: s.openWorktree,
		// The refs pane lists the worktrees, including one added from a
		// ref's menu once Git lists it.
		Worktrees: true,
		OnWrite: func(op, _ string) {
			if op == gitview.OpWorktree {
				s.reloadWorktrees()
			}
		},
	})
	return s.repo.view
}

// gitEnv drops Git overrides from a launcher or development shell, which
// would otherwise describe every path as the launcher's repository.
func gitEnv() []string {
	var env []string
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GIT_") {
			env = append(env, entry)
		}
	}
	return env
}

// repositoryChanged shows the selected worktree, rereading it when it was
// already shown.
func (s *settings) repositoryChanged() {
	view := s.gitView()
	if view.Path() == s.overview.SelectedPath {
		view.Reload()
		return
	}
	view.SetPath(s.overview.SelectedPath)
}

// welcome builds the repository page of the selected project.
func (s *settings) welcome(c *ui.Context) {
	if s.selectedProject == nil || !s.selectedProject.Repository {
		return
	}
	s.repositoryPage(c)
}

func (s *settings) repositoryPage(c *ui.Context) {
	ui.Column(c).Fill().Padding(sidebar.HeaderHeight(c), 0, 0, 0).Children(func() {
		// A hairline closes the title bar above the page.
		ui.Box(c).FillWidth().Height(1).Shrink(0).Background(c.Theme().Border)
		if s.content == contentTerminal {
			s.terminalPage(c)
		} else if s.overview.Loading {
			s.repositoryMessage(c, s.text("正在读取 worktree…"))
		} else if s.overview.Problem != "" {
			s.repositoryMessage(c, s.text("读取 worktree 失败：")+s.overview.Problem)
		} else {
			view := s.gitView()
			// The language preference can change while the view lives.
			view.SetLanguage(s.language())
			ui.Column(c).Grow(1).FillWidth().MinHeight(0).Children(func() { view.Layout(c) })
		}
		s.terminalDialogView(c)
	})
}

func (s *settings) repositoryMessage(c *ui.Context, message string) {
	ui.Column(c).Grow(1).FillWidth().AlignItems(ui.Center).Justify(ui.Center).Padding(16).Children(func() {
		components.Text(c, message).Wrap().TextColor(c.Theme().TextMuted)
	})
}

// repositoryTitle fills the window's title bar, as Neorio does, instead of
// adding a toolbar strip: the project's name, the view's toolbar centered
// between the sides, and the launchers.
func (s *settings) repositoryTitle(c *ui.Context) {
	project := s.selectedProject
	if project == nil || !project.Repository {
		components.Text(c, "Onegit").Bold()
		return
	}
	size := max(32, c.Theme().Rem(2))
	ui.Row(c).Grow(1).MinWidth(0).FillHeight().AlignItems(ui.Center).Gap(8).Margin(0, titleTrailing(c, size), 0, 0).Children(func() {
		// Equal sides keep the toolbar centered; the launchers' side never
		// shrinks below them, and the name gives way first.
		ui.Row(c).Key("title-name").Grow(1).Basis(0).MinWidth(0).AlignItems(ui.Center).Children(func() {
			components.Text(c, projectName(project.Path)).Bold().SingleLine().Shrink(1).MinWidth(0)
		})
		s.gitView().Toolbar(c, size)
		ui.Row(c).Key("launchers").Grow(1).Basis(0).Justify(ui.End).AlignItems(ui.Center).Gap(6).Children(func() {
			s.pageTabSwitch(c)
			s.launcherButtons(c)
		})
	})
}

// titleTrailing balances the space after the title bar actions with the space
// above them. The shell pads its header and title area; a negative margin may
// reclaim only the title area's own padding.
func titleTrailing(c *ui.Context, size float32) float32 {
	if runtime.GOOS == "windows" {
		return 0 // caption buttons follow the title area
	}
	bar := c.TitleBar()
	vertical := (sidebar.HeaderHeight(c) - size) / 2
	return max(-8, vertical-max(8, bar.Right)-8)
}

// repositoryLoading reports whether the view is reading the selected
// worktree.
func (s *settings) repositoryLoading() bool { return s.repo.view != nil && s.repo.view.Loading() }
