package cli

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"cc-switch/internal/output"
	"cc-switch/internal/profile"
	"cc-switch/internal/settings"
)

type Paths struct {
	Profiles string
	Settings string
}

var (
	promptReader      io.Reader = os.Stdin
	promptWriter      io.Writer = os.Stdout
	promptInteractive           = func() bool {
		stat, err := os.Stdin.Stat()
		if err != nil {
			return false
		}

		return stat.Mode()&os.ModeCharDevice != 0
	}
)

func Run(args []string, stdout, stderr io.Writer) int {
	command := Parse(args)
	paths := defaultPaths()

	switch command.Name {
	case "current":
		return runCurrent(paths, stdout, stderr)
	case "list":
		return runList(paths, stdout, stderr)
	case "status":
		return runStatus(paths, stdout, stderr)
	case "use":
		return runUse(paths, command.Args, stdout, stderr)
	case "add":
		return runAdd(paths, command.Args, stdout, stderr)
	case "edit":
		return runEdit(paths, command.Args, stdout, stderr)
	case "remove":
		return runRemove(paths, command.Args, stdout, stderr)
	case "rename":
		return runRename(paths, command.Args, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "未知命令：%s\n", command.Name)
		return 1
	}
}

func defaultPaths() Paths {
	profilesPath := os.Getenv("CC_SWITCH_PROFILES_PATH")
	if profilesPath == "" {
		profilesPath = os.ExpandEnv("$HOME/.claude/cc-switch/profiles.json")
	}

	settingsPath := os.Getenv("CC_SWITCH_SETTINGS_PATH")
	if settingsPath == "" {
		settingsPath = os.ExpandEnv("$HOME/.claude/settings.json")
	}

	return Paths{
		Profiles: profilesPath,
		Settings: settingsPath,
	}
}

func runCurrent(paths Paths, stdout, stderr io.Writer) int {
	data, err := profile.Load(paths.Profiles)
	if err != nil {
		_, _ = io.WriteString(stdout, "未知\n")
		return 0
	}

	if data.Current == "" {
		_, _ = io.WriteString(stdout, "未知\n")
		return 0
	}

	_, _ = fmt.Fprintf(stdout, "%s\n", data.Current)
	return 0
}

func runList(paths Paths, stdout, stderr io.Writer) int {
	data, err := profile.Load(paths.Profiles)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}

	names := profileNames(data.Profiles)
	return output.RenderList(stdout, names)
}

func runStatus(paths Paths, stdout, stderr io.Writer) int {
	data, err := profile.Load(paths.Profiles)
	if err != nil {
		_, _ = io.WriteString(stdout, "当前配置：未知\n")
		return 0
	}

	currentProfile, ok := data.Profiles[data.Current]
	if !ok {
		_, _ = io.WriteString(stdout, "当前配置：未知\n")
		return 0
	}

	return output.RenderStatus(stdout, data.Current, currentProfile, availableNames(data.Profiles, data.Current))
}

