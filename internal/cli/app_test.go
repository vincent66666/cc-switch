package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cc-switch/internal/profile"
)

func TestMain(m *testing.M) {
	promptInteractive = func() bool { return false }
	os.Exit(m.Run())
}

func TestTTYHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TTY_HELPER") != "1" {
		return
	}

	promptReader = os.Stdin
	promptWriter = os.Stdout
	promptInteractive = func() bool {
		stat, err := os.Stdin.Stat()
		if err != nil {
			return false
		}
		return stat.Mode()&os.ModeCharDevice != 0
	}

	args := helperArgs(os.Args)
	os.Exit(Run(args, os.Stdout, os.Stderr))
}

func TestRun_NoArgsShowsStatus(t *testing.T) {
	t.Setenv("CC_SWITCH_PROFILES_PATH", filepath.Join(t.TempDir(), "profiles.json"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(nil, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if got := stdout.String(); got != "当前配置：未知\n" {
		t.Fatalf("expected status output, got %q", got)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("expected empty stderr, got %q", got)
	}
}

func TestRun_UseUpdatesSettingsEnvAndCurrentProfile(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "beta",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
					"ANTHROPIC_MODEL":    "glm-5",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})
	settingsPath := writeSettingsFixture(t, `{
  "model": "opus",
  "enabledPlugins": {
    "demo": true
  },
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "old-token"
  }
}
`)

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"use", "demo"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := stdout.String(); got != "已切换到配置：demo\n" {
		t.Fatalf("expected switch success output, got %q", got)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after switch: %v", err)
	}

	if savedProfiles.Current != "demo" {
		t.Fatalf("expected current profile demo, got %q", savedProfiles.Current)
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings after switch: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("unmarshal settings after switch: %v", err)
	}

	if got["model"] != "opus" {
		t.Fatalf("expected model to remain unchanged, got %#v", got["model"])
	}

	env, ok := got["env"].(map[string]any)
	if !ok {
		t.Fatalf("expected env object, got %#v", got["env"])
	}

	if env[profile.EnvAuthToken] != "token-demo" {
		t.Fatalf("expected switched token, got %#v", env[profile.EnvAuthToken])
	}

	if env[profile.EnvBaseURL] != "https://demo.example.com" {
		t.Fatalf("expected switched base url, got %#v", env[profile.EnvBaseURL])
	}
}

func TestRun_UseDoesNotAdvanceCurrentWhenSettingsWriteFails(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "beta",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})
	settingsPath := writeSettingsFixture(t, `{"env":`)

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"use", "demo"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected switch to fail when settings json is invalid")
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after failed switch: %v", err)
	}

	if savedProfiles.Current != "beta" {
		t.Fatalf("expected current profile to remain beta, got %q", savedProfiles.Current)
	}
}

func TestRun_AddPersistsProfile(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version:  1,
		Profiles: map[string]profile.Profile{},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{
		"add",
		"demo",
		"--description", "Demo profile",
		"--token", "token-demo",
		"--base-url", "https://demo.example.com",
		"--model", "glm-5",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected add to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := stdout.String(); got != "已添加配置：demo\n" {
		t.Fatalf("expected add success output, got %q", got)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after add: %v", err)
	}

	saved := savedProfiles.Profiles["demo"]
	if saved.Description != "Demo profile" {
		t.Fatalf("expected description to persist, got %q", saved.Description)
	}

	if saved.Env[profile.EnvAuthToken] != "token-demo" {
		t.Fatalf("expected token to persist, got %q", saved.Env[profile.EnvAuthToken])
	}

	if saved.Env["ANTHROPIC_MODEL"] != "glm-5" {
		t.Fatalf("expected model to persist, got %q", saved.Env["ANTHROPIC_MODEL"])
	}
}

func TestRun_EditUpdatesExistingProfile(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Description: "Old description",
				Env: map[string]string{
					profile.EnvAuthToken: "token-old",
					profile.EnvBaseURL:   "https://old.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{
		"edit",
		"demo",
		"--description", "New description",
		"--base-url", "https://new.example.com",
		"--model", "glm-5",
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected edit to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := stdout.String(); got != "已更新配置：demo\n" {
		t.Fatalf("expected edit success output, got %q", got)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after edit: %v", err)
	}

	saved := savedProfiles.Profiles["demo"]
	if saved.Description != "New description" {
		t.Fatalf("expected description to update, got %q", saved.Description)
	}

	if saved.Env[profile.EnvAuthToken] != "token-old" {
		t.Fatalf("expected missing token flag to preserve previous value, got %q", saved.Env[profile.EnvAuthToken])
	}

	if saved.Env[profile.EnvBaseURL] != "https://new.example.com" {
		t.Fatalf("expected base url to update, got %q", saved.Env[profile.EnvBaseURL])
	}

	if saved.Env["ANTHROPIC_MODEL"] != "glm-5" {
		t.Fatalf("expected model to update, got %q", saved.Env["ANTHROPIC_MODEL"])
	}
}

func TestRun_AddRejectsMissingRequiredFields(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version:  1,
		Profiles: map[string]profile.Profile{},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{
		"add",
		"demo",
		"--base-url", "https://demo.example.com",
	}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected add to fail when token is missing")
	}
}

func TestRun_AddRejectsMissingNameNonInteractive(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version:  1,
		Profiles: map[string]profile.Profile{},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{
		"add",
		"--token", "token-demo",
		"--base-url", "https://demo.example.com",
	}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected add to fail when name is missing in non-interactive mode")
	}

	if got := stderr.String(); got != "必须提供配置名称\n" {
		t.Fatalf("expected missing name error, got %q", got)
	}
}

func TestRun_AddRejectsMissingBaseURLNonInteractive(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version:  1,
		Profiles: map[string]profile.Profile{},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{
		"add",
		"demo",
		"--token", "token-demo",
	}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected add to fail when base url is missing in non-interactive mode")
	}

	if got := stderr.String(); !strings.Contains(got, "配置 \"demo\" 缺少必填字段：ANTHROPIC_BASE_URL") {
		t.Fatalf("expected missing base url error, got %q", got)
	}
}

