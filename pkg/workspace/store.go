package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// baseDirOverride is set by tests to redirect all filesystem operations.
var baseDirOverride string

// BaseDir returns the path to ~/workspaces.
func BaseDir() (string, error) {
	if baseDirOverride != "" {
		return baseDirOverride, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, "workspaces"), nil
}

// WorkspaceDir returns the full path to a named workspace directory.
func WorkspaceDir(name string) (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, name), nil
}

// Exists reports whether a workspace with the given name exists on disk.
func Exists(name string) (bool, error) {
	dir, err := WorkspaceDir(name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// Create creates the workspace directory and writes an initial config.yaml.
// If template is non-empty, the named template's files are copied into the workspace.
// Returns the workspace directory path on success.
func Create(displayName string, template string) (string, error) {
	dirName := ToDirName(displayName)
	if err := ValidateDirName(dirName); err != nil {
		return "", fmt.Errorf("cannot derive valid workspace name from %q: %w", displayName, err)
	}
	ok, err := Exists(dirName)
	if err != nil {
		return "", err
	}
	if ok {
		return "", fmt.Errorf("workspace %q already exists", dirName)
	}
	if template != "" {
		ok, err := TemplateExists(template)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("template %q does not exist", template)
		}
	}
	dir, err := WorkspaceDir(dirName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating workspace directory: %w", err)
	}
	cfg := Config{Name: displayName, Template: template, Links: map[string]string{}}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0644); err != nil {
		return "", fmt.Errorf("writing config.yaml: %w", err)
	}
	if template != "" {
		if err := ApplyTemplate(template, dirName); err != nil {
			return "", fmt.Errorf("applying template: %w", err)
		}
	}
	return dir, nil
}

// ReadConfig loads and parses the config.yaml for the named workspace.
func ReadConfig(name string) (*Config, error) {
	dir, err := WorkspaceDir(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("reading config.yaml: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config.yaml: %w", err)
	}
	return &cfg, nil
}

// Delete removes the workspace directory. Returns an error if it does not exist.
func Delete(name string) error {
	ok, err := Exists(name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("workspace %q does not exist", name)
	}
	dir, err := WorkspaceDir(name)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// List returns the names of all workspaces (subdirectories excluding dot-entries).
func List() ([]string, error) {
	base, err := BaseDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading workspaces directory: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
