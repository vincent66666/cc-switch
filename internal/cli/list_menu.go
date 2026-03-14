package cli

import "strings"

type listMenuMode int

const (
	listMenuModeProfiles listMenuMode = iota
	listMenuModeActions
	listMenuModeDeleteConfirm
)

type listMenuAction string

const (
	listMenuActionSwitch listMenuAction = "切换"
	listMenuActionEdit   listMenuAction = "修改"
	listMenuActionRename listMenuAction = "重命名"
	listMenuActionRemove listMenuAction = "删除"
	listMenuActionBack   listMenuAction = "返回"
)

var listMenuActions = []listMenuAction{
	listMenuActionSwitch,
	listMenuActionEdit,
	listMenuActionRename,
	listMenuActionRemove,
	listMenuActionBack,
}

var listMenuCurrentActions = []listMenuAction{
	listMenuActionSwitch,
	listMenuActionEdit,
	listMenuActionRename,
	listMenuActionBack,
}

type listMenuConfirmAction string

const (
	listMenuConfirmDelete listMenuConfirmAction = "确认删除"
	listMenuConfirmCancel listMenuConfirmAction = "取消"
)

var listMenuConfirmActions = []listMenuConfirmAction{
	listMenuConfirmDelete,
	listMenuConfirmCancel,
}

type listMenu struct {
	profiles     []string
	currentName  string
	descriptions map[string]string
	index        int
	mode         listMenuMode
	actionIndex  int
	confirmIndex int
}

func (m listMenu) actions() []listMenuAction {
	if m.selectedProfile() == m.currentName {
		return listMenuCurrentActions
	}

	return listMenuActions
}

func (m *listMenu) moveUp() {
	switch m.mode {
	case listMenuModeDeleteConfirm:
		m.confirmIndex--
		if m.confirmIndex < 0 {
			m.confirmIndex = len(listMenuConfirmActions) - 1
		}
	case listMenuModeActions:
		m.actionIndex--
		if m.actionIndex < 0 {
			m.actionIndex = len(m.actions()) - 1
		}
	default:
		profiles := m.orderedProfiles()
		if len(profiles) == 0 {
			return
		}
		m.index--
		if m.index < 0 {
			m.index = len(profiles) - 1
		}
	}
}

func (m *listMenu) moveDown() {
	switch m.mode {
	case listMenuModeDeleteConfirm:
		m.confirmIndex++
		if m.confirmIndex >= len(listMenuConfirmActions) {
			m.confirmIndex = 0
		}
	case listMenuModeActions:
		m.actionIndex++
		if m.actionIndex >= len(m.actions()) {
			m.actionIndex = 0
		}
	default:
		profiles := m.orderedProfiles()
		if len(profiles) == 0 {
			return
		}
		m.index++
		if m.index >= len(profiles) {
			m.index = 0
		}
	}
}

func (m *listMenu) enterActions() {
	if len(m.profiles) == 0 {
		return
	}
	m.mode = listMenuModeActions
	m.actionIndex = 0
}

func (m *listMenu) backToList() {
	m.mode = listMenuModeProfiles
	m.actionIndex = 0
	m.confirmIndex = 0
}

func (m *listMenu) backToActions() {
	m.mode = listMenuModeActions
	m.confirmIndex = 0
}

func (m *listMenu) enterDeleteConfirm() {
	m.mode = listMenuModeDeleteConfirm
	m.confirmIndex = 0
}

func (m listMenu) selectedProfile() string {
	profiles := m.orderedProfiles()
	if len(profiles) == 0 || m.index < 0 || m.index >= len(profiles) {
		return ""
	}
	return profiles[m.index]
}

func (m listMenu) selectedAction() listMenuAction {
	actions := m.actions()
	if m.actionIndex < 0 || m.actionIndex >= len(actions) {
		return ""
	}
	return actions[m.actionIndex]
}

func (m listMenu) selectedConfirmAction() listMenuConfirmAction {
	if m.confirmIndex < 0 || m.confirmIndex >= len(listMenuConfirmActions) {
		return ""
	}
	return listMenuConfirmActions[m.confirmIndex]
}

func (m listMenu) render() string {
	var out strings.Builder

	switch m.mode {
	case listMenuModeDeleteConfirm:
		out.WriteString("删除配置\n")
		out.WriteString("确认删除：" + m.profileDisplayName(m.selectedProfile()) + "\n")
		out.WriteString("目标配置：" + m.profileDisplayName(m.selectedProfile()) + "\n")
		out.WriteString("此操作不可恢复，请再次确认。\n\n")
		for i, action := range listMenuConfirmActions {
			prefix := "  "
			if i == m.confirmIndex {
				prefix = "> "
			}
			out.WriteString(prefix + string(action) + "\n")
		}
	case listMenuModeActions:
		out.WriteString("配置操作\n")
		out.WriteString("操作：" + m.profileDisplayName(m.selectedProfile()) + "\n")
		out.WriteString("目标配置：" + m.profileDisplayName(m.selectedProfile()) + "\n\n")
		out.WriteString("可执行操作：\n")
		for i, action := range m.actions() {
			prefix := "  "
			if i == m.actionIndex {
				prefix = "> "
			}
			out.WriteString(prefix + string(action) + "\n")
		}
	default:
		profiles := m.orderedProfiles()
		if len(profiles) == 0 {
			out.WriteString("配置列表为空\n")
			out.WriteString("还没有可用的配置。\n")
			out.WriteString(interactiveQuitHint + "\n")
			return out.String()
		}

		out.WriteString("配置列表：\n")
		if m.currentName != "" {
			out.WriteString("当前配置：" + m.profileDisplayName(m.currentName) + "\n\n")
		}
		out.WriteString("选择配置：\n")
		for i, profile := range profiles {
			prefix := "  "
			if i == m.index {
				prefix = "> "
			}
			out.WriteString(prefix + m.profileDisplayName(profile) + "\n")
		}
	}

	out.WriteString("\n" + m.shortcutHint() + "\n")
	return out.String()
}

func (m listMenu) shortcutHint() string {
	removeHint := m.removeHintLabel()

	switch m.mode {
	case listMenuModeDeleteConfirm:
		return "↑/↓ 选择  Enter 确认  " + interactiveQuitHint
	case listMenuModeActions:
		return "↑/↓ 选择  Enter 确认  e 编辑  r 重命名  " + removeHint + "  " + interactiveQuitHint
	default:
		return "↑/↓ 选择  Enter 操作  e 编辑  r 重命名  " + removeHint + "  " + interactiveQuitHint
	}
}

func (m listMenu) removeHintLabel() string {
	selected := m.selectedProfile()
	if selected != "" && selected == m.currentName {
		return "d 当前不可用"
	}

	return "d 删除"
}

func (m listMenu) orderedProfiles() []string {
	return prioritizeCurrentProfile(m.profiles, m.currentName)
}

func (m listMenu) profileDisplayName(name string) string {
	return profileListDisplayName(name, m.descriptions[name], name == m.currentName)
}

func prioritizeCurrentProfile(names []string, current string) []string {
	ordered := make([]string, 0, len(names))
	if current != "" {
		for _, name := range names {
			if name == current {
				ordered = append(ordered, current)
				break
			}
		}
	}

	for _, name := range names {
		if name == current {
			continue
		}
		ordered = append(ordered, name)
	}

	return ordered
}
