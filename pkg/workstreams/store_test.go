package workstreams

import (
	"os"
	"path/filepath"
	"testing"
)

func useTemp(t *testing.T) {
	t.Helper()
	baseDirOverride = t.TempDir()
	t.Cleanup(func() { baseDirOverride = "" })
}

func TestExists(t *testing.T) {
	useTemp(t)

	if ok, err := Exists("missing"); err != nil || ok {
		t.Fatalf("Exists(missing) = %v, %v; want false, nil", ok, err)
	}

	if _, err := Create("My Project", ""); err != nil {
		t.Fatal(err)
	}

	if ok, err := Exists("my-project"); err != nil || !ok {
		t.Fatalf("Exists(my-project) = %v, %v; want true, nil", ok, err)
	}
}

func TestCreate(t *testing.T) {
	tests := []struct {
		displayName string
		wantDir     string
		wantName    string
		wantErr     bool
	}{
		{"My Project", "my-project", "My Project", false},
		{"hello world", "hello-world", "hello world", false},
		{"Simple", "simple", "Simple", false},
		{"!!!", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.displayName, func(t *testing.T) {
			useTemp(t)
			dir, err := Create(tt.displayName, "")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Create(%q) error = %v, wantErr %v", tt.displayName, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			base, _ := BaseDir()
			wantPath := filepath.Join(base, tt.wantDir)
			if dir != wantPath {
				t.Errorf("returned path = %q, want %q", dir, wantPath)
			}
			if _, err := os.Stat(wantPath); err != nil {
				t.Errorf("workspace directory not created: %v", err)
			}
			cfg, err := ReadConfig(tt.wantDir)
			if err != nil {
				t.Fatalf("ReadConfig: %v", err)
			}
			if cfg.Name != tt.wantName {
				t.Errorf("config.Name = %q, want %q", cfg.Name, tt.wantName)
			}
			if cfg.Links == nil {
				t.Error("config.Links is nil, want empty map")
			}
		})
	}
}

func TestCreateDuplicate(t *testing.T) {
	useTemp(t)
	if _, err := Create("My Project", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := Create("My Project", ""); err == nil {
		t.Fatal("expected error creating duplicate workspace, got nil")
	}
}

func TestDelete(t *testing.T) {
	useTemp(t)
	if _, err := Create("My Project", ""); err != nil {
		t.Fatal(err)
	}
	if err := Delete("my-project"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ok, _ := Exists("my-project"); ok {
		t.Error("workspace still exists after Delete")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	useTemp(t)
	if err := Delete("ghost"); err == nil {
		t.Error("expected error deleting non-existent workspace, got nil")
	}
}

func TestList(t *testing.T) {
	useTemp(t)

	names, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}

	for _, display := range []string{"Alpha Project", "Beta Project", "Gamma Project"} {
		if _, err := Create(display, ""); err != nil {
			t.Fatal(err)
		}
	}

	// Create a dot-entry that should be excluded.
	base, _ := BaseDir()
	if err := os.MkdirAll(filepath.Join(base, ".hidden"), 0755); err != nil {
		t.Fatal(err)
	}

	names, err = List()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 3 {
		t.Errorf("List() returned %d names, want 3: %v", len(names), names)
	}
	want := map[string]bool{"alpha-project": true, "beta-project": true, "gamma-project": true}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected name in list: %q", n)
		}
	}
}

func TestReadConfig(t *testing.T) {
	useTemp(t)
	if _, err := Create("Hello World", ""); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadConfig("hello-world")
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if cfg.Name != "Hello World" {
		t.Errorf("Name = %q, want %q", cfg.Name, "Hello World")
	}
	if cfg.Links == nil {
		t.Error("Links is nil")
	}
}
