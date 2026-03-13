package importer

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cc-switch/internal/profile"
)

type Result struct {
	Imported map[string]profile.Profile
	Skipped  []string
}

func ImportDir(dir string, overwrite bool, existing map[string]struct{}) (Result, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Imported: map[string]profile.Profile{},
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".env" {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".env")
		if !overwrite {
			if _, exists := existing[name]; exists {
				result.Skipped = append(result.Skipped, name)
				continue
			}
		}

		profileData, err := parseLegacyEnvFile(filepath.Join(dir, entry.Name()))
		if err != nil || profile.ValidateProfile(name, profileData) != nil {
			result.Skipped = append(result.Skipped, name)
			continue
		}

		result.Imported[name] = profileData
	}

	sort.Strings(result.Skipped)
	return result, nil
}

func parseLegacyEnvFile(path string) (profile.Profile, error) {
	file, err := os.Open(path)
	if err != nil {
		return profile.Profile{}, err
	}
	defer file.Close()

	parsed := profile.Profile{
		Env: map[string]string{},
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#!") && parsed.Description == "" {
			parsed.Description = strings.TrimSpace(strings.TrimPrefix(line, "#"))
			continue
		}

		if !strings.HasPrefix(line, "export ") {
			continue
		}

		assignment := strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(assignment, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		if _, ok := profile.SupportedEnvKeys[key]; !ok {
			continue
		}

		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")
		parsed.Env[key] = value
	}

	if err := scanner.Err(); err != nil {
		return profile.Profile{}, err
	}

	return parsed, nil
}
