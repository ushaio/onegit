package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/shells/sidebar"
	"github.com/oneo-me/onevra/source/theme"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	configDir := filepath.Join(home, ".onegit")
	if err := migrateDataDir(configDir, filepath.Join(home, ".oneo", "onegit")); err != nil {
		log.Printf("Move data directory: %v", err)
	}
	if err := os.MkdirAll(configDir, 0700); err != nil {
		log.Fatal(err)
	}
	mygo.App.SetPath(mygo.PathUserData, configDir)
	mygo.App.SetName("Onegit")
	mygo.App.WhenReady(func() {
		settings := newSettings(configDir)
		// MyGo restores and saves the window's bounds through the store.
		mygo.App.SetWindowStateStore(windowStateStore{settings})
		shell := sidebar.NewShell(sidebar.ShellOptions{
			Title:        "Onegit",
			Theme:        theme.Onevra,
			StateKey:     mainWindowStateKey,
			Width:        1100,
			Height:       720,
			MinWidth:     820,
			MinHeight:    600,
			LeftWidth:    200,
			LeftMinWidth: 160,
			LeftMaxWidth: 400,
			Left:         settings.worktreeSidebar,
			LeftToolbar:  settings.toolbar,
			Content: func(c *ui.Context) {
				settings.welcome(c)
				settings.render(c)
				settings.renderCommands(c)
			},
			ContentTitle: settings.repositoryTitle,
		})
		settings.window = shell.Window()
		settings.restoreShell(shell)
		settings.scanProjects()
		settings.detectLaunchers(mygo.RunOnMain)
		mygo.App.SetMenu(settings.menu())
		shell.Window().OnClose(func(*mygo.CloseEvent) { settings.saveSidebarState() })
		shell.Window().OnClosed(func() {
			settings.cancelWorktrees()
			if settings.repo.view != nil {
				settings.repo.view.Close()
			}
			settings.scanVersion++
			if settings.scanCancel != nil {
				settings.scanCancel()
			}
			settings.closeTerminals()
			mygo.App.Quit()
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
