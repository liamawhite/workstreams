package workstreams

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"

	"github.com/liamawhite/workstreams/pkg/types"
)

// baseDirOverride is set by tests to redirect all filesystem operations.
var baseDirOverride string

// BaseDir returns the path to ~/workstreams.
func BaseDir() (string, error) {
	if baseDirOverride != "" {
		return baseDirOverride, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, "workstreams"), nil
}

// WorkstreamDir returns the full path to a named workstream directory.
func WorkstreamDir(name string) (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, name), nil
}

// Exists reports whether a workstream with the given name exists on disk.
func Exists(name string) (bool, error) {
	dir, err := WorkstreamDir(name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// Create creates the workstream directory and writes an initial config.yaml.
// If typeName is non-empty it must match an existing type; onInit.sh is run if present.
// Returns the workstream directory path on success.
func Create(displayName, typeName string) (string, error) {
	dirName := ToDirName(displayName)
	if err := ValidateDirName(dirName); err != nil {
		return "", fmt.Errorf("cannot derive valid workstream name from %q: %w", displayName, err)
	}
	if typeName != "" {
		ok, err := types.Exists(typeName)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("type %q does not exist", typeName)
		}
	}
	ok, err := Exists(dirName)
	if err != nil {
		return "", err
	}
	if ok {
		return "", fmt.Errorf("workstream %q already exists", dirName)
	}
	dir, err := WorkstreamDir(dirName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating workstream directory: %w", err)
	}
	cfg := Config{Name: displayName, Type: typeName}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0644); err != nil {
		return "", fmt.Errorf("writing config.yaml: %w", err)
	}
	if typeName != "" {
		env := append(os.Environ(),
			"WORKSTREAM_DIR="+dir,
			"WORKSTREAM_NAME="+displayName,
			"WORKSTREAM_TYPE="+typeName,
		)

		// Sync phase: blocks until complete, streams output to the terminal.
		syncScript := types.OnInitScript(typeName)
		if _, err := os.Stat(syncScript); err == nil {
			cmd := exec.Command("bash", syncScript)
			cmd.Dir = dir
			cmd.Env = env
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run() //nolint:errcheck
		}

		// Async phase: detached process that outlives the workstreams binary.
		// stdout/stderr are written to .async-init.log inside the workstream dir.
		asyncScript := types.OnInitAsyncScript(typeName)
		if _, err := os.Stat(asyncScript); err == nil {
			logFile, err := os.Create(filepath.Join(dir, ".async-init.log"))
			if err == nil {
				cmd := exec.Command("bash", asyncScript)
				cmd.Dir = dir
				cmd.Env = env
				cmd.Stdout = logFile
				cmd.Stderr = logFile
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				if cmd.Start() == nil {
					go func() {
						cmd.Wait()   //nolint:errcheck
						logFile.Close()
					}()
				} else {
					logFile.Close()
				}
			}
		}
	}
	return dir, nil
}

// ReadConfig loads and parses the config.yaml for the named workstream.
func ReadConfig(name string) (*Config, error) {
	dir, err := WorkstreamDir(name)
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

// Delete removes the workstream directory. Returns an error if it does not exist.
func Delete(name string) error {
	ok, err := Exists(name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("workstream %q does not exist", name)
	}
	dir, err := WorkstreamDir(name)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// DeleteAsync removes the workstream directory using a detached rm process that
// outlives the caller. The rm process runs in its own process group (Setpgid)
// so it is not killed if the parent process exits.
func DeleteAsync(name string) error {
	ok, err := Exists(name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("workstream %q does not exist", name)
	}
	dir, err := WorkstreamDir(name)
	if err != nil {
		return err
	}
	cmd := exec.Command("rm", "-rf", dir)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}

// List returns the names of all workstreams (subdirectories excluding dot-entries).
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
		return nil, fmt.Errorf("reading workstreams directory: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ListByType groups all workstream dir names by their type.
// Workstreams with no type are placed under the "" key.
func ListByType() (map[string][]string, error) {
	names, err := List()
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	for _, name := range names {
		cfg, err := ReadConfig(name)
		if err != nil {
			result[""] = append(result[""], name)
			continue
		}
		result[cfg.Type] = append(result[cfg.Type], name)
	}
	return result, nil
}
