package types

import (
	"os"
	"path/filepath"
	"testing"
)

func useTemp(t *testing.T) {
	t.Helper()
	configDirOverride = t.TempDir()
	t.Cleanup(func() { configDirOverride = "" })
}

func TestExists(t *testing.T) {
	useTemp(t)

	if ok, err := Exists("missing"); err != nil || ok {
		t.Fatalf("Exists(missing) = %v, %v; want false, nil", ok, err)
	}

	if _, err := Create("My Type"); err != nil {
		t.Fatal(err)
	}

	if ok, err := Exists("my-type"); err != nil || !ok {
		t.Fatalf("Exists(my-type) = %v, %v; want true, nil", ok, err)
	}
}

func TestCreate(t *testing.T) {
	tests := []struct {
		displayName string
		wantDir     string
		wantErr     bool
	}{
		{"My Type", "my-type", false},
		{"golang", "golang", false},
		{"Go Lang", "go-lang", false},
		{"!!!", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.displayName, func(t *testing.T) {
			useTemp(t)
			dir, err := Create(tt.displayName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Create(%q) error = %v, wantErr %v", tt.displayName, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			base, _ := TypesDir()
			wantPath := filepath.Join(base, tt.wantDir)
			if dir != wantPath {
				t.Errorf("returned path = %q, want %q", dir, wantPath)
			}
			if _, err := os.Stat(wantPath); err != nil {
				t.Errorf("type directory not created: %v", err)
			}
			// config.yaml
			cfg, err := ReadConfig(tt.wantDir)
			if err != nil {
				t.Fatalf("ReadConfig: %v", err)
			}
			if cfg.Name != tt.displayName {
				t.Errorf("config.Name = %q, want %q", cfg.Name, tt.displayName)
			}
			// hook scripts
			for _, script := range []string{OnInitScript(tt.wantDir), OnInitAsyncScript(tt.wantDir), OnLoadScript(tt.wantDir)} {
				info, err := os.Stat(script)
				if err != nil {
					t.Errorf("hook script %q not created: %v", script, err)
					continue
				}
				if info.Mode()&0111 == 0 {
					t.Errorf("hook script %q is not executable", script)
				}
			}
		})
	}
}

func TestCreateDuplicate(t *testing.T) {
	useTemp(t)
	if _, err := Create("My Type"); err != nil {
		t.Fatal(err)
	}
	if _, err := Create("My Type"); err == nil {
		t.Fatal("expected error creating duplicate type, got nil")
	}
}

func TestDelete(t *testing.T) {
	useTemp(t)
	if _, err := Create("My Type"); err != nil {
		t.Fatal(err)
	}
	if err := Delete("my-type"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ok, _ := Exists("my-type"); ok {
		t.Error("type still exists after Delete")
	}
}

func TestDeleteNonExistent(t *testing.T) {
	useTemp(t)
	if err := Delete("ghost"); err == nil {
		t.Error("expected error deleting non-existent type, got nil")
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

	for _, display := range []string{"Alpha Type", "Beta Type", "Gamma Type"} {
		if _, err := Create(display); err != nil {
			t.Fatal(err)
		}
	}

	// Dot-entry should be excluded.
	base, _ := TypesDir()
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
	want := map[string]bool{"alpha-type": true, "beta-type": true, "gamma-type": true}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected name in list: %q", n)
		}
	}
}

func TestReadConfig(t *testing.T) {
	useTemp(t)
	if _, err := Create("Go Lang"); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadConfig("go-lang")
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if cfg.Name != "Go Lang" {
		t.Errorf("Name = %q, want %q", cfg.Name, "Go Lang")
	}
}

func TestOnScriptPaths(t *testing.T) {
	useTemp(t)
	if _, err := Create("My Type"); err != nil {
		t.Fatal(err)
	}
	base, _ := TypesDir()
	cases := []struct {
		got  string
		want string
	}{
		{OnInitScript("my-type"), filepath.Join(base, "my-type", "onInit.sh")},
		{OnInitAsyncScript("my-type"), filepath.Join(base, "my-type", "onInitAsync.sh")},
		{OnLoadScript("my-type"), filepath.Join(base, "my-type", "onLoad.sh")},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
