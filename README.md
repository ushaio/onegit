# Onegit

**English** | [简体中文](README.zh-CN.md)

Onegit is a worktree-centered macOS app that gathers many Git repositories in one window. It covers everyday repository browsing and worktree switching, not every feature of a general Git client.

![Onegit showing a project's commit graph with a commit menu open](docs/images/screenshot.jpg)

## Features

- Add folders to the sidebar and browse the Git repositories under each one in a tree; add as many folders as you like.
- Pin projects above the directory tree, or sort them into your own groups and drag the groups into order.
- See every repository of the added folders in one Repositories list beside Pinned, even the pinned ones.
- Open a project's folder, choose an image as its icon, or remove it from the sidebar in its menu, with undo and restore.
- Find projects and settings with the command palette (`⌘F`).
- Switch between a project's worktrees by double-clicking one above its branches; open the shown one in the file manager, a terminal or a development tool chosen in Settings, or the repository on GitHub, from the title bar.
- Review staged and unstaged changes, stage whole files, single changes or selected lines, discard a change or selected lines, stash chosen files or copy their paths, and commit them with a message.
- Amend the last commit, commit and push in one action, and keep a draft message per worktree.
- Browse the full commit graph with collapsible merges, find in history, and commit details with diffs and file contents.
- Pin refs, show lost commits from the reflog, search history by message, author, changed path or content, compare commits or a commit with the worktree, and open a file's history or blame.
- Open or reveal a changed or committed file, open a folder in the file manager, save the version shown, or reset a file to a commit's or its parent's version from its menu.
- Browse local branches, remote branches and tags, create branches or tags from their group menus, delete stale local branches, and jump to their commits.
- Double-click a ref or commit to check it out, or use its menu for checkout, worktrees, pull, merge, rebase, branch, tag, delete and copy; branch menus also push, fast-forward, rename and track.
- Manage remotes from their menu: fetch, edit or delete one, fetch it when a worktree opens, open it on GitHub or copy its address.
- Rebase from a dialog that reports conflicts and can update dependent branches or stash local changes; rebase interactively with reword, squash, fixup and drop.
- Continue or abort a stopped merge, rebase, cherry-pick or revert, resolving a conflicted file with either side and staging it.
- Reset, cherry-pick, revert or save a commit as a patch, and apply, rename or delete a stash, from their menus.
- Choose the merge type, including squash and no commit, and delete or push several branches or tags at once.
- Every Git write, such as fetch, pull, push, stash or a menu action, runs only when you click, except remotes you set to fetch automatically when a worktree opens.

## Run

Onegit is not released yet. Build it from source as described in [Development](#development). Git must be on `PATH` at runtime.

## Getting started

1. Add a folder with the `+` at the top of the sidebar, its right-click menu or the command palette (`⌘F`); folders holding your projects work best.
2. Wait for the background scan to add repositories to the sidebar.
3. Select a project; with several worktrees, double-click one under Worktrees to show it.
4. Choose Current changes or All commits in the project sidebar.

## Development

Requirements: macOS, Go 1.27.3 or later, and Xcode Command Line Tools. The MyGo CLI is a Go tool dependency in [go.mod](source_mygo/go.mod), so no global install is needed.

Run from `source_mygo/`:

```sh
cd source_mygo
GOWORK=off go mod download
go tool mygo dev
```

`go tool mygo dev` runs `Onegit Dev` and restarts it after code changes.

Format, test, vet and build from `source_mygo/`:

```sh
gofmt -l .
GOWORK=off go test ./...
GOWORK=off go test -race ./...
GOWORK=off go tool mygo vet
GOWORK=off go tool mygo build
```

Tests are headless and never open windows or native dialogs. The directory and icon pickers, packaging and the app icon need manual review. Builds go to `dist/<os>-<arch>/` with ad-hoc signing and no notarization.

Build release packages for every platform from `source_mygo/`: a universal macOS app and disk image, Windows installers and Linux archives and Debian packages. Windows installers need NSIS (`brew install makensis`).

```sh
GOWORK=off go tool mygo build -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64
```

Neorio users get the same Run, Build, Test and Vet tasks from [.neorio/config.yaml](.neorio/config.yaml); Build makes these release packages.

Read [ARCHITECTURE.md](ARCHITECTURE.md) before changing code, and [AGENTS.md](AGENTS.md) for working rules. User-visible changes are listed in [CHANGELOG.md](CHANGELOG.md).

## Author

[Blog](https://oneo.me) · [X @oneo666](https://x.com/oneo666)

## License

[MIT](LICENSE) © 2026 ONEO.
