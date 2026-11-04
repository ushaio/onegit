package main

import "strings"

func (s *settings) text(zh string) string {
	language := s.values.Language
	if language == "" {
		language = s.systemLanguage
	}
	if strings.HasPrefix(strings.ToLower(language), "zh") {
		return zh
	}
	if en, ok := english[zh]; ok {
		return en
	}
	return zh
}

var english = map[string]string{
	"搜索项目与设置": "Search projects and settings", "搜索…": "Search…", "搜索项目与设置…": "Search projects and settings…", "命令面板": "Command palette", "没有匹配的项目或设置": "No matching projects or settings",
	"正在读取 worktree…": "Loading worktrees…", "读取 worktree 失败：": "Could not load worktrees: ",
	"置顶": "Pinned", "仓库": "Repositories", "未分组项目": "Ungrouped Projects", "置顶项目": "Pin Project", "取消置顶": "Unpin Project",
	"移到分组": "Move to Group", "新建分组…": "New Group…", "新建分组": "New Group", "重命名分组…": "Rename Group…", "重命名分组": "Rename Group",
	"删除分组": "Delete Group", "分组名称": "Group name", "创建": "Create", "重命名": "Rename", "取消": "Cancel",
	"打开文件夹": "Open Folder", "移除项目": "Remove Project", "恢复已移除的项目": "Restore Removed Projects",
	"无法打开目录": "Could not open folder", "选择图标…": "Choose Icon…", "恢复默认图标": "Reset Icon",
	"选择项目图标": "Choose project icon", "图片": "Images", "无法使用该图片": "Could not use this image", "已移除项目": "Project removed", "撤销": "Undo",
	"（已不存在）":      " (missing)",
	"读取状态失败：":     "Could not load state: ",
	"保存状态失败：":     "Could not save state: ",
	"扫描未完成：":      "Scan incomplete: ",
	"未找到 Git 项目。": "No Git projects found.",
	"Git 项目":      "Git project",
	"从侧边栏选择项目。分支状态与实时更新将在后续实现。": "Select a project in the sidebar. Branch status and live updates are coming later.",
	"语言偏好": "Language preference",
	"设置":   "Settings", "设置…": "Settings…", "关闭设置": "Close settings",
	"主题": "Theme", "主题偏好": "Theme preference", "浅色": "Light", "深色": "Dark", "跟随系统": "System",
	"通用": "General", "应用": "Application", "项目": "Projects", "外观": "Appearance", "语言": "Language",
	"修改后立即生效，并自动保存。": "Changes apply immediately and are saved automatically.",
	"与侧边栏底部的主题选项同步。": "Shared with the theme menu at the bottom of the sidebar.",
	"添加文件夹": "Add Folder", "添加文件夹…": "Add Folder…", "移除文件夹": "Remove Folder", "该文件夹已在侧栏中": "That folder is already in the sidebar",
	"该文件夹已包含在「%s」中": "That folder is already inside “%s”", "「%s」已包含在此文件夹中": "“%s” is already inside that folder",
	"选择要加入侧栏的文件夹。": "Choose a folder to show in the sidebar.", "添加": "Add", "选择": "Choose",
	"读取设置失败：": "Could not load settings: ", "选择目录失败：": "Could not choose folder: ", "无法访问所选目录：": "Could not access folder: ", "请选择文件夹。": "Please choose a folder.",
	"无法确定设置保存位置。": "Could not determine where to save settings.", "保存失败，设置未更改：%v": "Could not save; settings remain unchanged: %v",
	"搜索设置": "Search settings", "搜索设置…": "Search settings…", "设置分类": "Settings category", "搜索结果": "Search results", "没有匹配的设置": "No matching settings",
	"工具": "Tools", "终端": "Terminal", "开发工具": "Development tool", "自动": "Automatic", "自动（%s）": "Automatic (%s)", "%s（未安装）": "%s (not installed)", "不显示": "Don't show",
	"在 %s 中打开": "Open in %s", "在 GitHub 上打开": "Open on GitHub", "无法打开 %s": "Could not open %s",
	"进入终端": "Open Terminal", "终端菜单": "Terminal menu", "在内置终端中打开": "Open in Built-in Terminal", "关闭终端": "Close Terminal", "无法打开终端": "Could not open terminal",
	"关闭": "Close", "关闭其他": "Close Others", "关闭右侧": "Close to the Right",
	"选择工作目录": "Choose Working Directory", "选择终端启动时所在的文件夹。": "Choose the folder the terminal starts in.",
	"终端列表": "Terminals", "新建终端": "New Terminal", "编辑终端": "Edit Terminal", "还没有终端": "No terminals yet", "选择或新建终端": "Select or create a terminal",
	"名称": "Name", "启动命令": "Start command", "工作目录": "Working directory", "编辑": "Edit", "删除": "Delete", "保存": "Save",
	"切换侧边栏": "Toggle sidebar", "调整侧边栏宽度": "Resize sidebar", "尚未接入仓库": "No repositories connected",
	"欢迎使用 Onegit": "Welcome to Onegit", "在一个地方查看多个 Git 仓库与 worktree。": "View multiple Git repositories and worktrees in one place.",
	"应用已初始化。仓库接入、分支状态与实时更新将在后续实现。": "The app is ready. Repository integration, branch status and live updates are coming later.",
}
