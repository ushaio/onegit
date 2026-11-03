package main

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
)

// ungroupedID is the reserved ID of Ungrouped Projects in a parent directory's
// group order. It has no name and cannot be renamed or deleted.
const ungroupedID = "ungrouped"

// projectGroup is a user-made sidebar group. Projects name their group by ID,
// so renaming a group keeps its projects.
type projectGroup struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name,omitempty"`
	Open *bool  `yaml:"open,omitempty"`
}

// groupDrag is the value a group header drags to reorder groups.
type groupDrag struct{ id string }

// groupDialog names a new group or renames one. A new group made from a
// project's menu takes that project.
type groupDialog struct {
	open    bool
	groupID string
	project *projectNode
	name    string
}

// projectGroups returns the sidebar's groups in order, always including
// Ungrouped Projects. Every added folder shares them.
func (s *settings) projectGroups() []*projectGroup {
	groups := s.state.Groups[globalRoot]
	for _, group := range groups {
		if group.ID == ungroupedID {
			return groups
		}
	}
	groups = append(groups, &projectGroup{ID: ungroupedID})
	if s.state.Groups == nil {
		s.state.Groups = make(map[string][]*projectGroup)
	}
	s.state.Groups[globalRoot] = groups
	return groups
}

func (s *settings) findGroup(id string) *projectGroup {
	for _, group := range s.projectGroups() {
		if group.ID == id {
			return group
		}
	}
	return nil
}

// groupIDs lists the user-made groups that exist, so projects naming any
// other group show as ungrouped.
func (s *settings) groupIDs() map[string]bool {
	ids := map[string]bool{}
	for _, group := range s.projectGroups() {
		if group.ID != ungroupedID {
			ids[group.ID] = true
		}
	}
	return ids
}

// createGroup adds a group just before Ungrouped Projects and returns it.
func (s *settings) createGroup(name string) *projectGroup {
	groups := s.projectGroups()
	next := 1
	for _, group := range groups {
		if n, err := strconv.Atoi(strings.TrimPrefix(group.ID, "g")); err == nil && strings.HasPrefix(group.ID, "g") && n >= next {
			next = n + 1
		}
	}
	group := &projectGroup{ID: "g" + strconv.Itoa(next), Name: name}
	at := len(groups)
	for i, existing := range groups {
		if existing.ID == ungroupedID {
			at = i
		}
	}
	groups = append(groups[:at], append([]*projectGroup{group}, groups[at:]...)...)
	s.state.Groups[globalRoot] = groups
	return group
}

// deleteGroup removes a user-made group; its projects return to Ungrouped
// Projects.
func (s *settings) deleteGroup(id string) {
	if id == ungroupedID {
		return
	}
	groups := s.projectGroups()
	kept := groups[:0]
	for _, group := range groups {
		if group.ID != id {
			kept = append(kept, group)
		}
	}
	s.state.Groups[globalRoot] = kept
	walkProjects(s.projectTree, func(node *projectNode) {
		if node.Group == id {
			node.Group = ""
		}
	})
	s.saveSidebarState()
	s.invalidate()
}

// moveGroup places the group from before the group to, where the pointer
// dropped it.
func (s *settings) moveGroup(from, to string) {
	if from == to {
		return
	}
	groups := s.projectGroups()
	var moved *projectGroup
	kept := make([]*projectGroup, 0, len(groups))
	for _, group := range groups {
		if group.ID == from {
			moved = group
		} else {
			kept = append(kept, group)
		}
	}
	if moved == nil {
		return
	}
	at := len(kept)
	for i, group := range kept {
		if group.ID == to {
			at = i
		}
	}
	// Dragging down puts the group after the target, so every drop moves it.
	for i, group := range groups {
		if group.ID == from {
			if i <= at {
				at++
			}
			break
		}
	}
	at = min(at, len(kept))
	s.state.Groups[globalRoot] = append(kept[:at], append([]*projectGroup{moved}, kept[at:]...)...)
	s.saveSidebarState()
	s.invalidate()
}

func (s *settings) moveProjectToGroup(node *projectNode, id string) {
	if id == ungroupedID {
		id = ""
	}
	node.Group = id
	s.saveSidebarState()
	s.invalidate()
}

func (s *settings) openGroupDialog(groupID string, project *projectNode) {
	s.groupDialog = groupDialog{open: true, groupID: groupID, project: project}
	if group := s.findGroup(groupID); group != nil {
		s.groupDialog.name = group.Name
	}
}

func (s *settings) submitGroupDialog() {
	d := &s.groupDialog
	name := strings.TrimSpace(d.name)
	if name == "" {
		return
	}
	d.open = false
	if group := s.findGroup(d.groupID); group != nil {
		group.Name = name
	} else {
		group = s.createGroup(name)
		if d.project != nil {
			d.project.Group = group.ID
		}
	}
	s.saveSidebarState()
	s.invalidate()
}

// groupMenu adds the items shared by every group header.
func (s *settings) groupMenu(m *ui.Menu, id string) {
	if m.Item(s.text("新建分组…")).Chosen() {
		s.openGroupDialog("", nil)
	}
	if id == "" || id == ungroupedID {
		return
	}
	m.Separator()
	if m.Item(s.text("重命名分组…")).Chosen() {
		s.openGroupDialog(id, nil)
	}
	if m.Item(s.text("删除分组")).Chosen() {
		s.deleteGroup(id)
	}
}

// projectGroupMenu moves a repository between groups.
func (s *settings) projectGroupMenu(m *ui.Menu, node *projectNode) {
	ids := s.groupIDs()
	m.Submenu(s.text("移到分组"), func(m *ui.Menu) {
		for _, group := range s.projectGroups() {
			label := group.Name
			if group.ID == ungroupedID {
				label = s.text("未分组项目")
			}
			current := node.Group == group.ID || group.ID == ungroupedID && !ids[node.Group]
			if m.Item(label).Checked(current).Chosen() {
				s.moveProjectToGroup(node, group.ID)
			}
		}
		m.Separator()
		if m.Item(s.text("新建分组…")).Chosen() {
			s.openGroupDialog("", node)
		}
	})
}

func (s *settings) groupDialogView(c *ui.Context) {
	d := &s.groupDialog
	title := s.text("新建分组")
	action := s.text("创建")
	if d.groupID != "" {
		title, action = s.text("重命名分组"), s.text("重命名")
	}
	ui.Box(c).Key("group-dialog").Absolute().Children(func() {
		components.Dialog(c, &d.open, components.DialogOptions{Label: title}, func(panel ui.Element) {
			t := c.Theme()
			panel.Width(t.Space(90))
			components.Text(c, title).Bold().FontSize(t.Rem(1.1))
			ready := strings.TrimSpace(d.name) != ""
			box := components.TextBox(c, &d.name, components.TextBoxOptions{}).
				FillWidth().Label(s.text("分组名称")).Placeholder(s.text("分组名称")).AutoFocus()
			box.OnTextKey(func(k ui.TextKey) bool {
				if k.Key != ui.KeyEnter || k.Mods != 0 {
					return false
				}
				d.name = k.Text()
				s.submitGroupDialog()
				return true
			})
			ui.Row(c).FillWidth().Wrap().Justify(ui.End).Gap(t.Space(2)).Children(func() {
				if components.Button(c, s.text("取消"), components.ButtonSecondary).Clicked() {
					d.open = false
				}
				if components.Button(c, action, components.ButtonPrimary).Disabled(!ready).Clicked() {
					s.submitGroupDialog()
				}
			})
		})
	})
}
