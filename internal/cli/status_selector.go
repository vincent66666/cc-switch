package cli

import "strings"

const interactiveQuitHint = "按 q 或 Ctrl+C 退出"

type statusSelector struct {
	currentName        string
	currentDescription string
	baseURL            string
	model              string
	names              []string
	descriptions       map[string]string
	index              int
}

func (s *statusSelector) moveUp() {
	if len(s.orderedNames()) == 0 {
		return
	}

	s.index--
	if s.index < 0 {
		s.index = len(s.orderedNames()) - 1
	}
}

func (s *statusSelector) moveDown() {
	if len(s.orderedNames()) == 0 {
		return
	}

	s.index++
	if s.index >= len(s.orderedNames()) {
		s.index = 0
	}
}

func (s statusSelector) selectedName() string {
	names := s.orderedNames()
	if len(names) == 0 {
		return ""
	}

	if s.index < 0 || s.index >= len(names) {
		return ""
	}

	return names[s.index]
}

func (s statusSelector) render() string {
	var out strings.Builder
	model := s.model
	if model == "" {
		model = "-"
	}

	out.WriteString("状态总览\n")
	out.WriteString("当前配置：" + profileDisplayName(s.currentName, s.currentDescription) + "\n")
	out.WriteString("接口地址：" + s.baseURL + "\n")
	out.WriteString("模型：" + model + "\n")

	names := s.orderedNames()
	if len(names) == 0 {
		return out.String()
	}

	out.WriteString("\n可用配置：\n")
	out.WriteString("选择配置：\n")
	for i, name := range names {
		prefix := "  "
		if i == s.index {
			prefix = "> "
		}
		out.WriteString(prefix + s.displayName(name) + "\n")
	}

	out.WriteString("↑/↓ 选择  Enter 切换  " + interactiveQuitHint + "\n")
	return out.String()
}

func (s statusSelector) orderedNames() []string {
	names := make([]string, 0, len(s.names)+1)
	if s.currentName != "" {
		names = append(names, s.currentName)
	}

	for _, name := range s.names {
		if name == s.currentName {
			continue
		}
		names = append(names, name)
	}

	return names
}

func (s statusSelector) displayName(name string) string {
	description := s.descriptions[name]
	if name == s.currentName && strings.TrimSpace(description) == "" {
		description = s.currentDescription
	}

	return profileListDisplayName(name, description, name == s.currentName)
}
