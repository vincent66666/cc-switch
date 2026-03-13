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
		return fmt.Errorf("profile name is required")
	}

	if len(profile.Env) == 0 {
		return fmt.Errorf("profile %q env is required", name)
	}

	for key := range profile.Env {
		if _, ok := SupportedEnvKeys[key]; !ok {
			return fmt.Errorf("profile %q contains unsupported field: %s", name, key)
		}
	}

	for _, key := range []string{EnvAuthToken, EnvBaseURL} {
		if strings.TrimSpace(profile.Env[key]) == "" {
			return fmt.Errorf("profile %q missing required field: %s", name, key)
		}
	}

	return nil
}
