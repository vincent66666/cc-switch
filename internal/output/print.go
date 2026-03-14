package output

import (
	"fmt"
	"io"
	"strings"

	"cc-switch/internal/profile"
)

func RenderStatus(w io.Writer, currentDisplay string, currentProfile profile.Profile, names []string) int {
	styler := newStyler(w)
	if styler.enabled && len(names) > 0 {
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

func RenderList(w io.Writer, names []string) int {
	styler := newStyler(w)
	if styler.enabled {
		renderStyledList(w, styler, names)
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

	writeStyledSection(w, styler.heading("当前配置"), styler.current(currentDisplay))
	writeStyledSection(w, styler.heading("接口地址"), currentProfile.Env[profile.EnvBaseURL])
	writeStyledSection(w, styler.heading("模型"), model)

	if len(names) == 0 {
		return
	}

	_, _ = fmt.Fprintf(w, "\n%s\n", styler.heading("可用配置"))
	for _, name := range names {
		_, _ = fmt.Fprintf(w, "  %s\n", name)
	}
}

func renderStyledList(w io.Writer, styler styler, names []string) {
	current, others := splitCurrent(names)
	if current != "" {
		writeStyledSection(w, styler.heading("当前配置"), styler.current(current))
	}

	if len(others) == 0 {
		return
	}

	if current != "" {
		_, _ = io.WriteString(w, "\n")
	}
	_, _ = fmt.Fprintf(w, "%s\n", styler.heading("其他配置"))
	for _, name := range others {
		_, _ = fmt.Fprintf(w, "  %s\n", name)
	}
}

func writeStyledSection(w io.Writer, heading, value string) {
	_, _ = fmt.Fprintf(w, "%s\n", heading)
	_, _ = fmt.Fprintf(w, "  %s\n", value)
}

func splitCurrent(names []string) (string, []string) {
	others := make([]string, 0, len(names))
	for i, name := range names {
		if strings.Contains(name, "（当前）") {
			others = append(others, names[:i]...)
			others = append(others, names[i+1:]...)
			return name, others
		}
	}

	return "", append(others, names...)
}
