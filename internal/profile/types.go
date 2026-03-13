package profile

type Profile struct {
	Description string            `json:"description,omitempty"`
	Env         map[string]string `json:"env"`
}

type ProfilesFile struct {
	Version  int                `json:"version"`
	Current  string             `json:"current,omitempty"`
	Profiles map[string]Profile `json:"profiles"`
}

var SupportedEnvKeys = map[string]struct{}{
	"ANTHROPIC_AUTH_TOKEN":           {},
	"ANTHROPIC_BASE_URL":             {},
	"ANTHROPIC_MODEL":                {},
	"ANTHROPIC_DEFAULT_OPUS_MODEL":   {},
	"ANTHROPIC_DEFAULT_SONNET_MODEL": {},
	"ANTHROPIC_DEFAULT_HAIKU_MODEL":  {},
}
