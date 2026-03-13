package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func Load(path string) (ProfilesFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return ProfilesFile{}, err
	}

	var data ProfilesFile
	if err := json.Unmarshal(content, &data); err != nil {
		return ProfilesFile{}, err
	}

	if data.Version == 0 {
		data.Version = 1
	}

	if data.Profiles == nil {
		data.Profiles = map[string]Profile{}
	}

	for name, profile := range data.Profiles {
		if err := ValidateProfile(name, profile); err != nil {
			return ProfilesFile{}, err
		}
	}

	return data, nil
}

func Save(path string, data ProfilesFile) error {
	if data.Version == 0 {
		data.Version = 1
	}

	if data.Profiles == nil {
		data.Profiles = map[string]Profile{}
	}

	for name, profile := range data.Profiles {
		if err := ValidateProfile(name, profile); err != nil {
			return err
		}
	}

	var payload bytes.Buffer
	encoder := json.NewEncoder(&payload)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), ".profiles-*.json")
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

func SetCurrent(path, name string) error {
	data, err := Load(path)
	if err != nil {
		return err
	}

	if _, ok := data.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	data.Current = name
	return Save(path, data)
}

func Remove(path, name string) error {
	data, err := Load(path)
	if err != nil {
		return err
	}

	if name == data.Current {
		return fmt.Errorf("cannot remove the active profile")
	}

	if _, ok := data.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	delete(data.Profiles, name)
	return Save(path, data)
}

func Rename(path, oldName, newName string) error {
	data, err := Load(path)
	if err != nil {
		return err
	}

	if _, ok := data.Profiles[oldName]; !ok {
		return fmt.Errorf("profile %q not found", oldName)
	}

	if _, exists := data.Profiles[newName]; exists {
		return fmt.Errorf("profile %q already exists", newName)
	}

	data.Profiles[newName] = data.Profiles[oldName]
	delete(data.Profiles, oldName)

	if data.Current == oldName {
		data.Current = newName
	}

	return Save(path, data)
}