func TestRun_AddRejectsDuplicateName(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{
		"add",
		"demo",
		"--token", "token-new",
		"--base-url", "https://new.example.com",
	}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected add to fail when profile name already exists")
	}

	if got := stderr.String(); got != "配置 \"demo\" 已存在\n" {
		t.Fatalf("expected duplicate profile error, got %q", got)
	}
}

func TestRun_AddInteractivePromptsForAllFields(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version:  1,
		Profiles: map[string]profile.Profile{},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	promptOutput := withPromptSession(t, strings.Join([]string{
		"demo",
		"Demo profile",
		"token-demo",
		"https://demo.example.com",
		"glm-5",
		"glm-5",
		"kimi-k2.5",
		"MiniMax-M2.5",
		"",
	}, "\n"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"add"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected interactive add to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interactive add: %v", err)
	}

	saved := savedProfiles.Profiles["demo"]
	if saved.Description != "Demo profile" {
		t.Fatalf("expected description to be collected, got %q", saved.Description)
	}

	if saved.Env[profile.EnvAuthToken] != "token-demo" {
		t.Fatalf("expected token to be collected, got %q", saved.Env[profile.EnvAuthToken])
	}

	if saved.Env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != "MiniMax-M2.5" {
		t.Fatalf("expected optional models to be collected, got %q", saved.Env["ANTHROPIC_DEFAULT_HAIKU_MODEL"])
	}

	for _, fragment := range []string{
		"名称：",
		"描述（可选）：",
		"ANTHROPIC_AUTH_TOKEN：",
		"ANTHROPIC_BASE_URL：",
		"ANTHROPIC_MODEL（可选）：",
	} {
		if !strings.Contains(promptOutput.String(), fragment) {
			t.Fatalf("expected prompt output to contain %q, got %q", fragment, promptOutput.String())
		}
	}
}

func TestRun_AddInteractiveRejectsDuplicateNameBeforeFurtherPrompts(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	promptOutput := withPromptSession(t, "demo\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"add"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected interactive add to fail for duplicate name")
	}

	if got := stderr.String(); got != "配置 \"demo\" 已存在\n" {
		t.Fatalf("expected duplicate name error, got %q", got)
	}

	if strings.Contains(promptOutput.String(), "描述（可选）：") {
		t.Fatalf("expected duplicate name to stop before prompting other fields, got %q", promptOutput.String())
	}
}

func TestRun_AddInteractiveInterruptedInputFails(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version:  1,
		Profiles: map[string]profile.Profile{},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	withPromptSession(t, "demo\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"add"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected interactive add to fail when input is interrupted")
	}

	if got := stderr.String(); got != "输入已结束\n" {
		t.Fatalf("expected EOF error, got %q", got)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interrupted add: %v", err)
	}

	if len(savedProfiles.Profiles) != 0 {
		t.Fatalf("expected no profile to be written on interrupted input, got %#v", savedProfiles.Profiles)
	}
}

func TestRun_AddRealTTYCtrlDAbortsWithoutWrite(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version:  1,
		Profiles: map[string]profile.Profile{},
	})

	exitCode, output := runWithTTY(t, scriptPath, []string{"add"}, "demo\n\x04", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})

	if exitCode == 0 {
		t.Fatalf("expected Ctrl+D in real TTY add to fail, output=%q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after real TTY Ctrl+D: %v", err)
	}

	if len(savedProfiles.Profiles) != 0 {
		t.Fatalf("expected no profile to be written after Ctrl+D, got %#v", savedProfiles.Profiles)
	}
}

func TestRun_AddRealTTYCtrlCAbortsWithoutWrite(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version:  1,
		Profiles: map[string]profile.Profile{},
	})

	exitCode, output := runWithTTY(t, scriptPath, []string{"add"}, "\x03", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})

	if exitCode == 0 {
		t.Fatalf("expected Ctrl+C in real TTY add to fail, output=%q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after real TTY Ctrl+C: %v", err)
	}

	if len(savedProfiles.Profiles) != 0 {
		t.Fatalf("expected no profile to be written after Ctrl+C, got %#v", savedProfiles.Profiles)
	}

	if strings.TrimSpace(output) == "" {
		t.Fatal("expected Ctrl+C output to contain an interruption trace")
	}

	lowerOutput := strings.ToLower(output)
	if !strings.Contains(output, "^C") && !strings.Contains(lowerOutput, "interrupt") {
		t.Fatalf("expected Ctrl+C output to show interruption, got %q", output)
	}
}

func TestRun_AddRealTTYBlankInputFails(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version:  1,
		Profiles: map[string]profile.Profile{},
	})

	exitCode, output := runWithTTY(t, scriptPath, []string{"add"}, "\n", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})

	if exitCode == 0 {
		t.Fatalf("expected blank input in real TTY add to fail, output=%q", output)
	}

	if !strings.Contains(output, "缺少必填字段：名称") {
		t.Fatalf("expected missing name message in real TTY add, got %q", output)
	}
}

func TestRun_EditInteractivePromptsAndKeepsExistingValuesOnBlank(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Description: "Old description",
				Env: map[string]string{
					profile.EnvAuthToken:             "token-old",
					profile.EnvBaseURL:               "https://old.example.com",
					"ANTHROPIC_MODEL":                "glm-old",
					"ANTHROPIC_DEFAULT_OPUS_MODEL":   "opus-old",
					"ANTHROPIC_DEFAULT_SONNET_MODEL": "sonnet-old",
					"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "haiku-old",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	promptOutput := withPromptSession(t, strings.Join([]string{
		"",
		"",
		"https://new.example.com",
		"",
		"",
		"",
		"",
		"",
	}, "\n"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"edit", "demo"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected interactive edit to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interactive edit: %v", err)
	}

	saved := savedProfiles.Profiles["demo"]
	if saved.Description != "Old description" {
		t.Fatalf("expected blank description input to preserve old value, got %q", saved.Description)
	}

	if saved.Env[profile.EnvAuthToken] != "token-old" {
		t.Fatalf("expected blank token input to preserve old value, got %q", saved.Env[profile.EnvAuthToken])
	}

	if saved.Env[profile.EnvBaseURL] != "https://new.example.com" {
		t.Fatalf("expected base url to update, got %q", saved.Env[profile.EnvBaseURL])
	}

	if saved.Env["ANTHROPIC_MODEL"] != "glm-old" {
		t.Fatalf("expected blank model input to preserve old value, got %q", saved.Env["ANTHROPIC_MODEL"])
	}

	if strings.Contains(promptOutput.String(), "token-old") {
		t.Fatalf("expected token prompt to be masked, got %q", promptOutput.String())
	}

	if !strings.Contains(promptOutput.String(), "ANTHROPIC_AUTH_TOKEN [to") {
		t.Fatalf("expected masked token prompt, got %q", promptOutput.String())
	}
}

