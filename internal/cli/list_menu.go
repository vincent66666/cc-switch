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
	listMenuActionRemove listMenuAction = "删除"
	listMenuActionBack   listMenuAction = "返回"
)

var listMenuActions = []listMenuAction{
	listMenuActionSwitch,
	listMenuActionEdit,
	listMenuActionRemove,
	listMenuActionBack,
}

var listMenuCurrentActions = []listMenuAction{
	listMenuActionSwitch,
	listMenuActionEdit,
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
		out.WriteString("确认删除：" + m.selectedProfile() + "\n")
		for i, action := range listMenuConfirmActions {
			prefix := "  "
			if i == m.confirmIndex {
				prefix = "> "
			}
			out.WriteString(prefix + string(action) + "\n")
		}
	case listMenuModeActions:
		out.WriteString("操作：" + m.selectedProfile() + "\n")
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
			out.WriteString(interactiveQuitHint + "\n")
			return out.String()
		}

		out.WriteString("配置列表：\n")
		for i, profile := range profiles {
			prefix := "  "
			if i == m.index {
				prefix = "> "
			}
			out.WriteString(prefix + m.displayProfile(profile) + "\n")
		}
	}

	out.WriteString(interactiveQuitHint + "\n")
	return out.String()
}

func (m listMenu) orderedProfiles() []string {
	return prioritizeCurrentProfile(m.profiles, m.currentName)
}

func (m listMenu) displayProfile(name string) string {
	if name == m.currentName {
		return name + "（当前）"
	}

	return name
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
