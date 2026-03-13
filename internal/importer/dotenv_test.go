package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportDir_ParsesLegacyEnvFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.env")
	content := "# Demo profile\nexport ANTHROPIC_AUTH_TOKEN=\"token-demo\"\nexport ANTHROPIC_BASE_URL=\"https://demo.example.com\"\nexport ANTHROPIC_MODEL=\"glm-5\"\nexport EXTRA_FIELD=\"ignored\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write env fixture: %v", err)
	}

	result, err := ImportDir(dir, false, nil)
	if err != nil {
		t.Fatalf("import dir failed: %v", err)
	}

	imported, ok := result.Imported["demo"]
	if !ok {
		t.Fatal("expected demo profile to be imported")
	}

	if imported.Description != "Demo profile" {
		t.Fatalf("expected description to be parsed, got %q", imported.Description)
	}

	if imported.Env["ANTHROPIC_MODEL"] != "glm-5" {
		t.Fatalf("expected optional model to be imported, got %q", imported.Env["ANTHROPIC_MODEL"])
	}

	if _, exists := imported.Env["EXTRA_FIELD"]; exists {
		t.Fatal("expected unsupported env key to be ignored")
	}
}

func TestImportDir_SkipsMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.env")
	content := "# Broken\nexport ANTHROPIC_AUTH_TOKEN=\"token-demo\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write env fixture: %v", err)
	}

	result, err := ImportDir(dir, false, nil)
	if err != nil {
		t.Fatalf("import dir failed: %v", err)
	}

	if len(result.Imported) != 0 {
		t.Fatalf("expected invalid profiles to be skipped, got %#v", result.Imported)
	}

	if len(result.Skipped) != 1 || result.Skipped[0] != "broken" {
		t.Fatalf("expected broken profile to be skipped, got %#v", result.Skipped)
	}
}

func TestImportDir_DoesNotOverwriteWithoutFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.env")
	content := "# Demo profile\nexport ANTHROPIC_AUTH_TOKEN=\"token-demo\"\nexport ANTHROPIC_BASE_URL=\"https://demo.example.com\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write env fixture: %v", err)
	}

	result, err := ImportDir(dir, false, map[string]struct{}{"demo": {}})
	if err != nil {
		t.Fatalf("import dir failed: %v", err)
	}

	if len(result.Imported) != 0 {
		t.Fatalf("expected existing profile not to be overwritten, got %#v", result.Imported)
	}

	if len(result.Skipped) != 1 || result.Skipped[0] != "demo" {
		t.Fatalf("expected demo profile to be skipped, got %#v", result.Skipped)
	}
}