func TestRun_EditInteractiveBlankOptionalFieldKeepsMissingKey(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Description: "Old description",
				Env: map[string]string{
					profile.EnvAuthToken: "token-old",
					profile.EnvBaseURL:   "https://old.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	promptOutput := withPromptSession(t, strings.Join([]string{
		"",
		"",
		"",
		"",
		"",
		"",
		"",
	}, "\n")+"\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"edit", "demo"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected edit with blank optional fields to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after edit: %v", err)
	}

	if _, exists := savedProfiles.Profiles["demo"].Env["ANTHROPIC_MODEL"]; exists {
		t.Fatalf("expected blank optional model to keep key absent, got %#v", savedProfiles.Profiles["demo"].Env)
	}

	if !strings.Contains(promptOutput.String(), "ANTHROPIC_MODEL（直接回车保留当前值）：") {
		t.Fatalf("expected optional model prompt, got %q", promptOutput.String())
	}
}

func TestRun_EditInteractiveInterruptedInputFailsWithoutMutation(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Description: "Old description",
				Env: map[string]string{
					profile.EnvAuthToken: "token-old",
					profile.EnvBaseURL:   "https://old.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	withPromptSession(t, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"edit", "demo"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected interactive edit to fail when input is interrupted")
	}

	if got := stderr.String(); got != "输入已结束\n" {
		t.Fatalf("expected EOF error, got %q", got)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interrupted edit: %v", err)
	}

	if savedProfiles.Profiles["demo"].Description != "Old description" {
		t.Fatalf("expected interrupted edit to leave profile unchanged, got %#v", savedProfiles.Profiles["demo"])
	}
}

func TestRun_EditRealTTYMultipleEntersKeepValues(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Description: "Old description",
				Env: map[string]string{
					profile.EnvAuthToken:             "token-old",
					profile.EnvBaseURL:               "https://old.example.com",
					"ANTHROPIC_MODEL":                "glm-old",
					"ANTHROPIC_DEFAULT_OPUS_MODEL":   "opus-old",
					"ANTHROPIC_DEFAULT_SONNET_MODEL": "sonnet-old",
					"ANTHROPIC_DEFAULT_HAIKU_MODEL":  "haiku-old",
				},
			},
		},
	})

	exitCode, output := runWithTTY(t, scriptPath, []string{"edit", "demo"}, strings.Repeat("\n", 7), map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})

	if exitCode != 0 {
		t.Fatalf("expected multiple enters in real TTY edit to succeed, output=%q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after real TTY edit: %v", err)
	}

	saved := savedProfiles.Profiles["demo"]
	if saved.Description != "Old description" {
		t.Fatalf("expected description to remain unchanged, got %q", saved.Description)
	}
	if saved.Env[profile.EnvAuthToken] != "token-old" || saved.Env[profile.EnvBaseURL] != "https://old.example.com" {
		t.Fatalf("expected required fields to remain unchanged, got %#v", saved.Env)
	}
}

func TestRun_EditInteractiveSkipsPromptForExplicitFields(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Description: "Old description",
				Env: map[string]string{
					profile.EnvAuthToken: "token-old",
					profile.EnvBaseURL:   "https://old.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	promptOutput := withPromptSession(t, strings.Join([]string{
		"",
		"",
		"",
		"",
		"",
		"",
	}, "\n")+"\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"edit", "demo", "--token", "token-new"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected edit with explicit token to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after edit: %v", err)
	}

	if savedProfiles.Profiles["demo"].Env[profile.EnvAuthToken] != "token-new" {
		t.Fatalf("expected explicit token to win, got %q", savedProfiles.Profiles["demo"].Env[profile.EnvAuthToken])
	}

	if strings.Contains(promptOutput.String(), "ANTHROPIC_AUTH_TOKEN [") {
		t.Fatalf("expected token prompt to be skipped when explicit token is provided, got %q", promptOutput.String())
	}
}

func TestRun_EditInteractiveMasksShortToken(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Description: "Old description",
				Env: map[string]string{
					profile.EnvAuthToken: "abcd",
					profile.EnvBaseURL:   "https://old.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	promptOutput := withPromptSession(t, strings.Join([]string{
		"",
		"",
		"",
		"",
		"",
		"",
		"",
	}, "\n")+"\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"edit", "demo"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected edit with short token to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	if strings.Contains(promptOutput.String(), "abcd") {
		t.Fatalf("expected short token to stay masked, got %q", promptOutput.String())
	}

	if !strings.Contains(promptOutput.String(), "ANTHROPIC_AUTH_TOKEN [****]（直接回车保留当前值）：") {
		t.Fatalf("expected short token mask prompt, got %q", promptOutput.String())
	}
}

func TestRun_UseDoesNotAdvanceCurrentWhenBackupDirUnwritable(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "beta",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})
	settingsPath := writeSettingsFixture(t, `{"env":{"ANTHROPIC_AUTH_TOKEN":"old-token"}}`)

	homeFile := filepath.Join(t.TempDir(), "home-file")
	if err := os.WriteFile(homeFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write fake home file: %v", err)
	}

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)
	t.Setenv("HOME", homeFile)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"use", "demo"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected use to fail when backup dir is not writable")
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after failed use: %v", err)
	}

	if savedProfiles.Current != "beta" {
		t.Fatalf("expected current profile to remain beta, got %q", savedProfiles.Current)
	}
}

