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

	baseName := fmt.Sprintf("%s.%s", filepath.Base(path), now().UTC().Format("20060102T150405Z"))
	for attempt := 0; ; attempt++ {
		name := baseName + ".bak"
		if attempt > 0 {
			name = fmt.Sprintf("%s.%d.bak", baseName, attempt)
		}

		backupPath := filepath.Join(backupDir, name)
		file, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", err
		}

		if _, err := file.Write(content); err != nil {
			file.Close()
			_ = os.Remove(backupPath)
			return "", err
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(backupPath)
			return "", err
		}

		return backupPath, nil
	}
}
