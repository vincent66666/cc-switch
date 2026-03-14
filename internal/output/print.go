package output

import (
	"fmt"
	"io"
	"strings"

	"cc-switch/internal/profile"
)

func RenderStatus(w io.Writer, current string, currentProfile profile.Profile, names []string) int {
	_, _ = fmt.Fprintf(w, "当前配置：%s\n", current)
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
	for _, name := range names {
		_, _ = fmt.Fprintf(w, "%s\n", name)
	}
	return 0
}
