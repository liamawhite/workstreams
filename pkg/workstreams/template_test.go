package workstreams

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTemplate(t *testing.T) {
	useTemp(t)

	if err := CreateTemplate("my-tpl"); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	dir, _ := TemplateDir("my-tpl")
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("template dir not created: %v", err)
	}

	if err := CreateTemplate("my-tpl"); err == nil {
		t.Error("expected error creating duplicate template, got nil")
	}
}

func TestDeleteTemplate(t *testing.T) {
	useTemp(t)

	if err := CreateTemplate("my-tpl"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteTemplate("my-tpl"); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}
	dir, _ := TemplateDir("my-tpl")
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("template dir still exists after delete")
	}

	if err := DeleteTemplate("ghost"); err == nil {
		t.Error("expected error deleting non-existent template, got nil")
	}
}

func TestListTemplates(t *testing.T) {
	useTemp(t)

	names, err := ListTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}

	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := CreateTemplate(name); err != nil {
			t.Fatal(err)
		}
	}

	// dot-dirs should be excluded
	base, _ := TemplatesBaseDir()
	if err := os.MkdirAll(filepath.Join(base, ".hidden"), 0755); err != nil {
		t.Fatal(err)
	}

	names, err = ListTemplates()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 3 {
		t.Errorf("ListTemplates() returned %d names, want 3: %v", len(names), names)
	}
	want := map[string]bool{"alpha": true, "beta": true, "gamma": true}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected template name: %q", n)
		}
	}
}

func TestApplyTemplate(t *testing.T) {
	useTemp(t)

	if err := CreateTemplate("my-tpl"); err != nil {
		t.Fatal(err)
	}
	tmplDir, _ := TemplateDir("my-tpl")
	if err := os.MkdirAll(filepath.Join(tmplDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "AGENTS.md"), []byte("# Agents\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmplDir, "subdir", "notes.txt"), []byte("notes\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := Create("My Workspace", ""); err != nil {
		t.Fatal(err)
	}
	wsDir, _ := WorkstreamDir("my-workspace")

	// pre-existing workspace-only file that must survive
	if err := os.WriteFile(filepath.Join(wsDir, "my-own-file.txt"), []byte("mine\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ApplyTemplate("my-tpl", "my-workspace"); err != nil {
		t.Fatalf("ApplyTemplate: %v", err)
	}

	for _, rel := range []string{"AGENTS.md", "subdir/notes.txt"} {
		if _, err := os.Stat(filepath.Join(wsDir, rel)); err != nil {
			t.Errorf("template file %q not copied: %v", rel, err)
		}
	}

	if _, err := os.Stat(filepath.Join(wsDir, "my-own-file.txt")); err != nil {
		t.Errorf("workspace-only file was removed: %v", err)
	}
}

func TestApplyTemplateOverwrites(t *testing.T) {
	useTemp(t)

	if err := CreateTemplate("my-tpl"); err != nil {
		t.Fatal(err)
	}
	tmplDir, _ := TemplateDir("my-tpl")
	if err := os.WriteFile(filepath.Join(tmplDir, "AGENTS.md"), []byte("v1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := Create("My Workspace", "my-tpl"); err != nil {
		t.Fatal(err)
	}

	// update template and re-apply
	if err := os.WriteFile(filepath.Join(tmplDir, "AGENTS.md"), []byte("v2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyTemplate("my-tpl", "my-workspace"); err != nil {
		t.Fatalf("ApplyTemplate: %v", err)
	}

	wsDir, _ := WorkstreamDir("my-workspace")
	content, err := os.ReadFile(filepath.Join(wsDir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "v2\n" {
		t.Errorf("AGENTS.md after sync = %q, want %q", string(content), "v2\n")
	}
}

func TestCreateWithTemplate(t *testing.T) {
	useTemp(t)

	if err := CreateTemplate("my-tpl"); err != nil {
		t.Fatal(err)
	}
	tmplDir, _ := TemplateDir("my-tpl")
	if err := os.WriteFile(filepath.Join(tmplDir, "AGENTS.md"), []byte("# Agents\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := Create("My Workspace", "my-tpl"); err != nil {
		t.Fatalf("Create with template: %v", err)
	}

	wsDir, _ := WorkstreamDir("my-workspace")
	if _, err := os.Stat(filepath.Join(wsDir, "AGENTS.md")); err != nil {
		t.Errorf("template file not found in workspace: %v", err)
	}

	cfg, err := ReadConfig("my-workspace")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Template != "my-tpl" {
		t.Errorf("config.Template = %q, want %q", cfg.Template, "my-tpl")
	}
}

func TestCreateWithMissingTemplate(t *testing.T) {
	useTemp(t)
	if _, err := Create("My Workspace", "nonexistent"); err == nil {
		t.Error("expected error creating workspace with missing template, got nil")
	}
}
