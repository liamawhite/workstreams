package types

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// configDirOverride is set by tests to redirect all filesystem operations.
var configDirOverride string

// ConfigDir returns ~/.workstreams (the config root, distinct from ~/workstreams).
func ConfigDir() (string, error) {
	if configDirOverride != "" {
		return configDirOverride, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".workstreams"), nil
}

// TypesDir returns ~/.workstreams/types.
func TypesDir() (string, error) {
	base, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "types"), nil
}

// TypeDir returns the full path to a named type directory.
func TypeDir(dirName string) (string, error) {
	base, err := TypesDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, dirName), nil
}

// Exists reports whether a type with the given dir name exists on disk.
func Exists(dirName string) (bool, error) {
	dir, err := TypeDir(dirName)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// List returns the dir names of all types (subdirectories excluding dot-entries).
func List() ([]string, error) {
	base, err := TypesDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading types directory: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ReadConfig loads and parses the config.yaml for the named type.
func ReadConfig(dirName string) (*TypeConfig, error) {
	dir, err := TypeDir(dirName)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("reading type config: %w", err)
	}
	var cfg TypeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing type config: %w", err)
	}
	return &cfg, nil
}

// Create creates a new type directory with config.yaml and boilerplate hook scripts.
// displayName is stored as the human-readable name; the dir name is derived from it.
func Create(displayName string) (string, error) {
	dirName := ToDirName(displayName)
	if err := ValidateDirName(dirName); err != nil {
		return "", fmt.Errorf("cannot derive valid type name from %q: %w", displayName, err)
	}
	ok, err := Exists(dirName)
	if err != nil {
		return "", err
	}
	if ok {
		return "", fmt.Errorf("type %q already exists", dirName)
	}
	dir, err := TypeDir(dirName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating type directory: %w", err)
	}
	cfg := TypeConfig{Name: displayName}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0644); err != nil {
		return "", fmt.Errorf("writing config.yaml: %w", err)
	}
	if err := os.WriteFile(OnInitScript(dirName), onInitBoilerplate, 0755); err != nil {
		return "", fmt.Errorf("writing onInit.sh: %w", err)
	}
	if err := os.WriteFile(OnInitAsyncScript(dirName), onInitAsyncBoilerplate, 0755); err != nil {
		return "", fmt.Errorf("writing onInitAsync.sh: %w", err)
	}
	if err := os.WriteFile(OnLoadScript(dirName), onLoadBoilerplate, 0755); err != nil {
		return "", fmt.Errorf("writing onLoad.sh: %w", err)
	}
	return dir, nil
}

// Delete removes a type directory. Returns an error if it does not exist.
func Delete(dirName string) error {
	ok, err := Exists(dirName)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("type %q does not exist", dirName)
	}
	dir, err := TypeDir(dirName)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}