func TestRun_CustomPathsSupportAddUseAndCurrentFlow(t *testing.T) {
	root := t.TempDir()
	profilesPath := filepath.Join(root, "custom", "profiles.json")
	settingsPath := filepath.Join(root, "custom", "settings.json")

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)

	var addStdout bytes.Buffer
	var addStderr bytes.Buffer

	addExitCode := Run([]string{
		"add",
		"demo",
		"--description", "Demo profile",
		"--token", "token-demo",
		"--base-url", "https://demo.example.com",
	}, &addStdout, &addStderr)
	if addExitCode != 0 {
		t.Fatalf("expected add on custom paths to succeed, got %d, stderr=%q", addExitCode, addStderr.String())
	}

	var useStdout bytes.Buffer
	var useStderr bytes.Buffer

	useExitCode := Run([]string{"use", "demo"}, &useStdout, &useStderr)
	if useExitCode != 0 {
		t.Fatalf("expected use on custom paths to succeed, got %d, stderr=%q", useExitCode, useStderr.String())
	}

	var currentStdout bytes.Buffer
	var currentStderr bytes.Buffer

	currentExitCode := Run([]string{"current"}, &currentStdout, &currentStderr)
	if currentExitCode != 0 {
		t.Fatalf("expected current on custom paths to succeed, got %d, stderr=%q", currentExitCode, currentStderr.String())
	}

	if got := currentStdout.String(); got != "demo\n" {
		t.Fatalf("expected current output for custom paths, got %q", got)
	}

	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings from custom path: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("unmarshal settings from custom path: %v", err)
	}

	env, ok := got["env"].(map[string]any)
	if !ok {
		t.Fatalf("expected env object in custom settings, got %#v", got["env"])
	}

	if env[profile.EnvAuthToken] != "token-demo" {
		t.Fatalf("expected custom settings token, got %#v", env[profile.EnvAuthToken])
	}
}

func TestRun_UseDoesNotAdvanceCurrentWhenCustomSettingsPathUnwritable(t *testing.T) {
	root := t.TempDir()
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "beta",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})
	settingsDirAsFile := filepath.Join(root, "settings-parent")
	if err := os.WriteFile(settingsDirAsFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write fake settings parent file: %v", err)
	}
	settingsPath := filepath.Join(settingsDirAsFile, "settings.json")

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)
	t.Setenv("CC_SWITCH_SETTINGS_PATH", settingsPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"use", "demo"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected use to fail when custom settings path is not writable")
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after failed use: %v", err)
	}

	if savedProfiles.Current != "beta" {
		t.Fatalf("expected current profile to remain beta, got %q", savedProfiles.Current)
	}
}

func TestRun_RemoveRejectsCurrentProfile(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"remove", "demo"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected remove to fail for current profile")
	}

	if got := stderr.String(); got != "不能删除当前正在使用的配置\n" {
		t.Fatalf("expected active profile remove error, got %q", got)
	}
}

func TestRun_RemoveDeletesNonCurrentProfile(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"remove", "beta"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected remove to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := stdout.String(); got != "已删除配置：beta\n" {
		t.Fatalf("expected remove success output, got %q", got)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after remove: %v", err)
	}

	if _, exists := savedProfiles.Profiles["beta"]; exists {
		t.Fatal("expected beta profile to be removed")
	}
}

func TestRun_RenameMovesProfileAndCurrentPointer(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"demo": {
				Description: "Demo",
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"rename", "demo", "prod"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected rename to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := stdout.String(); got != "已将配置 demo 重命名为 prod\n" {
		t.Fatalf("expected rename success output, got %q", got)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after rename: %v", err)
	}

	if savedProfiles.Current != "prod" {
		t.Fatalf("expected current profile to move to prod, got %q", savedProfiles.Current)
	}

	if _, exists := savedProfiles.Profiles["demo"]; exists {
		t.Fatal("expected old profile name to be removed")
	}

	if savedProfiles.Profiles["prod"].Description != "Demo" {
		t.Fatalf("expected profile data to survive rename, got %#v", savedProfiles.Profiles["prod"])
	}
}

func TestRun_ImportCommandIsUnknown(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"import", "--from", "/tmp/legacy"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("expected import to fail, got %d", exitCode)
	}

	if got := stdout.String(); got != "" {
		t.Fatalf("expected empty stdout, got %q", got)
	}

	if got := stderr.String(); got != "未知命令：import\n" {
		t.Fatalf("expected unknown command error, got %q", got)
	}
}

func TestRun_StatusInteractiveArrowSelectionSwitchesProfile(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
					"ANTHROPIC_MODEL":    "glm-5",
				},
			},
			"prod": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-prod",
					profile.EnvBaseURL:   "https://prod.example.com",
				},
			},
		},
	})
	settingsPath := writeSettingsFixture(t, `{"env":{"ANTHROPIC_AUTH_TOKEN":"old-token"}}`)

	exitCode, output := runWithTTYBestEffort(t, scriptPath, nil, "\x1b[B\r", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
		"CC_SWITCH_SETTINGS_PATH": settingsPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive status switch to succeed, output=%q", output)
	}

	if !strings.Contains(output, "可用配置：") || !strings.Contains(output, "> prod") {
		t.Fatalf("expected interactive selector output, got %q", output)
	}
	if !strings.Contains(output, "已切换到配置：prod") {
		t.Fatalf("expected switch success output, got %q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interactive status switch: %v", err)
	}
	if savedProfiles.Current != "prod" {
		t.Fatalf("expected current profile to switch to prod, got %q", savedProfiles.Current)
	}
}

func TestRun_StatusInteractiveQuitLeavesCurrentUnchanged(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, nil, "q", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive status quit to succeed, output=%q", output)
	}

	if !strings.Contains(output, "可用配置：") {
		t.Fatalf("expected interactive selector output, got %q", output)
	}
	if strings.Contains(output, "已切换到配置：") {
		t.Fatalf("expected quit to avoid switching, got %q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interactive quit: %v", err)
	}
	if savedProfiles.Current != "demo" {
		t.Fatalf("expected current profile to remain demo, got %q", savedProfiles.Current)
	}
}

