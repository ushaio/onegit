# Repository page

The repository page shows one selected project. The window's title bar holds the project's name on the left, the view's toolbar in the middle and the page tabs and launchers on the right, as in Neorio, rather than a strip of their own. A hairline separates it from the [Git view](git-view.md) of the selected worktree. Onegit owns the title bar's layout and which worktree is shown; the view owns its toolbar and everything below it.

## Worktrees

The view's Worktrees group, below Current changes and All commits and above the refs, lists every worktree Git reports for the repository, including worktrees outside the parent directory. It shows only for more than one worktree. The main worktree comes first, named after its directory: the project's name; the shown worktree's name is bold.

Double-clicking another worktree, or a branch it has checked out, switches the page to it in place. That only changes the path the view shows. A worktree added from a ref's menu joins the group once Git lists it, one deleted with its branch leaves it, and the shown worktree stays. Switching projects or worktrees, or closing the window, drops in-flight reads and discards their results.

## Title bar actions

The view's `Toolbar`, shared with Neorio's Git window, is centered between the name and the launchers, which keep their width while the name truncates: raised capsules with an icon and name for Fetch, Pull, Push and Stash, and commit counts after the name. A running action shows a spinner in place of its icon. The view runs one action at a time per worktree, shows failures as toasts and rereads the worktree after each action.

Raised icon buttons on the right are drawn with each kind's glyph rather than the application's icon:

- The file manager, terminal and development tool open the selected worktree. Onegit finds them with the shared `launchers` package off the main thread at startup; Settings chooses the terminal and tool, or hides either, saved as `terminal` and `editor` in `config.yaml`.
- GitHub opens the view's `GitHubPage`: the page of the remote the branch tracks, else of `origin` or the only remote, shown only when that is on GitHub.
- A launcher that fails to open shows a toast.

## Built-in terminal

The terminal button is a menu: the chosen external terminal, when one is installed, and Onegit's built-in terminal. The project's menu opens the same built-in terminal at the repository's main worktree.

The Git and Terminal page tabs sit on the title bar's right, and switching keeps every terminal session running. The built-in terminal fills the content beside a list of the repository's launch shortcuts, saved in `state.yaml` under `terminalPresets`.

Each shortcut opens a tab from a form asking its name, start command and working directory; the working directory is read-only and picked from a folder chooser starting at the current path. The command is typed into a login shell so the shell stays after it exits. A shortcut reuses the untouched default shell tab instead of opening a second one.

Tabs switch between the open terminals and can be dragged into another order. A tab closes from its menu, which also closes the others or those to its right, or from the close button; switching to Terminal with none open starts a shell.

## Onegit's choices for the view

- Watching is off: the view reads when a worktree is shown and after an action.
- The Worktrees group is on; no ref filter and no inserted history rows.
- The view's layout, ref group disclosure, line wrapping, drafts, pinned refs, reflog choice, merge type, fetch, pull, push, stash, rebase and cherry-pick options, and remotes fetched automatically are saved in `state.yaml` under `gitView`.
- Commit authors' avatars are cached in `~/.onegit/avatars/`, downloaded from GitHub or Gravatar.

## Selection defaults

The last selected project and the worktree per project are saved in state. Without a valid saved choice, Onegit selects:

- the first available project, in sidebar order, ignoring collapse;
- the first worktree Git lists.

The view shows the worktree's changes when it has some, else all commits. A missing saved project falls back to an available one. With nothing to select, the content area stays blank.
