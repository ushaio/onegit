# Changelog

User-facing release notes for Onegit.

## [Unreleased]

- Browse many Git repositories in one native window; it reopens with the same size and sidebar layout.
- Add folders to the sidebar from its header, its menu or the command palette, and remove them there; the first folder replaces the old single project directory. Choose the theme and language in Settings, saved under `~/.onegit/` with legacy settings imported.
- Scan every added folder in the background into one project tree that drops projects no longer on disk and remembers expansion.
- Pin projects, or sort them into your own groups that you rename, delete and drag into order; projects start in Ungrouped Projects and every group collapses independently.
- Open a project's folder, choose an image from it as the project's icon, or remove the project from the sidebar, from its menu; a removed project stays hidden until you undo or restore it.
- Show the host's mark, GitHub, Gitee or GitCode, on a repository without a chosen icon.
- Find projects and open Settings from the command palette (`⌘F`).
- See a project's worktrees above its branches when it has several and double-click one to switch to it, returning to the last project and worktree.
- Open the shown worktree in the file manager, a terminal or a development tool, chosen or hidden in Settings, or the repository's GitHub page, from buttons after the title bar's Git actions.
- Open a built-in terminal in the repository's main worktree from the title bar's terminal menu or the project's menu, and keep it running while browsing Git.
- Save terminal launch shortcuts per repository, each opening a tab from a form with its name, start command and working directory; drag tabs into order and close them from their menu.
- Fetch, pull, push and stash from the middle of the title bar, beside the project's name, with dialogs and commit counts; fetch every remote or one branch.
- Push a repository without remotes to a new GitHub repository, private unless you choose public, using the GitHub CLI.
- Browse changes, commits, filterable branches, remote branches, tags and stashes, whose groups and folders stay as you left them, open their commits, delete refs and see stashed files.
- Double-click a ref or commit to check it out, keeping, stashing or discarding changes; a branch checked out elsewhere opens its worktree.
- Apply a stash by double-clicking it, deleting it afterwards unless you keep it or it conflicts.
- Rebase the checked-out branch from a dialog that names both sides, tests for conflicts, and can update dependent branches or stash local changes; rebase interactively with reword, squash, fixup and drop.
- Continue or abort a stopped merge, rebase, cherry-pick or revert, and resolve a conflicted file with either side before staging it.
- Act on refs and commits from their menus: checkout, worktree, pull, merge, rebase, branch, tag, reset, cherry-pick, revert, patch, delete or copy; a commit a branch, remote branch or tag points at also acts as that ref.
- Branch menus also push, fast-forward, rename, change tracking and delete a branch with its worktree; choose a merge type and delete or push several branches or tags at once.
- Deleting a branch warns about commits its upstream or the checked out branch lacks, and deletes it anyway only when you choose to lose them.
- Fetch, edit or delete remotes from their menu, fetch one automatically when a worktree opens, view a GitHub remote or copy its address.
- Create branches and tags from their group menus, and remove the stale local branches you choose.
- Apply, rename or delete a stash from its menu, or save its patch.
- Split changes into unstaged and staged lists, stage, unstage, discard or stash chosen files, copy paths, stage or discard changes and selected lines, and amend, commit (`⌘Enter`) and push from the commit box with a per-worktree draft.
- Open, save, reveal or reset changed and committed files from their menus, and open their folders in the file manager.
- See the full commit graph with collapsible merges, find, avatars, and commit details with diffs and file contents.
- Pin refs, show lost commits from the reflog, search history by message, author, changed path or content, compare commits or a commit with the worktree, and open a file's history or blame.
- Install Onegit on macOS, Windows and Linux from a disk image, an installer, a Debian package or an archive.
- See every repository of the added folders in a Repositories section beside Pinned, including the ones pinned or moved to a group.
- Avoid the UI freezing while choosing a folder.
- Reinstalling on Windows starts from the folder Onegit is already installed in, instead of always offering `%LOCALAPPDATA%\Programs\Onegit`.