func TestRun_StatusInteractiveUsesAlternateScreen(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, nil, "q", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive status alt-screen flow to succeed, output=%q", output)
	}

	if !strings.Contains(output, "\x1b[?1049h") || !strings.Contains(output, "\x1b[?1049l") {
		t.Fatalf("expected interactive status to enter and leave alternate screen, got %q", output)
	}
}

func TestRun_StatusInteractiveAlternateScreenWrapsMultipleMoves(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"prod": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-prod",
					profile.EnvBaseURL:   "https://prod.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, nil, "\x1b[B\x1b[Aq", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected multi-move status flow to succeed, output=%q", output)
	}

	if got := strings.Count(output, enterAlternateScreenMode); got != 1 {
		t.Fatalf("expected one alternate-screen enter sequence, got %d in %q", got, output)
	}
	if got := strings.Count(output, exitAlternateScreenMode); got != 1 {
		t.Fatalf("expected one alternate-screen exit sequence, got %d in %q", got, output)
	}
}

func TestRun_StatusInteractiveCtrlCLeavesCurrentUnchanged(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, nil, "\x03", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive status Ctrl+C to exit cleanly, output=%q", output)
	}

	if !strings.Contains(output, "可用配置：") {
		t.Fatalf("expected interactive selector output, got %q", output)
	}
	if strings.Contains(output, "^C") || strings.Contains(strings.ToLower(output), "interrupt") {
		t.Fatalf("expected Ctrl+C to be handled as a clean exit, got %q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interactive Ctrl+C: %v", err)
	}
	if savedProfiles.Current != "demo" {
		t.Fatalf("expected current profile to remain demo, got %q", savedProfiles.Current)
	}
}

func TestRun_StatusInteractiveWithoutAlternativesPrintsStatusOnly(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
					"ANTHROPIC_MODEL":    "glm-5",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, nil, "", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected status-only TTY run to succeed, output=%q", output)
	}

	if !strings.Contains(output, "当前配置：demo") || !strings.Contains(output, "模型：glm-5") {
		t.Fatalf("expected status output, got %q", output)
	}
	if strings.Contains(output, "可用配置：") {
		t.Fatalf("expected no selector list when there are no alternatives, got %q", output)
	}
}

func TestRun_StatusInteractiveEOFClosesAlternateScreen(t *testing.T) {
	readerFile, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatalf("create temp input file: %v", err)
	}
	t.Cleanup(func() {
		_ = readerFile.Close()
	})

	oldPromptReader := promptReader
	oldStartInteractiveSession := startInteractiveSession
	t.Cleanup(func() {
		promptReader = oldPromptReader
		startInteractiveSession = oldStartInteractiveSession
	})
	promptReader = readerFile
	startInteractiveSession = func(_ *os.File, stdout io.Writer) (func(), error) {
		_, _ = io.WriteString(stdout, enterAlternateScreenMode)
		active := true
		return func() {
			if !active {
				return
			}
			active = false
			_, _ = io.WriteString(stdout, exitAlternateScreenMode)
		}, nil
	}

	selector := statusSelector{
		currentName: "demo",
		baseURL:     "https://demo.example.com",
		model:       "glm-5",
		names:       []string{"beta"},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runInteractiveStatus(Paths{}, selector, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected interactive status EOF flow to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := strings.Count(stdout.String(), enterAlternateScreenMode); got != 1 {
		t.Fatalf("expected one alternate-screen enter on EOF exit, got %d in %q", got, stdout.String())
	}
	if got := strings.Count(stdout.String(), exitAlternateScreenMode); got != 1 {
		t.Fatalf("expected one alternate-screen exit on EOF exit, got %d in %q", got, stdout.String())
	}
}

func TestRun_StatusFallsBackToPlainTextWhenStdoutIsNotTTY(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
					"ANTHROPIC_MODEL":    "glm-5",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)
	withPromptSession(t, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(nil, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected status fallback to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	want := "当前配置：demo\n接口地址：https://demo.example.com\n模型：glm-5\n可用配置：beta\n"
	if got := stdout.String(); got != want {
		t.Fatalf("expected plain-text status output %q, got %q", want, got)
	}
}

func TestRun_StatusFallsBackToPlainTextWhenRawTerminalUnavailable(t *testing.T) {
	readerFile, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatalf("create temp input file: %v", err)
	}
	t.Cleanup(func() {
		_ = readerFile.Close()
	})

	oldPromptReader := promptReader
	t.Cleanup(func() {
		promptReader = oldPromptReader
	})
	promptReader = readerFile

	selector := statusSelector{
		currentName: "demo",
		baseURL:     "https://demo.example.com",
		model:       "glm-5",
		names:       []string{"beta"},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := runInteractiveStatus(Paths{}, selector, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected raw-terminal fallback to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	want := selector.render()
	if got := stdout.String(); got != want {
		t.Fatalf("expected plain selector render on raw-terminal failure, got %q", got)
	}
	if strings.Contains(stdout.String(), enterAlternateScreenMode) || strings.Contains(stdout.String(), exitAlternateScreenMode) {
		t.Fatalf("expected no alternate-screen sequences on raw-terminal fallback, got %q", stdout.String())
	}
}

func TestRun_StatusFallsBackToPlainTextWhenStdoutIsTTYAndStdinIsFile(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
					"ANTHROPIC_MODEL":    "glm-5",
				},
			},
		},
	})

	exitCode, output := runWithTTYStdoutAndFileStdin(t, scriptPath, nil, "", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected tty-stdout/file-stdin status fallback to succeed, output=%q", output)
	}

	want := "当前配置：demo\r\n接口地址：https://demo.example.com\r\n模型：glm-5\r\n可用配置：beta\r\n"
	if output != want {
		t.Fatalf("expected plain-text tty-stdout/file-stdin status output %q, got %q", want, output)
	}
	if strings.Contains(output, clearScreenSequence) || strings.Contains(output, enterAlternateScreenMode) {
		t.Fatalf("expected no interactive control sequences in tty-stdout/file-stdin status output, got %q", output)
	}
}

