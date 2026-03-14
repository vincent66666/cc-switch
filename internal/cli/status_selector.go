package cli

import "strings"

const interactiveQuitHint = "按 q 或 Ctrl+C 退出"

type statusSelector struct {
	currentName string
	baseURL     string
	model       string
	names       []string
	index       int
}

func (s *statusSelector) moveUp() {
	if len(s.names) == 0 {
		return
	}

	s.index--
	if s.index < 0 {
		s.index = len(s.names) - 1
	}
}

func (s *statusSelector) moveDown() {
	if len(s.names) == 0 {
		return
	}

	s.index++
	if s.index >= len(s.names) {
		s.index = 0
	}
}

func (s statusSelector) selectedName() string {
	if len(s.names) == 0 {
		return ""
	}

	if s.index < 0 || s.index >= len(s.names) {
		return ""
	}

	return s.names[s.index]
}

func (s statusSelector) render() string {
	var out strings.Builder
	model := s.model
	if model == "" {
		model = "-"
	}

	out.WriteString("当前配置：" + s.currentName + "\n")
	out.WriteString("接口地址：" + s.baseURL + "\n")
	out.WriteString("模型：" + model + "\n")

	if len(s.names) == 0 {
		return out.String()
	}

	out.WriteString("可用配置：\n")
	for i, name := range s.names {
		prefix := "  "
		if i == s.index {
			prefix = "> "
		}
		out.WriteString(prefix + name + "\n")
	}

	out.WriteString(interactiveQuitHint + "\n")
	return out.String()
}
