package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteEnv_OnlyEnvIsUpdated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	original := `{
  "model": "opus",
  "statusLine": {
    "type": "command",
    "command": "demo"
  },
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "old-token"
  }
}
`

	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	now := func() time.Time {
		return time.Date(2026, 3, 13, 15, 0, 0, 0, time.UTC)
	}

	newEnv := map[string]string{
		"ANTHROPIC_AUTH_TOKEN": "new-token",
		"ANTHROPIC_BASE_URL":   "https://example.com",
	}

	if err := WriteEnv(path, newEnv, now); err != nil {
		t.Fatalf("write env failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated settings: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("unmarshal updated settings: %v", err)
	}

	if got["model"] != "opus" {
		t.Fatalf("expected model to remain unchanged, got %#v", got["model"])
	}

	statusLine, ok := got["statusLine"].(map[string]any)
	if !ok || statusLine["command"] != "demo" {
		t.Fatalf("expected statusLine to remain unchanged, got %#v", got["statusLine"])
	}

	env, ok := got["env"].(map[string]any)
	if !ok {
		t.Fatalf("expected env object, got %#v", got["env"])
	}

	if env["ANTHROPIC_AUTH_TOKEN"] != "new-token" {
		t.Fatalf("expected token to be updated, got %#v", env["ANTHROPIC_AUTH_TOKEN"])
	}

	if env["ANTHROPIC_BASE_URL"] != "https://example.com" {
		t.Fatalf("expected base url to be updated, got %#v", env["ANTHROPIC_BASE_URL"])
	}
}

func TestWriteEnv_CreatesBackupBeforeWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	t.Setenv("HOME", dir)

	if err := os.WriteFile(path, []byte(`{"env":{"ANTHROPIC_AUTH_TOKEN":"old-token"}}`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	now := func() time.Time {
		return time.Date(2026, 3, 13, 15, 1, 2, 0, time.UTC)
	}

	if err := WriteEnv(path, map[string]string{
		"ANTHROPIC_AUTH_TOKEN": "new-token",
		"ANTHROPIC_BASE_URL":   "https://example.com",
	}, now); err != nil {
		t.Fatalf("write env failed: %v", err)
	}

	backupPath := filepath.Join(dir, ".claude", "cc-switch", "backups", "settings.json.20260313T150102Z.bak")
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}

	if string(backupContent) != `{"env":{"ANTHROPIC_AUTH_TOKEN":"old-token"}}` {
		t.Fatalf("unexpected backup content: %s", string(backupContent))
	}
}

func TestWriteEnv_InvalidJSONFailsWithoutMutation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	original := `{"env":`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	now := func() time.Time {
		return time.Date(2026, 3, 13, 15, 1, 2, 0, time.UTC)
	}

	err := WriteEnv(path, map[string]string{
		"ANTHROPIC_AUTH_TOKEN": "new-token",
		"ANTHROPIC_BASE_URL":   "https://example.com",
	}, now)
	if err == nil {
		t.Fatal("expected invalid json to fail")
	}

	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read original file: %v", readErr)
	}

	if string(content) != original {
		t.Fatalf("expected invalid file to remain unchanged, got %s", string(content))
	}
}

func TestWriteEnv_BackupDirUnwritableFailsWithoutMutation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	homeFile := filepath.Join(dir, "home-file")
	t.Setenv("HOME", homeFile)

	original := `{"env":{"ANTHROPIC_AUTH_TOKEN":"old-token"}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	if err := os.WriteFile(homeFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write fake home file: %v", err)
	}

	now := func() time.Time {
		return time.Date(2026, 3, 13, 15, 1, 2, 0, time.UTC)
	}

	err := WriteEnv(path, map[string]string{
		"ANTHROPIC_AUTH_TOKEN": "new-token",
		"ANTHROPIC_BASE_URL":   "https://example.com",
	}, now)
	if err == nil {
		t.Fatal("expected unwritable backup dir to fail")
	}

	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read original file: %v", readErr)
	}

	if string(content) != original {
		t.Fatalf("expected original settings to remain unchanged, got %s", string(content))
	}
}
