package output

import (
	"bytes"
	"strings"
	"testing"

	"cc-switch/internal/profile"
)

func TestRenderStatusPlainOutputWhenWriterIsNotTTY(t *testing.T) {
	var buf bytes.Buffer

	code := RenderStatus(&buf, "demo - 正式环境", profile.Profile{
		Env: map[string]string{
			profile.EnvBaseURL: "https://example.com",
			"ANTHROPIC_MODEL":  "glm-5",
		},
	}, []string{"beta - 测试环境"})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	want := "当前配置：demo - 正式环境\n接口地址：https://example.com\n模型：glm-5\n可用配置：beta - 测试环境\n"
	if got := buf.String(); got != want {
		t.Fatalf("expected plain output %q, got %q", want, got)
	}

	if strings.Contains(buf.String(), "\x1b[") {
		t.Fatalf("expected plain output without ANSI escapes, got %q", buf.String())
	}
}

func TestRenderStatusUsesStyledSectionsWhenTTY(t *testing.T) {
	restore := forceStyledOutputForTest(true)
	defer restore()

	var buf bytes.Buffer
	RenderStatus(&buf, "demo - 正式环境", profile.Profile{
		Env: map[string]string{
			profile.EnvBaseURL: "https://example.com",
			"ANTHROPIC_MODEL":  "glm-5",
		},
	}, []string{"beta - 测试环境"})

	got := buf.String()
	plain := stripANSI(got)

	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected styled output to include ANSI escapes, got %q", got)
	}

	for _, fragment := range []string{
		"当前配置：demo - 正式环境\n",
		"接口地址：https://example.com\n",
		"模型：glm-5\n",
		"可用配置：\n",
	} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("expected styled output to contain %q after stripping ANSI, got %q", fragment, plain)
		}
	}
}

func TestRenderStatusUsesStyledOutputWithoutAlternativesWhenTTY(t *testing.T) {
	restore := forceStyledOutputForTest(true)
	defer restore()

	var buf bytes.Buffer
	RenderStatus(&buf, "demo - 正式环境", profile.Profile{
		Env: map[string]string{
			profile.EnvBaseURL: "https://example.com",
			"ANTHROPIC_MODEL":  "glm-5",
		},
	}, nil)

	got := buf.String()
	plain := stripANSI(got)

	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected styled output to include ANSI escapes, got %q", got)
	}
	for _, fragment := range []string{
		"当前配置：demo - 正式环境\n",
		"接口地址：https://example.com\n",
		"模型：glm-5\n",
	} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("expected single-profile styled output to contain %q after stripping ANSI, got %q", fragment, plain)
		}
	}
	if strings.Contains(plain, "可用配置：") {
		t.Fatalf("expected no alternatives section for single-profile styled output, got %q", plain)
	}
}

func TestRenderListPlainOutputWhenWriterIsNotTTY(t *testing.T) {
	var buf bytes.Buffer

	code := RenderList(&buf, "demo - 正式环境", []string{"beta - 测试环境", "demo - 正式环境"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	want := "beta - 测试环境\ndemo - 正式环境\n"
	if got := buf.String(); got != want {
		t.Fatalf("expected plain output %q, got %q", want, got)
	}
}

func TestRenderListUsesGroupedSectionsWhenTTY(t *testing.T) {
	restore := forceStyledOutputForTest(true)
	defer restore()

	var buf bytes.Buffer
	RenderList(&buf, "demo - 正式环境", []string{"beta - 包含（当前）字样", "demo - 正式环境", "prod"})

	got := buf.String()
	plain := stripANSI(got)

	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected styled output to include ANSI escapes, got %q", got)
	}

	for _, fragment := range []string{
		"当前配置\n",
		"  demo - 正式环境\n",
		"其他配置\n",
		"  beta - 包含（当前）字样\n",
		"  prod\n",
	} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("expected grouped output to contain %q after stripping ANSI, got %q", fragment, plain)
		}
	}
}

func stripANSI(input string) string {
	var out strings.Builder
	skip := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		if skip {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				skip = false
			}
			continue
		}

		if ch == 0x1b {
			skip = true
			continue
		}

		out.WriteByte(ch)
	}

	return out.String()
}