func TestStatusSelectorRenderIncludesQuitHint(t *testing.T) {
	selector := statusSelector{
		currentName: "demo",
		baseURL:     "https://demo.example.com",
		model:       "glm-5",
		names:       []string{"beta"},
	}

	rendered := selector.render()

	if !strings.Contains(rendered, "按 q 或 Ctrl+C 退出") {
		t.Fatalf("expected status selector to include quit hint, got %q", rendered)
	}
}

func TestRun_CurrentPrintsCurrentProfile(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token",
					profile.EnvBaseURL:   "https://example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"current"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if got := stdout.String(); got != "demo\n" {
		t.Fatalf("expected current profile output, got %q", got)
	}
}

func TestRun_ListPrintsProfiles(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-b",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-a",
					profile.EnvBaseURL:   "https://example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"list"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if got := stdout.String(); got != "beta\ndemo\n" {
		t.Fatalf("expected sorted profile list, got %q", got)
	}
}

func TestRun_ListFallsBackToPlainTextWhenStdoutIsNotTTY(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)
	withPromptSession(t, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"list"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected list fallback to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := stdout.String(); got != "beta\ndemo\n" {
		t.Fatalf("expected plain-text list output, got %q", got)
	}
}

func TestRun_ListFallsBackToPlainTextWhenStdinIsNotTTY(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	oldPromptInteractive := promptInteractive
	t.Cleanup(func() {
		promptInteractive = oldPromptInteractive
	})
	promptInteractive = func() bool { return false }

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"list"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected stdin non-tty fallback to succeed, got %d, stderr=%q", exitCode, stderr.String())
	}

	if got := stdout.String(); got != "beta\ndemo\n" {
		t.Fatalf("expected plain-text list output when stdin is not tty, got %q", got)
	}
	if strings.Contains(stdout.String(), clearScreenSequence) || strings.Contains(stdout.String(), enterAlternateScreenMode) {
		t.Fatalf("expected no interactive control sequences when stdin is not tty, got %q", stdout.String())
	}
}

func TestRun_ListFallsBackToPlainTextWhenStdoutIsTTYAndStdinIsFile(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYStdoutAndFileStdin(t, scriptPath, []string{"list"}, "", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected tty-stdout/file-stdin list fallback to succeed, output=%q", output)
	}

	want := "beta\r\ndemo\r\n"
	if output != want {
		t.Fatalf("expected plain-text tty-stdout/file-stdin list output %q, got %q", want, output)
	}
	if strings.Contains(output, clearScreenSequence) || strings.Contains(output, enterAlternateScreenMode) {
		t.Fatalf("expected no interactive control sequences in tty-stdout/file-stdin list output, got %q", output)
	}
}

func TestListMenuRenderIncludesQuitHintInAllModes(t *testing.T) {
	cases := []struct {
		name string
		menu listMenu
	}{
		{
			name: "profiles",
			menu: listMenu{
				profiles: []string{"beta", "demo"},
			},
		},
		{
			name: "actions",
			menu: listMenu{
				profiles: []string{"beta", "demo"},
				mode:     listMenuModeActions,
			},
		},
		{
			name: "delete-confirm",
			menu: listMenu{
				profiles: []string{"beta", "demo"},
				mode:     listMenuModeDeleteConfirm,
			},
		},
		{
			name: "empty",
			menu: listMenu{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rendered := tc.menu.render()
			if !strings.Contains(rendered, "按 q 或 Ctrl+C 退出") {
				t.Fatalf("expected list menu mode %s to include quit hint, got %q", tc.name, rendered)
			}
		})
	}
}

func TestRun_ListInteractiveSwitchesSelectedProfile(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"prod": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-prod",
					profile.EnvBaseURL:   "https://prod.example.com",
				},
			},
		},
	})
	settingsPath := writeSettingsFixture(t, `{"env":{"ANTHROPIC_AUTH_TOKEN":"old-token"}}`)

	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, "\x1b[B\x1b[B\r\r", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
		"CC_SWITCH_SETTINGS_PATH": settingsPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive list switch to succeed, output=%q", output)
	}

	if !strings.Contains(output, "配置列表：") || !strings.Contains(output, "操作：prod") {
		t.Fatalf("expected list and action menu output, got %q", output)
	}
	if !strings.Contains(output, "已切换到配置：prod") {
		t.Fatalf("expected switch success output, got %q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interactive list switch: %v", err)
	}
	if savedProfiles.Current != "prod" {
		t.Fatalf("expected current profile to switch to prod, got %q", savedProfiles.Current)
	}
}

func TestRun_ListInteractiveBackLeavesCurrentUnchanged(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, "\r\x1b[B\x1b[B\x1b[B\rq", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive list back flow to succeed, output=%q", output)
	}

	if !strings.Contains(output, "操作：beta") {
		t.Fatalf("expected action menu output, got %q", output)
	}
	if strings.Contains(output, "已切换到配置：") {
		t.Fatalf("expected back flow to avoid switching, got %q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interactive list back flow: %v", err)
	}
	if savedProfiles.Current != "demo" {
		t.Fatalf("expected current profile to remain demo, got %q", savedProfiles.Current)
	}
}

func TestRun_ListInteractiveUsesAlternateScreen(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, "q", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive list alt-screen flow to succeed, output=%q", output)
	}

	if !strings.Contains(output, "\x1b[?1049h") || !strings.Contains(output, "\x1b[?1049l") {
		t.Fatalf("expected interactive list to enter and leave alternate screen, got %q", output)
	}
}

func TestRun_ListInteractiveActionMenuQuitClosesAlternateScreen(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, "\rq", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected action-menu quit flow to succeed, output=%q", output)
	}

	if got := strings.Count(output, enterAlternateScreenMode); got != 1 {
		t.Fatalf("expected one alternate-screen enter in action-menu quit flow, got %d in %q", got, output)
	}
	if got := strings.Count(output, exitAlternateScreenMode); got != 1 {
		t.Fatalf("expected one alternate-screen exit in action-menu quit flow, got %d in %q", got, output)
	}
	if strings.Contains(output, "已切换到配置：") {
		t.Fatalf("expected action-menu quit to avoid switching, got %q", output)
	}
}

