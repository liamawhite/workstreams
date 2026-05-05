package workspace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TemplatesBaseDir returns the path to ~/workspaces/.templates.
func TemplatesBaseDir() (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, ".templates"), nil
}

// TemplateDir returns the full path to a named template directory.
func TemplateDir(name string) (string, error) {
	base, err := TemplatesBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, name), nil
}

// TemplateExists reports whether a template with the given name exists on disk.
func TemplateExists(name string) (bool, error) {
	dir, err := TemplateDir(name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// ListTemplates returns the names of all templates.
func ListTemplates() ([]string, error) {
	base, err := TemplatesBaseDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading templates directory: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// CreateTemplate creates a new empty template directory.
func CreateTemplate(name string) error {
	ok, err := TemplateExists(name)
	if err != nil {
		return err
	}
	if ok {
		return fmt.Errorf("template %q already exists", name)
	}
	dir, err := TemplateDir(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating template directory: %w", err)
	}
	return nil
}

// DeleteTemplate removes a template directory.
func DeleteTemplate(name string) error {
	ok, err := TemplateExists(name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("template %q does not exist", name)
	}
	dir, err := TemplateDir(name)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}

// ApplyTemplate copies all files from the named template into the workspace,
// overwriting matching files while leaving workspace-only files untouched.
func ApplyTemplate(templateName, workspaceName string) error {
	tmplDir, err := TemplateDir(templateName)
	if err != nil {
		return err
	}
	wsDir, err := WorkspaceDir(workspaceName)
	if err != nil {
		return err
	}
	return filepath.WalkDir(tmplDir, func(src string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(tmplDir, src)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dst := filepath.Join(wsDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		return copyFile(src, dst)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
