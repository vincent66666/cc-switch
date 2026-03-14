package profile

import (
	"fmt"
	"strings"
)

const (
	EnvAuthToken = "ANTHROPIC_AUTH_TOKEN"
	EnvBaseURL   = "ANTHROPIC_BASE_URL"
)

func ValidateProfile(name string, profile Profile) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("必须提供配置名称")
	}

	if len(profile.Env) == 0 {
		return fmt.Errorf("配置 %q 缺少 env", name)
	}

	for key := range profile.Env {
		if _, ok := SupportedEnvKeys[key]; !ok {
			return fmt.Errorf("配置 %q 包含不支持的字段：%s", name, key)
		}
	}

	for _, key := range []string{EnvAuthToken, EnvBaseURL} {
		if strings.TrimSpace(profile.Env[key]) == "" {
			return fmt.Errorf("配置 %q 缺少必填字段：%s", name, key)
		}
	}

	return nil
}
