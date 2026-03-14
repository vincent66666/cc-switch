package profile

import (
	"errors"
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

func TestStore_LoadNormalizesSpacedProfileNamesAndCurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	content := `{
  "version": 1,
  "current": " demo ",
  "profiles": {
    " demo ": {
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
		t.Fatalf("expected normalized current demo, got %q", got.Current)
	}

	if _, exists := got.Profiles["demo"]; !exists {
		t.Fatalf("expected normalized profile key, got %#v", got.Profiles)
	}
	if _, exists := got.Profiles[" demo "]; exists {
		t.Fatalf("expected spaced profile key to be normalized away, got %#v", got.Profiles)
	}
}

func TestStore_LoadRejectsDuplicateNormalizedProfileNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	content := `{
  "version": 1,
  "profiles": {
    "demo": {
      "env": {
        "ANTHROPIC_AUTH_TOKEN": "token-demo",
        "ANTHROPIC_BASE_URL": "https://demo.example.com"
      }
    },
    " demo ": {
      "env": {
        "ANTHROPIC_AUTH_TOKEN": "token-spaced",
        "ANTHROPIC_BASE_URL": "https://spaced.example.com"
      }
    }
  }
}
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := Load(path)
	if err == nil || err.Error() != "配置 \"demo\" 已存在" {
		t.Fatalf("expected normalized duplicate load error, got %v", err)
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

func TestStore_SaveNormalizesSpacedProfileNamesAndCurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Current: " demo ",
		Profiles: map[string]Profile{
			" demo ": {
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
		t.Fatalf("expected normalized saved content, got:\n%s", string(content))
	}
}

func TestStore_LoadRejectsBlankNormalizedProfileName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	content := `{
  "version": 1,
  "profiles": {
    "   ": {
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

	_, err := Load(path)
	if err == nil || err.Error() != "必须提供配置名称" {
		t.Fatalf("expected blank normalized name load error, got %v", err)
	}
}

func TestStore_LoadRejectsNormalizedCurrentWhenProfileIsMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	content := `{
  "version": 1,
  "current": " ghost ",
  "profiles": {
    "demo": {
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

	_, err := Load(path)
	if err == nil || err.Error() != "当前配置 \"ghost\" 不存在" {
		t.Fatalf("expected missing current load error, got %v", err)
	}
	if !errors.Is(err, ErrCurrentProfileMissing) {
		t.Fatalf("expected missing current load error to match sentinel, got %v", err)
	}
}

func TestStore_LoadForListReturnsEmptyProfilesWhenFileIsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.json")

	got, err := LoadForList(path)
	if err != nil {
		t.Fatalf("load for list failed: %v", err)
	}

	if got.Version != 1 {
		t.Fatalf("expected version 1 for missing file, got %d", got.Version)
	}
	if len(got.Profiles) != 0 {
		t.Fatalf("expected empty profiles for missing file, got %#v", got.Profiles)
	}
}

func TestStore_LoadForListAllowsMissingCurrentProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	content := `{
  "version": 1,
  "current": " ghost ",
  "profiles": {
    " demo ": {
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

	got, err := LoadForList(path)
	if err != nil {
		t.Fatalf("load for list failed: %v", err)
	}

	if got.Current != "ghost" {
		t.Fatalf("expected normalized missing current ghost, got %q", got.Current)
	}
	if _, exists := got.Profiles["demo"]; !exists {
		t.Fatalf("expected normalized profile key for list load, got %#v", got.Profiles)
	}
}

func TestStore_LoadForListRejectsInvalidProfilesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := LoadForList(path)
	if err == nil {
		t.Fatalf("expected invalid profiles file error")
	}
}

func TestStore_SaveRejectsDuplicateNormalizedProfileNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Profiles: map[string]Profile{
			"demo": {
				Env: map[string]string{
					EnvAuthToken: "token-demo",
					EnvBaseURL:   "https://demo.example.com",
				},
			},
			" demo ": {
				Env: map[string]string{
					EnvAuthToken: "token-spaced",
					EnvBaseURL:   "https://spaced.example.com",
				},
			},
		},
	}

	err := Save(path, data)
	if err == nil || err.Error() != "配置 \"demo\" 已存在" {
		t.Fatalf("expected normalized duplicate save error, got %v", err)
	}
}

func TestStore_SaveRejectsBlankNormalizedProfileName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Profiles: map[string]Profile{
			"   ": {
				Env: map[string]string{
					EnvAuthToken: "token",
					EnvBaseURL:   "https://example.com",
				},
			},
		},
	}

	err := Save(path, data)
	if err == nil || err.Error() != "必须提供配置名称" {
		t.Fatalf("expected blank normalized name save error, got %v", err)
	}
}

func TestStore_SaveRejectsNormalizedCurrentWhenProfileIsMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Current: " ghost ",
		Profiles: map[string]Profile{
			"demo": {
				Env: map[string]string{
					EnvAuthToken: "token",
					EnvBaseURL:   "https://example.com",
				},
			},
		},
	}

	err := Save(path, data)
	if err == nil || err.Error() != "当前配置 \"ghost\" 不存在" {
		t.Fatalf("expected missing current save error, got %v", err)
	}
	if !errors.Is(err, ErrCurrentProfileMissing) {
		t.Fatalf("expected missing current save error to match sentinel, got %v", err)
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

func TestStore_SetCurrentProfileTrimsName(t *testing.T) {
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

	if err := SetCurrent(path, " demo "); err != nil {
		t.Fatalf("set current with spaced name failed: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if got.Current != "demo" {
		t.Fatalf("expected normalized current demo, got %q", got.Current)
	}
}

func TestStore_SetCurrentProfileRejectsBlankName(t *testing.T) {
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

	err := SetCurrent(path, "   ")
	if err == nil || err.Error() != "必须提供配置名称" {
		t.Fatalf("expected blank-name set current error, got %v", err)
	}
}

func TestStore_RemoveRejectsActiveProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Current: "demo",
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

	err := Remove(path, "demo")
	if err == nil || err.Error() != "不能删除当前正在使用的配置" {
		t.Fatalf("expected active-profile remove error, got %v", err)
	}
}

func TestStore_RemoveTrimsName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]Profile{
			"demo": {
				Env: map[string]string{
					EnvAuthToken: "token-demo",
					EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					EnvAuthToken: "token-beta",
					EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	}

	if err := Save(path, data); err != nil {
		t.Fatalf("save fixture: %v", err)
	}

	if err := Remove(path, " beta "); err != nil {
		t.Fatalf("remove with spaced name failed: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if _, exists := got.Profiles["beta"]; exists {
		t.Fatalf("expected beta to be removed after normalization, got %#v", got.Profiles)
	}
}

func TestStore_RemoveRejectsBlankName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]Profile{
			"demo": {
				Env: map[string]string{
					EnvAuthToken: "token-demo",
					EnvBaseURL:   "https://demo.example.com",
				},
			},
			"beta": {
				Env: map[string]string{
					EnvAuthToken: "token-beta",
					EnvBaseURL:   "https://beta.example.com",
				},
			},
		},
	}

	if err := Save(path, data); err != nil {
		t.Fatalf("save fixture: %v", err)
	}

	err := Remove(path, "   ")
	if err == nil || err.Error() != "必须提供配置名称" {
		t.Fatalf("expected blank-name remove error, got %v", err)
	}
}

func TestStore_RenameRejectsDuplicateName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Profiles: map[string]Profile{
			"demo": {
				Env: map[string]string{
					EnvAuthToken: "token-demo",
					EnvBaseURL:   "https://demo.example.com",
				},
			},
			"prod": {
				Env: map[string]string{
					EnvAuthToken: "token-prod",
					EnvBaseURL:   "https://prod.example.com",
				},
			},
		},
	}

	if err := Save(path, data); err != nil {
		t.Fatalf("save fixture: %v", err)
	}

	err := Rename(path, "demo", "prod")
	if err == nil || err.Error() != "配置 \"prod\" 已存在" {
		t.Fatalf("expected duplicate rename error, got %v", err)
	}
}

func TestStore_RenameTrimsNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Current: "demo",
		Profiles: map[string]Profile{
			"demo": {
				Env: map[string]string{
					EnvAuthToken: "token-demo",
					EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	}

	if err := Save(path, data); err != nil {
		t.Fatalf("save fixture: %v", err)
	}

	if err := Rename(path, " demo ", " prod "); err != nil {
		t.Fatalf("rename with spaced names failed: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if got.Current != "prod" {
		t.Fatalf("expected normalized current prod, got %q", got.Current)
	}
	if _, exists := got.Profiles["demo"]; exists {
		t.Fatalf("expected old demo profile to be removed, got %#v", got.Profiles)
	}
	if _, exists := got.Profiles["prod"]; !exists {
		t.Fatalf("expected prod profile to exist after normalization, got %#v", got.Profiles)
	}
}

func TestStore_RenameRejectsNormalizedDuplicateName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Profiles: map[string]Profile{
			"demo": {
				Env: map[string]string{
					EnvAuthToken: "token-demo",
					EnvBaseURL:   "https://demo.example.com",
				},
			},
			"prod": {
				Env: map[string]string{
					EnvAuthToken: "token-prod",
					EnvBaseURL:   "https://prod.example.com",
				},
			},
		},
	}

	if err := Save(path, data); err != nil {
		t.Fatalf("save fixture: %v", err)
	}

	err := Rename(path, " demo ", " prod ")
	if err == nil || err.Error() != "配置 \"prod\" 已存在" {
		t.Fatalf("expected normalized duplicate rename error, got %v", err)
	}
}

func TestStore_RenameRejectsBlankName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.json")

	data := ProfilesFile{
		Version: 1,
		Profiles: map[string]Profile{
			"demo": {
				Env: map[string]string{
					EnvAuthToken: "token-demo",
					EnvBaseURL:   "https://demo.example.com",
				},
			},
		},
	}

	if err := Save(path, data); err != nil {
		t.Fatalf("save fixture: %v", err)
	}

	err := Rename(path, "demo", "   ")
	if err == nil || err.Error() != "必须提供旧配置名称和新配置名称" {
		t.Fatalf("expected blank-name rename error, got %v", err)
	}
}
