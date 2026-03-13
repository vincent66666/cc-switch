package output

import (
	"fmt"
	"io"
	"strings"

	"cc-switch/internal/profile"
)

func RenderStatus(w io.Writer, current string, currentProfile profile.Profile, names []string) int {
	_, _ = fmt.Fprintf(w, "current: %s\n", current)
	_, _ = fmt.Fprintf(w, "base_url: %s\n", currentProfile.Env[profile.EnvBaseURL])

	model := currentProfile.Env["ANTHROPIC_MODEL"]
	if model == "" {
		model = "-"
	}
	_, _ = fmt.Fprintf(w, "model: %s\n", model)

	if len(names) > 0 {
		_, _ = fmt.Fprintf(w, "available: %s\n", strings.Join(names, " "))
	}

	return 0
}

func RenderList(w io.Writer, names []string) int {
	for _, name := range names {
		_, _ = fmt.Fprintf(w, "%s\n", name)
	}
	return 0
}