func runUse(paths Paths, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, "必须提供配置名称\n")
		return 1
	}

	target := args[0]

	data, err := profile.Load(paths.Profiles)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}

	targetProfile, ok := data.Profiles[target]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "未找到配置 %q\n", target)
		return 1
	}

	if err := profile.ValidateProfile(target, targetProfile); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	if err := settings.WriteEnv(paths.Settings, targetProfile.Env, time.Now); err != nil {
		_, _ = fmt.Fprintf(stderr, "写入 settings.json 的 env 失败：%v\n", err)
		return 1
	}

	if err := profile.SetCurrent(paths.Profiles, target); err != nil {
		_, _ = fmt.Fprintf(stderr, "更新当前配置失败：%v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "已切换到配置：%s\n", target)
	return 0
}

func profileNames(profiles map[string]profile.Profile) []string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func availableNames(profiles map[string]profile.Profile, current string) []string {
	names := make([]string, 0, len(profiles))
	for _, name := range profileNames(profiles) {
		if name == current {
			continue
		}
		names = append(names, name)
	}
	return names
}

type profileFlags struct {
	description   string
	token         string
	baseURL       string
	model         string
	defaultOpus   string
	defaultSonnet string
	defaultHaiku  string
}

func runAdd(paths Paths, args []string, stdout, stderr io.Writer) int {
	name, input, err := parseProfileFlags(args, false)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	var promptSession *bufio.Reader
	if promptInteractive() {
		promptSession = bufio.NewReader(promptReader)
	}

	data, err := profile.Load(paths.Profiles)
	if err != nil && !os.IsNotExist(err) {
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}
	if os.IsNotExist(err) {
		data = profile.ProfilesFile{Version: 1, Profiles: map[string]profile.Profile{}}
	}

	if promptInteractive() && strings.TrimSpace(name) == "" {
		name, err = promptAddName(promptSession)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
			return 1
		}
	}

	if strings.TrimSpace(name) == "" {
		_, _ = io.WriteString(stderr, "必须提供配置名称\n")
		return 1
	}

	if _, exists := data.Profiles[name]; exists {
		_, _ = fmt.Fprintf(stderr, "配置 %q 已存在\n", name)
		return 1
	}

	if promptInteractive() {
		input, err = promptAddFields(promptSession, input)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
			return 1
		}
	}

	newProfile := profile.Profile{
		Description: input.description,
		Env:         buildProfileEnv(input, nil),
	}

	if err := profile.ValidateProfile(name, newProfile); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	data.Profiles[name] = newProfile
	if err := profile.Save(paths.Profiles, data); err != nil {
		_, _ = fmt.Fprintf(stderr, "保存配置失败：%v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "已添加配置：%s\n", name)
	return 0
}

func runEdit(paths Paths, args []string, stdout, stderr io.Writer) int {
	name, input, err := parseProfileFlags(args, true)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	data, err := profile.Load(paths.Profiles)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "加载配置失败：%v\n", err)
		return 1
	}

	existing, ok := data.Profiles[name]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "未找到配置 %q\n", name)
		return 1
	}

	if input.description != "" {
		existing.Description = input.description
	}
	if promptInteractive() {
		existing, err = promptEditFields(existing, input)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
			return 1
		}
	} else {
		existing.Env = buildProfileEnv(input, existing.Env)
	}

	if err := profile.ValidateProfile(name, existing); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	data.Profiles[name] = existing
	if err := profile.Save(paths.Profiles, data); err != nil {
		_, _ = fmt.Fprintf(stderr, "保存配置失败：%v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "已更新配置：%s\n", name)
	return 0
}

func parseProfileFlags(args []string, requireName bool) (string, profileFlags, error) {
	name := ""
	flagArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name = args[0]
		flagArgs = args[1:]
	}

	if requireName && strings.TrimSpace(name) == "" {
		return "", profileFlags{}, fmt.Errorf("必须提供配置名称")
	}

	flags := flag.NewFlagSet("profile", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var input profileFlags
	flags.StringVar(&input.description, "description", "", "profile description")
	flags.StringVar(&input.token, "token", "", "anthropic auth token")
	flags.StringVar(&input.baseURL, "base-url", "", "anthropic base url")
	flags.StringVar(&input.model, "model", "", "anthropic model")
	flags.StringVar(&input.defaultOpus, "default-opus-model", "", "default opus model")
	flags.StringVar(&input.defaultSonnet, "default-sonnet-model", "", "default sonnet model")
	flags.StringVar(&input.defaultHaiku, "default-haiku-model", "", "default haiku model")

	if err := flags.Parse(flagArgs); err != nil {
		return "", profileFlags{}, err
	}

	return name, input, nil
}

func buildProfileEnv(input profileFlags, existing map[string]string) map[string]string {
	env := map[string]string{}
	for key, value := range existing {
		env[key] = value
	}

	if input.token != "" {
		env[profile.EnvAuthToken] = input.token
	}
	if input.baseURL != "" {
		env[profile.EnvBaseURL] = input.baseURL
	}
	if input.model != "" {
		env["ANTHROPIC_MODEL"] = input.model
	}
	if input.defaultOpus != "" {
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = input.defaultOpus
	}
	if input.defaultSonnet != "" {
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = input.defaultSonnet
	}
	if input.defaultHaiku != "" {
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = input.defaultHaiku
	}

	return env
}

func promptAddName(reader *bufio.Reader) (string, error) {
	return promptAddValue(reader, "名称", "", true, false)
}

func promptAddFields(reader *bufio.Reader, input profileFlags) (profileFlags, error) {
	var err error
	if input.description == "" {
		input.description, err = promptAddValue(reader, "描述", "（可选）", false, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.token == "" {
		input.token, err = promptAddValue(reader, profile.EnvAuthToken, "", true, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.baseURL == "" {
		input.baseURL, err = promptAddValue(reader, profile.EnvBaseURL, "", true, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.model == "" {
		input.model, err = promptAddValue(reader, "ANTHROPIC_MODEL", "（可选）", false, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.defaultOpus == "" {
		input.defaultOpus, err = promptAddValue(reader, "ANTHROPIC_DEFAULT_OPUS_MODEL", "（可选）", false, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.defaultSonnet == "" {
		input.defaultSonnet, err = promptAddValue(reader, "ANTHROPIC_DEFAULT_SONNET_MODEL", "（可选）", false, false)
		if err != nil {
			return profileFlags{}, err
		}
	}
	if input.defaultHaiku == "" {
		input.defaultHaiku, err = promptAddValue(reader, "ANTHROPIC_DEFAULT_HAIKU_MODEL", "（可选）", false, false)
		if err != nil {
			return profileFlags{}, err
		}
	}

	return input, nil
}

func promptEditFields(existing profile.Profile, input profileFlags) (profile.Profile, error) {
	reader := bufio.NewReader(promptReader)
	var err error
	if input.description == "" {
		var keepCurrent bool
		existing.Description, keepCurrent, err = promptEditValue(reader, "描述", existing.Description, false, false)
		if err != nil {
			return profile.Profile{}, err
		}
		_ = keepCurrent
	} else {
		existing.Description = input.description
	}

	existing.Env, err = applyEditPrompt(reader, existing.Env, profile.EnvAuthToken, input.token, true, true)
	if err != nil {
		return profile.Profile{}, err
	}
	existing.Env, err = applyEditPrompt(reader, existing.Env, profile.EnvBaseURL, input.baseURL, true, false)
	if err != nil {
		return profile.Profile{}, err
	}
	existing.Env, err = applyEditPrompt(reader, existing.Env, "ANTHROPIC_MODEL", input.model, false, false)
	if err != nil {
		return profile.Profile{}, err
	}
	existing.Env, err = applyEditPrompt(reader, existing.Env, "ANTHROPIC_DEFAULT_OPUS_MODEL", input.defaultOpus, false, false)
	if err != nil {
		return profile.Profile{}, err
	}
	existing.Env, err = applyEditPrompt(reader, existing.Env, "ANTHROPIC_DEFAULT_SONNET_MODEL", input.defaultSonnet, false, false)
	if err != nil {
		return profile.Profile{}, err
	}
	existing.Env, err = applyEditPrompt(reader, existing.Env, "ANTHROPIC_DEFAULT_HAIKU_MODEL", input.defaultHaiku, false, false)
	if err != nil {
		return profile.Profile{}, err
	}

	return existing, nil
}

func applyEditPrompt(reader *bufio.Reader, env map[string]string, field, explicit string, required, sensitive bool) (map[string]string, error) {
	if explicit != "" {
		env[field] = explicit
		return env, nil
	}

	currentValue, exists := env[field]
	value, keepCurrent, err := promptEditValue(reader, field, currentValue, required, sensitive)
	if err != nil {
		return nil, err
	}

	if keepCurrent {
		if !exists {
			delete(env, field)
			return env, nil
		}
		env[field] = currentValue
		return env, nil
	}

	env[field] = value
	return env, nil
}

func promptAddValue(reader *bufio.Reader, label, suffix string, required, sensitive bool) (string, error) {
	_ = sensitive
	_, _ = fmt.Fprintf(promptWriter, "%s%s：", label, suffix)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	value = strings.TrimSpace(value)
	if required && value == "" {
		return "", fmt.Errorf("缺少必填字段：%s", label)
	}

	return value, nil
}

func promptEditValue(reader *bufio.Reader, label, current string, required, sensitive bool) (string, bool, error) {
	display := current
	if sensitive {
		display = maskValue(current)
	}

	if display != "" {
		_, _ = fmt.Fprintf(promptWriter, "%s [%s]（直接回车保留当前值）：", label, display)
	} else {
		_, _ = fmt.Fprintf(promptWriter, "%s（直接回车保留当前值）：", label)
	}

	value, err := reader.ReadString('\n')
	if err != nil {
		return "", false, err
	}

	value = strings.TrimSpace(value)
	if value == "" {
		if required && strings.TrimSpace(current) == "" {
			return "", false, fmt.Errorf("缺少必填字段：%s", label)
		}
		return current, true, nil
	}

	return value, false, nil
}

func maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}

	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func runRemove(paths Paths, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, "必须提供配置名称\n")
		return 1
	}

	if err := profile.Remove(paths.Profiles, args[0]); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "已删除配置：%s\n", args[0])
	return 0
}

func runRename(paths Paths, args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		_, _ = io.WriteString(stderr, "必须提供旧配置名称和新配置名称\n")
		return 1
	}

	if err := profile.Rename(paths.Profiles, args[0], args[1]); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", formatCLIError(err))
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "已将配置 %s 重命名为 %s\n", args[0], args[1])
	return 0
}

func formatCLIError(err error) string {
	if errors.Is(err, io.EOF) {
		return "输入已结束"
	}

	return err.Error()
}
