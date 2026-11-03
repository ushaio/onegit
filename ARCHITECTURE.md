# Onegit architecture

Onegit is a single Go module in [source_mygo](source_mygo/) built on [Onevra](https://github.com/oneo-me/onevra) components and the MyGo runtime. It has three moving parts: project discovery for the sidebar, a repository page for the selected project, and settings with persisted state. All Git access goes through the `git` executable on `PATH`; only creating a GitHub repository runs the GitHub CLI, `gh`.

Onevra and MyGo versions are pinned in [go.mod](source_mygo/go.mod). Onevra provides themed controls and the `shells/sidebar` window shell; Onegit owns application state, persistence and background work.

## Code map

| Path | Responsibility |
| --- | --- |
| `source_mygo/main.go` | MyGo startup, window creation and the app menu |
| `source_mygo/settings*.go` | Settings dialog, preferences file, theme and language |
| `source_mygo/localization.go` | UI strings keyed by their Chinese text, with English translations |
| `source_mygo/commands.go` | Command palette data and actions |
| `source_mygo/projects*.go`, `source_mygo/state.go` | Project scan, sidebar tree, pinning, groups, removal and persisted sidebar state |
| `source_mygo/worktrees*.go` | Worktree listing and the shown worktree |
| `source_mygo/repository_ui.go`, `source_mygo/launchers_ui.go` | Repository page and its title bar, with its launchers and their settings |
| `source_mygo/gitview/` | Reusable Git view shared with Neorio: toolbar, refs pane, changes, commit graph, details, dialogs and Git CLI layer |
| `source_mygo/launchers/` | File managers, terminals and development tools shared with Neorio: catalog, detection and opening a directory |
| `source_mygo/assets/` | App icon: an Icon Composer project for macOS, PNG and ICO for others |
| `source_mygo/mygo.json` | Packaging; output goes to `dist/` |

## Boundaries

- The worktree is the unit of context. Branch and working tree state belong to a worktree, never to the repository as a whole.
- Selecting a worktree changes what Onegit reads. It never runs `git checkout` or `git switch`.
- Git write commands run only from an explicit user action: fetch, pull, push, stash, checkout, worktree creation, merge, rebase, fast-forward, branch, tag, track, rename, delete refs, stage, discard, commit, reset, cherry-pick and revert.
- A remote the user set to fetch automatically is the only write that runs without a click, when a worktree opens; nothing polls.
- Status is read when a project is entered and after a user action. Onegit does not poll or watch files.
- Git calls run off the main thread with a clean Git environment. Results return to the main thread and are dropped if the project or worktree changed meanwhile.
- The sidebar tree mirrors the disk, except projects the user removed. It does not infer links between a repository and worktrees outside the added folders.
- Projects whose paths are gone leave the cache on the next scan; their files are never touched. Only a confirmed missing path counts; permission or Git errors and a missing added folder do not.
- Projects switch through the sidebar only, and worktrees through the view's Worktrees group.
- Empty states stay blank, with no welcome or placeholder text. Errors are always shown.

## Core flows

**Startup.** Onegit restores the cached tree of each added folder, then scans them in the background. Scan results merge into the cache by path and keep the user's expansion changes. See [Project discovery](docs/architecture/project-discovery.md).

**Opening a project.** Selecting a repository lists all its worktrees, including those outside the added folders. Onegit then reads changes, commits and refs for the selected worktree. See [Repository page](docs/architecture/repository-page.md).

**User Git actions.** The Git view's toolbar in the title bar runs one action at a time per worktree in the background, so other repositories and worktrees stay free. Failures show as toasts, and the view rereads the worktree when the action finishes.

## Data and state

Configuration lives in `~/.onegit/config.yaml` and UI state in `~/.onegit/state.yaml`. The same directory is the MyGo user data directory.

- Both files are written to a temporary file in the same directory, synced, then renamed into place.
- In-memory values change only after a successful write. Load and save errors appear in the UI.
- When `~/.onegit` does not exist, an earlier `~/.oneo/onegit` directory is moved there at startup.
- A missing `config.yaml` is imported once from the legacy `Onegit/settings.json` in the user config directory, which is left in place.
- State covers the main window's bounds and state through MyGo's window state store, sidebar width and collapse, project cache, expansion, pinning, project icons, project groups with their order and collapse, and last selections.
- It also covers the Git view's layout, options, drafts, pinned refs, reflog choice, merge type and automatic fetch choices per repository.
- Commit authors' avatars are downloaded from GitHub or Gravatar and cached in `avatars/` in the same directory.
- Worktree lists and Git data are never persisted.

## Verification

`go test` covers settings and persistence, project detection, scan cancellation and cache merging, sidebar menus, project groups and removal, legacy import, worktree switching, and the Git view's reads, graph layout, ref filtering, inserted rows and actions. It also covers conflict resolution and stopped operations, draft commits, reflog, pinned refs, comparisons, file history, blame, history search, interactive rebase and bulk ref actions. Tests use real temporary Git repositories.

Review these manually: the native directory and icon pickers, and packaging and the app icon on each platform.

## Subjects

| Document | When to read |
| --- | --- |
| [Project discovery](docs/architecture/project-discovery.md) | Changing scanning, the sidebar tree, pinning, groups or the project cache |
| [Repository page](docs/architecture/repository-page.md) | Changing worktree switching, the title bar actions or how Onegit configures the Git view |
| [Git view](docs/architecture/git-view.md) | Changing `source_mygo/gitview` or embedding it in another app |
