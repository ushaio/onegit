# Project discovery

Project discovery turns the folders the user adds into the sidebar tree. Every added folder is a top-level row; pinning, groups and the command palette span all of them. It scans each folder once per startup or folder change, caches the result per folder and never watches the file system.

## Scanning

The scan walks each added folder recursively in the background. Each candidate is confirmed with `git`, which resolves the real repository root.

- Recognized: regular repositories, linked worktrees, initialized submodules and bare repositories.
- Skipped: `.git` internals, symlinked directories, plain folders and invalid `.git` entries.
- Recursion stops at the first valid repository, so nested repositories and submodules are not listed.
- Each Git check has a timeout. Changing the folders or closing the window cancels the scan.
- Only repositories and the folders leading to them are kept. Paths, not display names, identify nodes.

The scan runs silently with no progress text. A failed directory keeps the projects already found and shows the first error.

## Cache and merging

The cache is keyed by folder, so removing and adding a folder back restores its tree. Scan results merge on the main thread by path.

- New nodes are added; existing nodes and their expansion are kept.
- Paths that no longer exist are dropped with their pinning and removal, as are folders they leave empty. A missing folder, such as an unmounted volume, keeps the cache.
- A stale scan for a previous folder never overwrites the current one.

State is saved after a merge, after pin, group or expansion changes and when the window closes normally.

## Sidebar groups

The sidebar shows collapsible groups: Pinned, Repositories, the user's groups and Ungrouped Projects, in that order. All start expanded except Repositories, which starts collapsed because it lists every repository.

- Pinned, Repositories and user groups list their projects flat, sorted by name case-insensitively, then by full path.
- Repositories lists every repository of every added folder, including the ones pinned or moved to a group, so it is the complete list beside the folder tree. It is fixed: it cannot be renamed, deleted or dragged.
- Ungrouped Projects holds every other project in the folder hierarchy. Each added folder is a top-level row there and keeps its row even when every project under it is pinned, grouped or removed.
- Pinning wins over a group; unpinning returns a project to its group. A project naming a deleted group is ungrouped.
- Groups are a view over the cached nodes. They do not change the tree, selection or expansion.

Group headers offer New Group; user groups also rename or delete, which returns their projects to Ungrouped Projects. Dragging a group header onto another group moves it there; Pinned stays first.

Pinning, group membership and icons live in the cached nodes, and the group order, names and collapse in `groups`, all shared by every folder. Pinned, Repositories and Ungrouped Projects share their collapse.

Nodes start collapsed. The sidebar follows Neorio's Git window: Onevra `SidebarItem` rows in a `ScrollView`, and `Collapsible` headings that count their projects.

- Repositories lead with their chosen icon, else the mark of their remote's host (GitHub, Gitee or GitCode), else a blank placeholder tile; folders with a folder icon.
- Clicking a folder, Right and Left open and close it; folders are never selected.

## Context menus

Right-clicking a folder offers Open Folder. A folder the user added also offers Add Folder…, Restore Removed Projects once one of its projects was removed, and Remove Folder. A repository row also offers pinning, Move to Group, which can make a new group for it, Choose Icon, Reset Icon once one is set, and Remove Project.

- The icon picker opens in the project's folder and accepts PNG, JPEG, GIF, WebP, BMP and SVG. An image that does not decode leaves the icon unchanged and shows a toast.
- The node stores the image's path. Images load in the background on first use; an unreadable one shows the placeholder.

- Removal hides the project from every group, the command palette and selection fallback without touching its files.
- The flag lives in the cached tree, so scans and restarts keep the project hidden.
- The toast after a removal undoes it; the palette's Restore Removed Projects command brings back every project removed from the sidebar.
- Opening a folder runs `Shell.OpenPath`; a failure shows as a toast.

## Command palette

The palette searches every cached project of every added folder, including collapsed ones but not removed ones. It matches names and paths and opens the same node the sidebar would. While a project is hidden, it also lists Restore Removed Projects.

The search box above the sidebar and `⌘F` open it. It also lists theme, language and folder commands, such as Add Folder…, that open the matching settings row. Onegit owns the command data; Onevra `CommandDialog` owns filtering, keyboard navigation and focus. The palette does not open over the settings dialog or the native directory picker.