func TestRun_ListInteractiveCurrentProfileActionsHideRemove(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, "\x1b[B\rq", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive current-profile action menu to succeed, output=%q", output)
	}

	if !strings.Contains(output, "操作：demo") {
		t.Fatalf("expected current-profile action menu output, got %q", output)
	}
	if strings.Contains(output, "删除") {
		t.Fatalf("expected current-profile action menu to hide remove, got %q", output)
	}
}

func TestRun_ListInteractiveDeleteConfirmQuitClosesAlternateScreen(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, "\r\x1b[B\x1b[B\rq", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected delete-confirm quit flow to succeed, output=%q", output)
	}

	if got := strings.Count(output, enterAlternateScreenMode); got != 1 {
		t.Fatalf("expected one alternate-screen enter in delete-confirm quit flow, got %d in %q", got, output)
	}
	if got := strings.Count(output, exitAlternateScreenMode); got != 1 {
		t.Fatalf("expected one alternate-screen exit in delete-confirm quit flow, got %d in %q", got, output)
	}
	if strings.Contains(output, "已删除配置：") {
		t.Fatalf("expected delete-confirm quit to avoid deleting, got %q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after delete-confirm quit: %v", err)
	}
	if _, exists := savedProfiles.Profiles["beta"]; !exists {
		t.Fatalf("expected beta to remain after quitting delete confirm, got %#v", savedProfiles.Profiles)
	}
}

func TestRun_ListInteractiveCtrlCLeavesCurrentUnchanged(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, "\x03", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive list Ctrl+C to exit cleanly, output=%q", output)
	}

	if !strings.Contains(output, "配置列表：") {
		t.Fatalf("expected interactive list output, got %q", output)
	}
	if strings.Contains(output, "^C") || strings.Contains(strings.ToLower(output), "interrupt") {
		t.Fatalf("expected Ctrl+C to be handled as a clean exit, got %q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interactive list Ctrl+C: %v", err)
	}
	if savedProfiles.Current != "demo" {
		t.Fatalf("expected current profile to remain demo, got %q", savedProfiles.Current)
	}
}

func TestRun_ListInteractiveEditUpdatesProfileAndReturnsToList(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Description: "旧描述",
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	input := "\r\x1b[B\r新描述\n\n\n\n\n\n\nq"
	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, input, map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive list edit to succeed, output=%q", output)
	}

	if !strings.Contains(output, "操作：beta") || !strings.Contains(output, "已更新配置：beta") {
		t.Fatalf("expected edit flow output, got %q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interactive list edit: %v", err)
	}
	if savedProfiles.Profiles["beta"].Description != "新描述" {
		t.Fatalf("expected beta description to update, got %#v", savedProfiles.Profiles["beta"])
	}
}

func TestRun_ListInteractiveEditReentersAlternateScreenAfterSuccess(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Description: "旧描述",
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	})

	input := "\r\x1b[B\r新描述\n\n\n\n\n\n\nq"
	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, input, map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive list edit alt-screen flow to succeed, output=%q", output)
	}

	if got := strings.Count(output, enterAlternateScreenMode); got != 2 {
		t.Fatalf("expected two alternate-screen enters around edit success flow, got %d in %q", got, output)
	}
	if got := strings.Count(output, exitAlternateScreenMode); got != 2 {
		t.Fatalf("expected two alternate-screen exits around edit success flow, got %d in %q", got, output)
	}
	if !strings.Contains(output, "已更新配置：beta") {
		t.Fatalf("expected edit success output, got %q", output)
	}
}

func TestRun_ListInteractiveRemoveConfirmsAndRefreshesList(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"prod": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-prod",
					profile.EnvBaseURL:   "https://prod.example.com",
				},
			},
		},
	})

	input := "\r\x1b[B\x1b[B\r\rq"
	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, input, map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive list remove to succeed, output=%q", output)
	}

	if !strings.Contains(output, "确认删除：beta") || !strings.Contains(output, "已删除配置：beta") {
		t.Fatalf("expected delete confirmation flow output, got %q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interactive list remove: %v", err)
	}
	if _, exists := savedProfiles.Profiles["beta"]; exists {
		t.Fatalf("expected beta profile to be removed, got %#v", savedProfiles.Profiles)
	}
}

func TestRun_ListInteractiveRemoveReentersAlternateScreenAfterSuccess(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-demo",
					profile.EnvBaseURL:   "https://demo.example.com",
				},
			},
			"prod": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-prod",
					profile.EnvBaseURL:   "https://prod.example.com",
				},
			},
		},
	})

	input := "\r\x1b[B\x1b[B\r\rq"
	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, input, map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive list remove alt-screen flow to succeed, output=%q", output)
	}

	if got := strings.Count(output, enterAlternateScreenMode); got != 2 {
		t.Fatalf("expected two alternate-screen enters around remove success flow, got %d in %q", got, output)
	}
	if got := strings.Count(output, exitAlternateScreenMode); got != 2 {
		t.Fatalf("expected two alternate-screen exits around remove success flow, got %d in %q", got, output)
	}
	if !strings.Contains(output, "已删除配置：beta") {
		t.Fatalf("expected remove success output, got %q", output)
	}
}

func TestRun_ListInteractiveRemoveLastProfileShowsEmptyState(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})

	input := "\r\x1b[B\x1b[B\r\rq"
	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, input, map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected interactive list remove-last flow to succeed, output=%q", output)
	}

	if !strings.Contains(output, "已删除配置：beta") {
		t.Fatalf("expected delete success output, got %q", output)
	}
	if !strings.Contains(output, "配置列表为空") || !strings.Contains(output, interactiveQuitHint) {
		t.Fatalf("expected empty-state output after deleting last profile, got %q", output)
	}

	savedProfiles, err := profile.Load(profilesPath)
	if err != nil {
		t.Fatalf("load profiles after interactive list remove-last flow: %v", err)
	}
	if len(savedProfiles.Profiles) != 0 {
		t.Fatalf("expected all profiles to be removed, got %#v", savedProfiles.Profiles)
	}
}

