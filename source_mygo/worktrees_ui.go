package main

import (
	"context"
	"path/filepath"
)

// projectOverview holds the selected repository's worktrees and the one the
// page shows, which the Git view's Worktrees group switches.
type projectOverview struct {
	Worktrees     []worktree
	SelectedPath  string
	selectedIndex int
	Loading       bool
	Problem       string
	version       uint64
	cancel        context.CancelFunc
}

func (s *settings) cancelWorktrees() {
	if s.overview.cancel != nil {
		s.overview.cancel()
	}
	s.overview = projectOverview{version: s.overview.version + 1}
	if s.repo.view != nil {
		s.repo.view.SetPath("")
	}
}

func (s *settings) selectProject(node *projectNode, onMain func(func())) {
	s.cancelWorktrees()
	s.repo.dispatch = onMain
	if node != s.selectedProject {
		// Another repository opens on its Git page.
		s.content = contentGit
	}
	s.selectedProject = node
	if node == nil || !node.Repository {
		return
	}
	if s.state.SelectedProjects == nil {
		s.state.SelectedProjects = make(map[string]string)
	}
	s.state.SelectedProjects[globalRoot] = node.Path
	s.saveSidebarState()
	s.overview.Loading = true
	s.readWorktrees(node, onMain)
}

// reloadWorktrees lists the selected repository's worktrees again, such as
// after the view added one, keeping the shown worktree.
func (s *settings) reloadWorktrees() {
	if node := s.selectedProject; node != nil && node.Repository {
		s.readWorktrees(node, s.repositoryDispatch())
	}
}

// readWorktrees lists the repository's worktrees in the background, then
// shows the remembered one, dropping an earlier listing still running.
func (s *settings) readWorktrees(node *projectNode, onMain func(func())) {
	if s.overview.cancel != nil {
		s.overview.cancel()
	}
	s.overview.version++
	version := s.overview.version
	ctx, cancel := context.WithCancel(context.Background())
	s.overview.cancel = cancel
	go func() {
		defer cancel()
		items, err := listWorktrees(ctx, node.Path)
		onMain(func() {
			if s.overview.version != version {
				return
			}
			s.overview.Loading = false
			s.overview.Worktrees = items
			s.overview.SelectedPath = ""
			s.overview.Problem = ""
			if err == nil {
				s.applyWorktreeSelection(node.Path)
			}
			if err != nil {
				s.overview.Problem = err.Error()
			}
			if view := s.gitView(); view.Path() != s.overview.SelectedPath {
				view.SetPath(s.overview.SelectedPath)
			}
			if s.window != nil {
				s.window.Invalidate()
			}
		})
	}()
}

// applyWorktreeSelection restores the remembered worktree of a repository and
// falls back to the first entry Git returned.
func (s *settings) applyWorktreeSelection(project string) {
	s.overview.SelectedPath = ""
	s.overview.selectedIndex = -1
	if len(s.overview.Worktrees) == 0 {
		return
	}
	wanted := s.state.SelectedWorktrees[project]
	for i, item := range s.overview.Worktrees {
		if item.Path == wanted {
			s.overview.SelectedPath = item.Path
			s.overview.selectedIndex = i
			break
		}
	}
	if s.overview.SelectedPath == "" {
		s.overview.SelectedPath = s.overview.Worktrees[0].Path
		s.overview.selectedIndex = 0
	}
	s.rememberWorktreeSelection()
}

// selectWorktree switches the active worktree and reloads its repository
// data. It is a navigation change, never a Git branch checkout.
func (s *settings) selectWorktree(index int) {
	if index < 0 || index >= len(s.overview.Worktrees) || index == s.overview.selectedIndex {
		return
	}
	s.overview.selectedIndex = index
	s.overview.SelectedPath = s.overview.Worktrees[index].Path
	s.rememberWorktreeSelection()
	s.repositoryChanged()
}

// openWorktree shows the worktree at path, as Git lists it, in place of the
// current one.
func (s *settings) openWorktree(path string) {
	path = filepath.Clean(filepath.FromSlash(path))
	for i, item := range s.overview.Worktrees {
		if item.Path == path {
			s.selectWorktree(i)
			return
		}
	}
}

func (s *settings) rememberWorktreeSelection() {
	if s.selectedProject == nil || !s.selectedProject.Repository {
		return
	}
	if s.state.SelectedWorktrees == nil {
		s.state.SelectedWorktrees = make(map[string]string)
	}
	s.state.SelectedWorktrees[s.selectedProject.Path] = s.overview.SelectedPath
	s.saveSidebarState()
}

// projectName labels a path by its last element. A volume root such as D:\
// has no last element, so its volume names it.
func projectName(path string) string {
	label := filepath.Base(path)
	if label == "" || label == "." || label == string(filepath.Separator) {
		if volume := filepath.VolumeName(path); volume != "" {
			return volume + string(filepath.Separator)
		}
		return path
	}
	return label
}
