package output

import (
	"fmt"
	"io"
	"strings"

	"cc-switch/internal/profile"
)

func RenderStatus(w io.Writer, currentDisplay string, currentProfile profile.Profile, names []string) int {
	styler := newStyler(w)
	if styler.enabled {
		renderStyledStatus(w, styler, currentDisplay, currentProfile, names)
		return 0
	}

	_, _ = fmt.Fprintf(w, "当前配置：%s\n", currentDisplay)
	_, _ = fmt.Fprintf(w, "接口地址：%s\n", currentProfile.Env[profile.EnvBaseURL])

	model := currentProfile.Env["ANTHROPIC_MODEL"]
	if model == "" {
		model = "-"
	}
	_, _ = fmt.Fprintf(w, "模型：%s\n", model)

	if len(names) > 0 {
		_, _ = fmt.Fprintf(w, "可用配置：%s\n", strings.Join(names, " "))
	}

	return 0
}

func RenderList(w io.Writer, currentDisplay string, names []string) int {
	styler := newStyler(w)
	if styler.enabled {
		renderStyledList(w, styler, currentDisplay, names)
		return 0
	}

	for _, name := range names {
		_, _ = fmt.Fprintf(w, "%s\n", name)
	}
	return 0
}

func renderStyledStatus(w io.Writer, styler styler, currentDisplay string, currentProfile profile.Profile, names []string) {
	model := currentProfile.Env["ANTHROPIC_MODEL"]
	if model == "" {
		model = "-"
	}

	writeStyledLine(w, styler.current("当前配置："+currentDisplay))
	writeStyledLine(w, styler.heading("接口地址："+currentProfile.Env[profile.EnvBaseURL]))
	writeStyledLine(w, styler.heading("模型："+model))

	if len(names) == 0 {
		return
	}

	_, _ = fmt.Fprintf(w, "\n%s\n", styler.heading("可用配置："))
	for _, name := range names {
		_, _ = fmt.Fprintf(w, "  %s\n", name)
	}
}

func renderStyledList(w io.Writer, styler styler, currentDisplay string, names []string) {
	current, others := splitCurrent(names, currentDisplay)
	if current != "" {
		writeStyledSection(w, styler.heading("当前配置"), styler.current(current))
	}

	if len(others) == 0 {
		return
	}

	heading := "其他配置"
	if current == "" {
		heading = "可用配置"
	}
	if current != "" {
		_, _ = io.WriteString(w, "\n")
	}
	_, _ = fmt.Fprintf(w, "%s\n", styler.heading(heading))
	for _, name := range others {
		_, _ = fmt.Fprintf(w, "  %s\n", name)
	}
}

func writeStyledLine(w io.Writer, line string) {
	_, _ = fmt.Fprintf(w, "%s\n", line)
}

func writeStyledSection(w io.Writer, heading, value string) {
	_, _ = fmt.Fprintf(w, "%s\n", heading)
	_, _ = fmt.Fprintf(w, "  %s\n", value)
}

func splitCurrent(names []string, currentDisplay string) (string, []string) {
	if currentDisplay == "" {
		return "", append([]string(nil), names...)
	}

	others := make([]string, 0, len(names))
	current := ""
	for _, name := range names {
		if name == currentDisplay {
			current = name
			continue
		}
		others = append(others, name)
	}

	return current, others
}