func TestRun_ListInteractiveEmptyStateQuitClosesAlternateScreen(t *testing.T) {
	scriptPath := requireScript(t)
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Profiles: map[string]profile.Profile{
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-beta",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})

	exitCode, output := runWithTTYBestEffort(t, scriptPath, []string{"list"}, "\r\x1b[B\x1b[B\r\rq", map[string]string{
		"CC_SWITCH_PROFILES_PATH": profilesPath,
	})
	if exitCode != 0 {
		t.Fatalf("expected empty-state quit flow to succeed, output=%q", output)
	}

	if got := strings.Count(output, enterAlternateScreenMode); got != 2 {
		t.Fatalf("expected two alternate-screen enters around empty-state flow, got %d in %q", got, output)
	}
	if got := strings.Count(output, exitAlternateScreenMode); got != 2 {
		t.Fatalf("expected two alternate-screen exits around empty-state flow, got %d in %q", got, output)
	}
	if !strings.Contains(output, "配置列表为空") {
		t.Fatalf("expected empty-state output, got %q", output)
	}
}

func TestRun_NoArgsShowsBaseURLAndModel(t *testing.T) {
	profilesPath := writeProfilesFixture(t, profile.ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]profile.Profile{
			"demo": {
				Env: map[string]string{
					profile.EnvAuthToken: "token",
					profile.EnvBaseURL:   "https://example.com",
					"ANTHROPIC_MODEL":    "glm-5",
				},
			},
			"beta": {
				Env: map[string]string{
					profile.EnvAuthToken: "token-b",
					profile.EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	})

	t.Setenv("CC_SWITCH_PROFILES_PATH", profilesPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(nil, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	want := "当前配置：demo\n接口地址：https://example.com\n模型：glm-5\n可用配置：beta\n"
	if got := stdout.String(); got != want {
		t.Fatalf("expected status output %q, got %q", want, got)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("expected empty stderr, got %q", got)
	}
}

func writeProfilesFixture(t *testing.T, data profile.ProfilesFile) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "profiles.json")
	if err := profile.Save(path, data); err != nil {
		t.Fatalf("save profiles fixture: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture missing: %v", err)
	}

	return path
}

func writeSettingsFixture(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write settings fixture: %v", err)
	}

	return path
}

func withPromptSession(t *testing.T, input string) *bytes.Buffer {
	t.Helper()

	oldPromptReader := promptReader
	oldPromptWriter := promptWriter
	oldPromptInteractive := promptInteractive

	t.Cleanup(func() {
		promptReader = oldPromptReader
		promptWriter = oldPromptWriter
		promptInteractive = oldPromptInteractive
	})

	var output bytes.Buffer
	promptReader = strings.NewReader(input)
	promptWriter = &output
	promptInteractive = func() bool { return true }

	return &output
}

func requireScript(t *testing.T) string {
	t.Helper()

	path, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script command not available")
	}

	return path
}

func runWithTTY(t *testing.T, scriptPath string, args []string, input string, env map[string]string) (int, string) {
	t.Helper()

	cmdArgs := append([]string{"-q", "/dev/null", os.Args[0], "-test.run=^TestTTYHelperProcess$"}, append([]string{"--"}, args...)...)
	cmd := exec.Command(scriptPath, cmdArgs...)
	cmd.Env = append(os.Environ(), "GO_WANT_TTY_HELPER=1")
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start tty command: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if _, err := io.WriteString(stdin, input); err != nil {
		t.Fatalf("write tty input: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if err := stdin.Close(); err != nil {
		t.Fatalf("close tty stdin: %v", err)
	}

	err = cmd.Wait()
	if err == nil {
		return 0, output.String()
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("wait tty command: %v", err)
	}

	return exitErr.ExitCode(), output.String()
}

func runWithTTYBestEffort(t *testing.T, scriptPath string, args []string, input string, env map[string]string) (int, string) {
	t.Helper()

	cmdArgs := append([]string{"-q", "/dev/null", os.Args[0], "-test.run=^TestTTYHelperProcess$"}, append([]string{"--"}, args...)...)
	cmd := exec.Command(scriptPath, cmdArgs...)
	cmd.Env = append(os.Environ(), "GO_WANT_TTY_HELPER=1")
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start tty command: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if input != "" {
		if _, err := io.WriteString(stdin, input); err != nil && !strings.Contains(err.Error(), "broken pipe") {
			t.Fatalf("write tty input: %v", err)
		}
	}
	time.Sleep(100 * time.Millisecond)
	_ = stdin.Close()

	err = cmd.Wait()
	if err == nil {
		return 0, output.String()
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("wait tty command: %v", err)
	}

	return exitErr.ExitCode(), output.String()
}

func runWithTTYStdoutAndFileStdin(t *testing.T, scriptPath string, args []string, input string, env map[string]string) (int, string) {
	t.Helper()

	inputPath := filepath.Join(t.TempDir(), "stdin.txt")
	if err := os.WriteFile(inputPath, []byte(input), 0o644); err != nil {
		t.Fatalf("write stdin fixture: %v", err)
	}

	commandArgs := append([]string{shellQuote(os.Args[0]), "-test.run=^TestTTYHelperProcess$", "--"}, quoteShellArgs(args)...)
	shellCommand := "exec < " + shellQuote(inputPath) + "; " + strings.Join(commandArgs, " ")

	cmd := exec.Command(scriptPath, "-q", "/dev/null", "sh", "-c", shellCommand)
	cmd.Env = append(os.Environ(), "GO_WANT_TTY_HELPER=1")
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err == nil {
		return 0, trimScriptEOFArtifact(output.String())
	} else {
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run tty-stdout/file-stdin command: %v", err)
		}
		return exitErr.ExitCode(), trimScriptEOFArtifact(output.String())
	}
}

func quoteShellArgs(args []string) []string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return quoted
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func trimScriptEOFArtifact(output string) string {
	return strings.TrimPrefix(output, "^D\b\b")
}

func helperArgs(argv []string) []string {
	for i, arg := range argv {
		if arg == "--" {
			return argv[i+1:]
		}
	}
	return nil
}
