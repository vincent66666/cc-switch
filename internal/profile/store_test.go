package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_LoadProfilesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	content := `{
  "version": 1,
  "current": "demo",
  "profiles": {
    "demo": {
      "description": "Demo",
      "env": {
        "ANTHROPIC_AUTH_TOKEN": "token",
        "ANTHROPIC_BASE_URL": "https://example.com"
      }
    }
  }
}
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if got.Current != "demo" {
		t.Fatalf("expected current demo, got %q", got.Current)
	}

	if got.Profiles["demo"].Description != "Demo" {
		t.Fatalf("expected description to round-trip, got %q", got.Profiles["demo"].Description)
	}
}

func TestStore_SaveProfilesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]Profile{
			"demo": {
				Description: "Demo",
				Env: map[string]string{
					EnvAuthToken: "token",
					EnvBaseURL:   "https://example.com",
				},
			},
		},
	}

	if err := Save(path, data); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}

	want := "{\n  \"version\": 1,\n  \"current\": \"demo\",\n  \"profiles\": {\n    \"demo\": {\n      \"description\": \"Demo\",\n      \"env\": {\n        \"ANTHROPIC_AUTH_TOKEN\": \"token\",\n        \"ANTHROPIC_BASE_URL\": \"https://example.com\"\n      }\n    }\n  }\n}\n"
	if string(content) != want {
		t.Fatalf("unexpected saved content:\n%s", string(content))
	}
}

func TestStore_SetCurrentProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Profiles: map[string]Profile{
			"demo": {
				Env: map[string]string{
					EnvAuthToken: "token",
					EnvBaseURL:   "https://example.com",
				},
			},
		},
	}

	if err := Save(path, data); err != nil {
		t.Fatalf("save fixture: %v", err)
	}

	if err := SetCurrent(path, "demo"); err != nil {
		t.Fatalf("set current failed: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if got.Current != "demo" {
		t.Fatalf("expected current demo, got %q", got.Current)
	}
}
