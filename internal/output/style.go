package output

import (
	"io"
	"os"
)

const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiPrimary = "\x1b[36m"
	ansiMuted   = "\x1b[90m"
)

var forcedStyledOutput *bool

type styler struct {
	enabled bool
}

func newStyler(w io.Writer) styler {
	return styler{enabled: styledOutputEnabled(w)}
}

func styledOutputEnabled(w io.Writer) bool {
	if forcedStyledOutput != nil {
		return *forcedStyledOutput
	}

	return writerIsTTY(w) && stdinIsTTY()
}

func writerIsTTY(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	stat, err := file.Stat()
	if err != nil {
		return false
	}

	return stat.Mode()&os.ModeCharDevice != 0
}

func stdinIsTTY() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return stat.Mode()&os.ModeCharDevice != 0
}

func (s styler) heading(text string) string {
	if !s.enabled {
		return text
	}

	return ansiBold + ansiPrimary + text + ansiReset
}

func (s styler) current(text string) string {
	if !s.enabled {
		return text
	}

	return ansiBold + text + ansiReset
}

func (s styler) muted(text string) string {
	if !s.enabled {
		return text
	}

	return ansiMuted + text + ansiReset
}

func forceStyledOutputForTest(enabled bool) func() {
	prev := forcedStyledOutput
	forcedStyledOutput = &enabled

	return func() {
		forcedStyledOutput = prev
	}
}
