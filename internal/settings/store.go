package settings

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

func WriteEnv(path string, env map[string]string, now func() time.Time) error {
	current := []byte("{}")

	if content, err := os.ReadFile(path); err == nil {
		current = content
	} else if !os.IsNotExist(err) {
		return err
	}

	var doc map[string]any
	if err := json.Unmarshal(current, &doc); err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil {
		if _, err := BackupFile(path, now); err != nil {
			return err
		}
	}

	doc["env"] = env

	var payload bytes.Buffer
	encoder := json.NewEncoder(&payload)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(doc); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), ".settings-*.json")
	if err != nil {
		return err
	}

	tempPath := tempFile.Name()
	if _, err := tempFile.Write(payload.Bytes()); err != nil {
		tempFile.Close()
		_ = os.Remove(tempPath)
		return err
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	return nil
}
