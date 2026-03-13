package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func BackupFile(path string, now func() time.Time) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	backupDir := filepath.Join(homeDir, ".claude", "cc-switch", "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s.%s.bak", filepath.Base(path), now().UTC().Format("20060102T150405Z"))
	backupPath := filepath.Join(backupDir, name)
	if err := os.WriteFile(backupPath, content, 0o644); err != nil {
		return "", err
	}

	return backupPath, nil
}
